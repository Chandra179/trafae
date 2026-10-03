package books

import (
	"context"
	"testing"
	"time"
)

func TestSearchServesIdenticalRequestsFromCache(t *testing.T) {
	t.Parallel()

	provider := &stubProvider{capability: ProviderCapability{ID: "gutendex", Filters: []string{FilterTopics, FilterGenre}}, books: []Book{discoveryBook("gutendex", "1", 100)}}
	service := NewDependencies(&DependenciesConfig{Providers: []Provider{provider}, SearchCacheTTL: time.Minute})

	first, err := service.Search(context.Background(), SearchRequest{Topics: []string{"history"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Search(context.Background(), SearchRequest{Topics: []string{"history"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 1 {
		t.Fatalf("provider called %d times, want 1 (second request must be cached)", provider.calls)
	}
	if len(second.Results) != len(first.Results) {
		t.Fatalf("cached result count = %d, want %d", len(second.Results), len(first.Results))
	}

	// Topic order and formatting must not split the cache.
	if _, err := service.Search(context.Background(), SearchRequest{Topics: []string{" HISTORY "}, Limit: 10}); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 1 {
		t.Fatalf("provider called %d times after reordered request, want 1", provider.calls)
	}

	// A different request must miss.
	if _, err := service.Search(context.Background(), SearchRequest{Topics: []string{"biology"}, Limit: 10}); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 2 {
		t.Fatalf("provider called %d times after a different topic, want 2", provider.calls)
	}
}

func TestSearchDoesNotCacheTotalProviderFailure(t *testing.T) {
	t.Parallel()

	provider := &stubProvider{capability: ProviderCapability{ID: "gutendex", Filters: []string{FilterTopics, FilterGenre}}, err: context.DeadlineExceeded}
	service := NewDependencies(&DependenciesConfig{Providers: []Provider{provider}, SearchCacheTTL: time.Minute})

	for i := 0; i < 2; i++ {
		if _, err := service.Search(context.Background(), SearchRequest{Topics: []string{"history"}}); err == nil {
			t.Fatalf("request %d: expected the all-providers-failed error", i+1)
		}
	}
	if provider.calls != 2 {
		t.Fatalf("provider called %d times, want 2 (failed searches must not be cached)", provider.calls)
	}
}

func TestSearchCachesDegradedResponses(t *testing.T) {
	t.Parallel()

	// On networks where a provider is blocked, every response is degraded;
	// the cache must still engage or every request re-fans out upstream.
	healthy := &stubProvider{capability: ProviderCapability{ID: "open_library", Filters: []string{FilterTopics, FilterGenre}}, books: []Book{discoveryBook("open_library", "OL1W", 50)}}
	blocked := &stubProvider{capability: ProviderCapability{ID: "doab", Filters: []string{FilterTopics, FilterGenre}}, err: context.DeadlineExceeded}
	service := NewDependencies(&DependenciesConfig{Providers: []Provider{healthy, blocked}, SearchCacheTTL: time.Minute})
	request := SearchRequest{Topics: []string{"history"}}

	first, err := service.Search(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Search(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if healthy.calls != 1 || blocked.calls != 1 {
		t.Fatalf("provider calls = (healthy %d, blocked %d), want (1, 1) — second request must be served from cache", healthy.calls, blocked.calls)
	}
	if len(second.Results) != len(first.Results) {
		t.Fatalf("cached result count = %d, want %d", len(second.Results), len(first.Results))
	}
	if second.Providers[0].Status != "error" || second.Providers[1].Status != "ok" {
		t.Fatalf("cached response must report the statuses observed when it was built, got %#v", second.Providers)
	}
}

func TestSearchCacheIsDisabledWithoutTTL(t *testing.T) {
	t.Parallel()

	provider := &stubProvider{capability: ProviderCapability{ID: "gutendex", Filters: []string{FilterTopics, FilterGenre}}, books: []Book{discoveryBook("gutendex", "1", 100)}}
	service := NewDependencies(&DependenciesConfig{Providers: []Provider{provider}})
	request := SearchRequest{Topics: []string{"history"}}
	for i := 0; i < 2; i++ {
		if _, err := service.Search(context.Background(), request); err != nil {
			t.Fatal(err)
		}
	}
	if provider.calls != 2 {
		t.Fatalf("provider called %d times, want 2 (cache must be disabled at TTL zero)", provider.calls)
	}
}

func TestSearchRefetchesWhenPoolNotDeepEnough(t *testing.T) {
	t.Parallel()

	provider := &stubProvider{capability: ProviderCapability{ID: "gutendex", Filters: []string{FilterTopics, FilterGenre}}, books: []Book{discoveryBook("gutendex", "1", 100)}}
	service := NewDependencies(&DependenciesConfig{Providers: []Provider{provider}, SearchCacheTTL: time.Minute})
	limit := 2
	page := func(n int) SearchRequest {
		return SearchRequest{Topics: []string{"history"}, Limit: limit, Page: n}
	}

	// Page 1 fetches (1+lookahead)*limit depth, so the next few pages hit.
	if _, err := service.Search(context.Background(), page(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Search(context.Background(), page(2)); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 1 {
		t.Fatalf("page 2: provider called %d times, want 1", provider.calls)
	}

	// A page beyond the cached depth re-fetches with fresh lookahead. Page 3
	// (needs 6 ≤ depth 10) still hits; page 6 needs 12 > 10, so it misses.
	if _, err := service.Search(context.Background(), page(3)); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 1 {
		t.Fatalf("page 3: provider called %d times, want 1", provider.calls)
	}
	if _, err := service.Search(context.Background(), page(6)); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 2 {
		t.Fatalf("page 6: provider called %d times, want 2", provider.calls)
	}
	last := provider.requests[len(provider.requests)-1]
	if last.Limit != (6+poolLookaheadPages)*limit {
		t.Fatalf("deep fetch limit = %d, want %d", last.Limit, (6+poolLookaheadPages)*limit)
	}

	// Going back must slice the new, deeper pool without another fetch.
	if _, err := service.Search(context.Background(), page(1)); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 2 {
		t.Fatalf("back to page 1: provider called %d times, want 2", provider.calls)
	}
}
