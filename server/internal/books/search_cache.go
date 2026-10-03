package books

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const searchCacheMaxEntries = 512

// searchCache remembers fully successful fused responses for a short window.
// Every search fans out to the providers, which rate limit per client IP, so
// serving identical requests from memory keeps traffic bursts from turning
// into upstream 403s and timeouts. The cache is deliberately tiny: a fixed
// entry cap, TTL expiry, and no invalidation.
type searchCache struct {
	mu      sync.Mutex
	entries map[string]searchCacheEntry
	ttl     time.Duration
}

type searchCacheEntry struct {
	response   SearchResponse
	insertedAt time.Time
	expiresAt  time.Time
}

func newSearchCache(ttl time.Duration) *searchCache {
	return &searchCache{
		entries: make(map[string]searchCacheEntry, searchCacheMaxEntries),
		ttl:     ttl,
	}
}

// get returns the cached response for the normalized request when one is
// fresh. Callers must treat the response as read-only: it is shared between
// requests and marshaled concurrently.
func (c *searchCache) get(request SearchRequest, now time.Time) (SearchResponse, bool) {
	if c == nil || c.ttl <= 0 {
		return SearchResponse{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[cacheKey(request)]
	if !ok || !now.Before(entry.expiresAt) {
		return SearchResponse{}, false
	}
	return entry.response, true
}

// put stores a response for the normalized request. Responses containing an
// errored provider are never cached: they are the degradation path, and the
// next identical request should retry the failed provider immediately.
func (c *searchCache) put(request SearchRequest, response SearchResponse, now time.Time) {
	if c == nil || c.ttl <= 0 || hasErrorStatus(response) {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= searchCacheMaxEntries {
		c.evictLocked(now)
	}
	c.entries[cacheKey(request)] = searchCacheEntry{
		response:   response,
		insertedAt: now,
		expiresAt:  now.Add(c.ttl),
	}
}

func (c *searchCache) evictLocked(now time.Time) {
	for key, entry := range c.entries {
		if !now.Before(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
	if len(c.entries) < searchCacheMaxEntries {
		return
	}
	var oldestKey string
	var oldest time.Time
	for key, entry := range c.entries {
		if oldest.IsZero() || entry.insertedAt.Before(oldest) {
			oldestKey, oldest = key, entry.insertedAt
		}
	}
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

func hasErrorStatus(response SearchResponse) bool {
	for _, status := range response.Providers {
		if status.Status == "error" {
			return true
		}
	}
	return false
}

// cacheKey canonically serializes the normalized request so topic order,
// provider order, and formatting differences do not split the cache.
func cacheKey(request SearchRequest) string {
	topics := sortedLowered(request.Topics)
	providers := sortedLowered(request.Providers)
	return fmt.Sprintf("topics=%s|genre=%s|providers=%s|language=%s|min_year=%s|max_year=%s|min_popularity=%s|min_rating=%s|limit=%d|page=%d",
		strings.Join(topics, ","), strings.ToLower(strings.TrimSpace(request.Genre)), strings.Join(providers, ","),
		strings.ToLower(strings.TrimSpace(request.Language)),
		orNone(request.MinYear), orNone(request.MaxYear),
		orNoneFloat(request.MinPopularity), orNoneFloat(request.MinRating),
		request.Limit, request.Page)
}

func sortedLowered(values []string) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = strings.ToLower(strings.TrimSpace(value))
	}
	sort.Strings(result)
	return result
}

// orNone renders a nilable int by value: fmt's %v of a pointer would print an
// address, making every request a distinct cache key.
func orNone(value *int) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *value)
}

func orNoneFloat(value *float64) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%g", *value)
}
