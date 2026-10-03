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
	entries map[string]searchCacheEntry
	mu      sync.Mutex
	ttl     time.Duration
}

// searchCacheEntry keeps the expiry as raw nanoseconds: with a uniform TTL,
// expiry order is insertion order, so eviction needs no second timestamp.
type searchCacheEntry struct {
	response  SearchResponse
	expiresAt int64
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
	if !ok || now.UnixNano() >= entry.expiresAt {
		return SearchResponse{}, false
	}
	return entry.response, true
}

// put stores a response for the normalized request. Search only reaches this
// point when at least one provider succeeded, so degraded responses (some
// providers errored) are cached too: on networks where a provider is blocked,
// the strict alternative would never cache at all and every request would
// re-fan-out to the healthy providers. Provider statuses inside a cached
// response describe the fetch that built it and are frozen until the entry
// expires; a fully failed search returns an error before it gets here and is
// never cached.
func (c *searchCache) put(request SearchRequest, response SearchResponse, now time.Time) {
	if c == nil || c.ttl <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= searchCacheMaxEntries {
		c.evictLocked(now)
	}
	c.entries[cacheKey(request)] = searchCacheEntry{
		response:  response,
		expiresAt: now.Add(c.ttl).UnixNano(),
	}
}

func (c *searchCache) evictLocked(now time.Time) {
	for key, entry := range c.entries {
		if now.UnixNano() >= entry.expiresAt {
			delete(c.entries, key)
		}
	}
	if len(c.entries) < searchCacheMaxEntries {
		return
	}
	var oldestKey string
	var oldestExpiry int64
	for key, entry := range c.entries {
		if oldestKey == "" || entry.expiresAt < oldestExpiry {
			oldestKey, oldestExpiry = key, entry.expiresAt
		}
	}
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
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
