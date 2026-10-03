package books

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubProvider struct {
	capability ProviderCapability
	books      []Book
	err        error
	calls      int
	requests   []SearchRequest
}

func (p *stubProvider) Capabilities() ProviderCapability { return p.capability }
func (p *stubProvider) Search(_ context.Context, request SearchRequest) ([]Book, error) {
	p.calls++
	p.requests = append(p.requests, request)
	return p.books, p.err
}

func discoveryBook(provider, id string, popularity float64) Book {
	book := Book{
		ID: id, Title: "Biology for Everyone", Authors: []string{"A. Author"},
		Subjects: []string{"Biology", "Non-Fiction"}, ISBNs: []string{"978-0-123456-78-9"},
		Source:     BookSource{Provider: provider, ID: id, URL: "https://example.test/" + id},
		Popularity: &Metric{Value: popularity, Metric: "downloads"}, Rating: &Rating{Value: 4.5, Scale: 5, Count: 10},
	}
	return book
}

func TestSearchFusesDuplicateBooksAndRetainsPerSourceMetrics(t *testing.T) {
	first := &stubProvider{capability: ProviderCapability{ID: "gutendex", Filters: []string{FilterTopics, FilterGenre}}, books: []Book{discoveryBook("gutendex", "1", 100)}}
	second := &stubProvider{capability: ProviderCapability{ID: "open_library", Filters: []string{FilterTopics, FilterGenre}}, books: []Book{discoveryBook("open_library", "OL1W", 50)}}
	service := NewDependencies(&DependenciesConfig{Providers: []Provider{first, second}})
	response, err := service.Search(context.Background(), SearchRequest{Topics: []string{"biology"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Results) != 1 {
		t.Fatalf("got %d results, want one deduplicated book", len(response.Results))
	}
	result := response.Results[0]
	if len(result.Sources) != 2 {
		t.Fatalf("got %d sources, want 2", len(result.Sources))
	}
	wantScore := 2.0 / 61.0
	if result.RRFScore < wantScore-1e-12 || result.RRFScore > wantScore+1e-12 {
		t.Fatalf("RRF score = %f, want %f", result.RRFScore, wantScore)
	}
	if result.Sources[0].Popularity == nil || result.Sources[1].Popularity == nil {
		t.Fatal("source popularity metrics were not preserved")
	}
}

func TestSearchSkipsProviderWithoutRequestedFilterAndUsesRemainingProvider(t *testing.T) {
	supported := &stubProvider{capability: ProviderCapability{ID: "open_library", Filters: []string{FilterTopics, FilterGenre, FilterRating}}, books: []Book{discoveryBook("open_library", "OL1W", 50)}}
	unsupported := &stubProvider{capability: ProviderCapability{ID: "doab", Filters: []string{FilterTopics, FilterGenre}}, books: []Book{discoveryBook("doab", "DOAB1", 1)}}
	service := NewDependencies(&DependenciesConfig{Providers: []Provider{unsupported, supported}})
	minRating := 4.0
	response, err := service.Search(context.Background(), SearchRequest{Topics: []string{"biology"}, MinRating: &minRating})
	if err != nil {
		t.Fatal(err)
	}
	if unsupported.calls != 0 {
		t.Fatal("provider lacking requested rating filter was called")
	}
	if len(response.Results) != 1 {
		t.Fatalf("got %d results, want 1", len(response.Results))
	}
	if response.Providers[0].Status != "skipped" {
		t.Fatalf("unsupported provider status = %q, want skipped", response.Providers[0].Status)
	}
	if !strings.Contains(response.Providers[0].Reason, FilterRating) {
		t.Fatalf("skip reason %q does not identify rating", response.Providers[0].Reason)
	}
}

func TestSearchReturnsPartialResultsWhenProviderFails(t *testing.T) {
	good := &stubProvider{capability: ProviderCapability{ID: "gutendex", Filters: []string{FilterTopics, FilterGenre}}, books: []Book{discoveryBook("gutendex", "1", 100)}}
	bad := &stubProvider{capability: ProviderCapability{ID: "doab", Filters: []string{FilterTopics, FilterGenre}}, err: errors.New("upstream unavailable")}
	service := NewDependencies(&DependenciesConfig{Providers: []Provider{good, bad}})
	response, err := service.Search(context.Background(), SearchRequest{Topics: []string{"biology"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Results) != 1 || len(response.Providers) != 2 {
		t.Fatalf("response = %#v", response)
	}
}

func TestSearchMarksPartiallyFailedProviderAndKeepsItsResults(t *testing.T) {
	partial := &stubProvider{
		capability: ProviderCapability{ID: "gutendex", Filters: []string{FilterTopics, FilterGenre}},
		books:      []Book{discoveryBook("gutendex", "1", 100)},
		err:        errors.New("one topic request failed: gutendex.test returned HTTP 500"),
	}
	service := NewDependencies(&DependenciesConfig{Providers: []Provider{partial}})
	response, err := service.Search(context.Background(), SearchRequest{Topics: []string{"biology"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Results) != 1 {
		t.Fatalf("results = %d, want the partial provider's books kept", len(response.Results))
	}
	status := response.Providers[0]
	if status.Status != "partial" || status.Count != 1 {
		t.Fatalf("status = %#v, want partial with count 1", status)
	}
	if !strings.Contains(status.Reason, "failed") {
		t.Fatalf("reason = %q, want the underlying failure", status.Reason)
	}
}

func TestProviderStatusAlwaysSerializesCount(t *testing.T) {
	raw, err := json.Marshal(ProviderStatus{Provider: "gutendex", Status: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"count":0`) {
		t.Fatalf("provider status = %s, want count serialized even when zero", raw)
	}
}

func TestSearchReturnsErrorWhenNoProviderCanServeRequest(t *testing.T) {
	provider := &stubProvider{capability: ProviderCapability{ID: "doab", Filters: []string{FilterTopics, FilterGenre}}}
	service := NewDependencies(&DependenciesConfig{Providers: []Provider{provider}})
	minRating := 4.0
	response, err := service.Search(context.Background(), SearchRequest{Topics: []string{"biology"}, MinRating: &minRating})
	if err == nil {
		t.Fatal("Search() returned nil error when every provider was skipped")
	}
	if response.Providers[0].Status != "skipped" {
		t.Fatalf("status = %q, want skipped", response.Providers[0].Status)
	}
}

func TestSearchPaginatesFusedResults(t *testing.T) {
	catalog := make([]Book, 0, 5)
	for i := 0; i < 5; i++ {
		book := discoveryBook("gutendex", fmt.Sprintf("id-%d", i), 10)
		book.Title = fmt.Sprintf("Book %02d", i)
		book.ISBNs = []string{fmt.Sprintf("978000000000%d", i)}
		catalog = append(catalog, book)
	}
	provider := &stubProvider{capability: ProviderCapability{ID: "gutendex", Filters: []string{FilterTopics, FilterGenre}}, books: catalog}
	service := NewDependencies(&DependenciesConfig{Providers: []Provider{provider}})
	query := func(page int) SearchRequest {
		return SearchRequest{Topics: []string{"biology"}, Limit: 2, Page: page}
	}

	first, err := service.Search(context.Background(), query(1))
	if err != nil {
		t.Fatal(err)
	}
	if first.Page != 1 || len(first.Results) != 2 || !first.HasMore {
		t.Fatalf("page 1 = %#v", first)
	}
	if first.Results[0].Book.Title != "Book 00" || first.Results[1].Book.Title != "Book 01" {
		t.Fatalf("page 1 titles = %q, %q", first.Results[0].Book.Title, first.Results[1].Book.Title)
	}

	second, err := service.Search(context.Background(), query(2))
	if err != nil {
		t.Fatal(err)
	}
	if second.Page != 2 || len(second.Results) != 2 || !second.HasMore {
		t.Fatalf("page 2 = %#v", second)
	}
	if second.Results[0].Book.Title != "Book 02" || second.Results[1].Book.Title != "Book 03" {
		t.Fatalf("page 2 titles = %q, %q", second.Results[0].Book.Title, second.Results[1].Book.Title)
	}

	third, err := service.Search(context.Background(), query(3))
	if err != nil {
		t.Fatal(err)
	}
	if third.Page != 3 || len(third.Results) != 1 || third.HasMore {
		t.Fatalf("page 3 = %#v", third)
	}
	if third.Results[0].Book.Title != "Book 04" {
		t.Fatalf("page 3 title = %q", third.Results[0].Book.Title)
	}

	fourth, err := service.Search(context.Background(), query(4))
	if err != nil {
		t.Fatal(err)
	}
	if len(fourth.Results) != 0 || fourth.HasMore {
		t.Fatalf("page 4 = %#v", fourth)
	}
	last := provider.requests[len(provider.requests)-1]
	if last.Limit != 8 {
		t.Fatalf("provider fetch limit = %d, want 8 (page 4 * limit 2)", last.Limit)
	}
}

func TestSearchSeedsGenreAsTopicForGenreOnlyBrowse(t *testing.T) {
	provider := &stubProvider{capability: ProviderCapability{ID: "doab", Filters: []string{FilterTopics, FilterGenre}}, books: []Book{discoveryBook("doab", "1", 1)}}
	service := NewDependencies(&DependenciesConfig{Providers: []Provider{provider}})

	if _, err := service.Search(context.Background(), SearchRequest{Genre: "history"}); err != nil {
		t.Fatal(err)
	}
	if topics := provider.requests[0].Topics; len(topics) != 1 || topics[0] != "history" {
		t.Fatalf("provider topics = %v, want [history]", topics)
	}

	if _, err := service.Search(context.Background(), SearchRequest{}); err != nil {
		t.Fatal(err)
	}
	if topics := provider.requests[1].Topics; len(topics) != 0 {
		t.Fatalf("provider topics = %v, want none for the default genre", topics)
	}
}

func TestGinSearchAndCapabilitiesHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := &stubProvider{capability: ProviderCapability{ID: "gutendex", Name: "Gutendex", Filters: []string{FilterTopics, FilterGenre}}, books: []Book{discoveryBook("gutendex", "1", 100)}}
	service := NewDependencies(&DependenciesConfig{Providers: []Provider{provider}})
	router := gin.New()
	router.GET("/books/search", service.HandleSearch)
	router.GET("/books/providers", service.HandleProviders)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/books/search?topic=biology", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("search status = %d body=%s", response.Code, response.Body.String())
	}
	var searchResponse SearchResponse
	if err := json.Unmarshal(response.Body.Bytes(), &searchResponse); err != nil {
		t.Fatal(err)
	}
	if len(searchResponse.Results) != 1 {
		t.Fatalf("search results = %d, want 1", len(searchResponse.Results))
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/books/providers", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"gutendex"`) {
		t.Fatalf("capabilities response = %d %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/books/search?min_rating=8", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid search status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	for _, invalidPage := range []string{"page=-1", "page=21"} {
		response = httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/books/search?"+invalidPage, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d, want %d", invalidPage, response.Code, http.StatusBadRequest)
		}
	}

	longValue := strings.Repeat("a", MaxParamLength+1)
	for _, tc := range []struct{ name, query string }{
		{"negative limit", "limit=-1"},
		{"limit above max", "limit=51"},
		{"over-long topic", "topic=" + longValue},
		{"over-long genre", "genre=" + longValue},
		{"over-long language", "language=" + longValue},
	} {
		response = httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/books/search?"+tc.query, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want %d", tc.name, response.Code, http.StatusBadRequest)
		}
	}
}
