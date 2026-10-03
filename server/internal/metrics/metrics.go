// Package metrics aggregates the launch funnel counters in memory: how many
// searches ran, how many came from the cache, which providers served or
// failed, and how often readers opened a book's read link. Counts survive
// only for the lifetime of the process — enough to answer "is the fused list
// actually sending people to books" without collecting personal data.
package metrics

import (
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

const unknownProvider = "unknown"

// Metrics is safe for concurrent use by handlers and services.
type Metrics struct {
	searches       atomic.Int64
	searchesCached atomic.Int64
	eventsRejected atomic.Int64

	mu             sync.Mutex
	providerStatus map[string]map[string]*atomic.Int64
	providerClicks map[string]map[string]*atomic.Int64
}

func New() *Metrics {
	return &Metrics{
		providerStatus: make(map[string]map[string]*atomic.Int64),
		providerClicks: make(map[string]map[string]*atomic.Int64),
	}
}

// RecordSearch counts one handled search request, cached or not. All record
// methods tolerate a nil receiver so services can run un-instrumented.
func (m *Metrics) RecordSearch(cached bool) {
	if m == nil {
		return
	}
	if cached {
		m.searchesCached.Add(1)
		return
	}
	m.searches.Add(1)
}

// RecordProviderStatus counts one provider outcome as reported in a search
// response ("ok", "partial", "skipped", or "error").
func (m *Metrics) RecordProviderStatus(provider, status string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordLocked(m.providerStatus, provider, status)
}

// RecordAccessClick counts a reader opening a book's read link, per provider.
func (m *Metrics) RecordAccessClick(provider string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordLocked(m.providerClicks, provider, "clicks")
}

// RecordRejectedEvent counts an event beacon that failed validation.
func (m *Metrics) RecordRejectedEvent() {
	if m == nil {
		return
	}
	m.eventsRejected.Add(1)
}

func (m *Metrics) recordLocked(counters map[string]map[string]*atomic.Int64, provider, status string) {
	provider = normalizedCounterKey(provider)
	byStatus, ok := counters[provider]
	if !ok {
		byStatus = make(map[string]*atomic.Int64)
		counters[provider] = byStatus
	}
	counter, ok := byStatus[status]
	if !ok {
		counter = &atomic.Int64{}
		byStatus[status] = counter
	}
	counter.Add(1)
}

func normalizedCounterKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return unknownProvider
	}
	return value
}

// Snapshot renders the counters as a plain JSON map. Per-provider maps are
// sorted by the JSON encoder, so the output is stable for diffing.
func (m *Metrics) Snapshot() gin.H {
	if m == nil {
		return gin.H{
			"searches_total":        0,
			"searches_cached_total": 0,
			"provider_status":       map[string]map[string]int64{},
			"access_clicks":         map[string]map[string]int64{},
			"events_rejected_total": 0,
		}
	}
	m.mu.Lock()
	providerStatus := snapshotCounters(m.providerStatus)
	providerClicks := snapshotCounters(m.providerClicks)
	m.mu.Unlock()

	return gin.H{
		"searches_total":        m.searches.Load(),
		"searches_cached_total": m.searchesCached.Load(),
		"provider_status":       providerStatus,
		"access_clicks":         providerClicks,
		"events_rejected_total": m.eventsRejected.Load(),
	}
}

func snapshotCounters(counters map[string]map[string]*atomic.Int64) map[string]map[string]int64 {
	snapshot := make(map[string]map[string]int64, len(counters))
	for provider, statuses := range counters {
		counts := make(map[string]int64, len(statuses))
		for status, counter := range statuses {
			counts[status] = counter.Load()
		}
		snapshot[provider] = counts
	}
	return snapshot
}

// Handler serves the snapshot at GET /metrics.
func (m *Metrics) Handler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, m.Snapshot()) }
}

// event is the payload the frontend sends when a reader clicks a read link.
// The topic is accepted but not recorded: free-text topics would make the
// counters unbounded without adding decision value.
type event struct {
	Type     string `json:"type"`
	Provider string `json:"provider"`
	Topic    string `json:"topic"`
}

const maxEventFieldLength = 200

// EventHandler accepts the POST /books/events beacon. It always answers
// quickly and never fails a user flow: an invalid payload costs a rejected
// counter, not a client error worth retrying — except malformed JSON, which
// is answered 400 so broken clients are visible.
func (m *Metrics) EventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload event
		if err := c.ShouldBindJSON(&payload); err != nil {
			m.RecordRejectedEvent()
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event payload"})
			return
		}
		if payload.Type != "access_click" ||
			len(payload.Provider) > maxEventFieldLength || len(payload.Topic) > maxEventFieldLength {
			m.RecordRejectedEvent()
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported event"})
			return
		}
		m.RecordAccessClick(payload.Provider)
		c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})
	}
}
