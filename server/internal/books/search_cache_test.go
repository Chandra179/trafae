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

func TestSearchDoesNotCacheErroredProviders(t *testing.T) {
	t.Parallel()

	provider := &stubProvider{capability: ProviderCapability{ID: "gutendex", Filters: []string{FilterTopics, FilterGenre}}, err: context.DeadlineExceeded}
	service := NewDependencies(&DependenciesConfig{Providers: []Provider{provider}, SearchCacheTTL: time.Minute})

	for i := 0; i < 2; i++ {
		if _, err := service.Search(context.Background(), SearchRequest{Topics: []string{"history"}}); err == nil {
			t.Fatalf("request %d: expected the all-providers-failed error", i+1)
		}
	}
	if provider.calls != 2 {
		t.Fatalf("provider called %d times, want 2 (failed responses must not be cached)", provider.calls)
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
