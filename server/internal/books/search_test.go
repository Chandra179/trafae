package books

import (
	"context"
	"encoding/json"
	"errors"
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
}

func (p *stubProvider) Capabilities() ProviderCapability { return p.capability }
func (p *stubProvider) Search(context.Context, SearchRequest) ([]Book, error) {
	p.calls++
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
}
