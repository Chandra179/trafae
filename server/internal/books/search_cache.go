package books

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// searchCacheMaxEntries bounds memory: each entry holds a whole fused result
// pool, not just one page of it.
const searchCacheMaxEntries = 128

// poolLookaheadPages is how many pages beyond the requested one a cache-miss
// fetch covers. With the frontend's limit of 24, one fetch serves the clicked
// page plus the next four Next-clicks from memory; going deeper re-fetches.
const poolLookaheadPages = 4

// searchCache remembers fused result pools for a short window. Pagination is
// stateless "fetch deeper, then slice": every page request would otherwise
// re-fan-out to every provider with a deeper fetch, so the cache stores the
// whole sorted pool per query — page changes then slice it locally instead of
// hammering rate-limited upstreams again.
type searchCache struct {
	entries map[string]searchCacheEntry
	mu      sync.Mutex
	ttl     time.Duration
}

// searchCacheEntry keeps the expiry as raw nanoseconds: with a uniform TTL,
// expiry order is insertion order, so eviction needs no second timestamp.
// fetchLimit is the page×limit depth the providers were asked for; page
// requests within it slice the pool, deeper ones re-fetch.
type searchCacheEntry struct {
	pool       []SearchResult
	providers  []ProviderStatus
	fetchLimit int
	expiresAt  int64
}

func newSearchCache(ttl time.Duration) *searchCache {
	return &searchCache{
		entries: make(map[string]searchCacheEntry, searchCacheMaxEntries),
		ttl:     ttl,
	}
}

// get returns the cached pool for the query when it is fresh and deep enough
// to serve the requested page. Callers must treat the pool and statuses as
// read-only: they are shared between requests and marshaled concurrently.
func (c *searchCache) get(request SearchRequest, now time.Time) (searchCacheEntry, bool) {
	if c == nil || c.ttl <= 0 {
		return searchCacheEntry{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[poolKey(request)]
	if !ok || now.UnixNano() >= entry.expiresAt {
		return searchCacheEntry{}, false
	}
	if request.Page*request.Limit > entry.fetchLimit {
		return searchCacheEntry{}, false
	}
	return entry, true
}

// put stores the pool built for a query. Search only reaches this point when
// at least one provider succeeded, so degraded pools (some providers errored)
// are cached too: on networks where a provider is blocked, the strict
// alternative would never cache at all and every request would re-fan-out to
// the healthy providers. Statuses inside a cached entry describe the fetch
// that built it and are frozen until the entry expires; a fully failed search
// returns an error before it gets here and is never cached.
func (c *searchCache) put(request SearchRequest, pool []SearchResult, providers []ProviderStatus, fetchLimit int, now time.Time) {
	if c == nil || c.ttl <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= searchCacheMaxEntries {
		c.evictLocked(now)
	}
	c.entries[poolKey(request)] = searchCacheEntry{
		pool:       pool,
		providers:  providers,
		fetchLimit: fetchLimit,
		expiresAt:  now.Add(c.ttl).UnixNano(),
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

// poolKey canonically serializes the normalized request minus the page, so
// topic order, provider order, and formatting differences do not split the
// cache while page navigation shares one pool. fmt's %v of a pointer would
// print an address, so nilable values render by value.
func poolKey(request SearchRequest) string {
	topics := sortedLowered(request.Topics)
	providers := sortedLowered(request.Providers)
	return fmt.Sprintf("topics=%s|genre=%s|providers=%s|language=%s|min_year=%s|max_year=%s|min_popularity=%s|min_rating=%s|limit=%d",
		strings.Join(topics, ","), strings.ToLower(strings.TrimSpace(request.Genre)), strings.Join(providers, ","),
		strings.ToLower(strings.TrimSpace(request.Language)),
		orNone(request.MinYear), orNone(request.MaxYear),
		orNoneFloat(request.MinPopularity), orNoneFloat(request.MinRating),
		request.Limit)
}

func sortedLowered(values []string) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = strings.ToLower(strings.TrimSpace(value))
	}
	sort.Strings(result)
	return result
}

// orNone renders a nilable int by value.
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
