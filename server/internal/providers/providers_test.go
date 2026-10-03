package providers

import (
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/Chandra179/trafae/server/internal/books"
	"github.com/Chandra179/trafae/server/store"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func fakeClient(t *testing.T, handler func(*http.Request) string) *http.Client {
	t.Helper()
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body := handler(request)
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})}
}

func responseClient(handler func(*http.Request) *http.Response) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) { return handler(request), nil })}
}

func jsonResponse(request *http.Request, value string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(value)), Request: request}
}

func providerRequest() books.SearchRequest {
	return books.SearchRequest{Topics: []string{"biology"}, Genre: "non-fiction", Limit: 10}
}

func TestOpenLibrarySearchMapsWorkAndRating(t *testing.T) {
	client := fakeClient(t, func(r *http.Request) string {
		if r.URL.Path != "/search.json" || r.URL.Query().Get("subject") != "biology" {
			t.Errorf("unexpected request: %s", r.URL.String())
		}
		return `{"docs":[{"key":"/works/OL1W","title":"Biology","author_name":["Ada Author"],"first_publish_year":1910,"subject":["Biology","Non-Fiction"],"isbn":["9780123456789"],"ratings_average":4.4,"ratings_count":21,"cover_i":123}]}`
	})
	provider := newOpenLibraryProvider(client, "https://openlibrary.test", "")
	results, err := provider.Search(context.Background(), providerRequest())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Rating == nil || results[0].Rating.Count != 21 || results[0].Year == nil || *results[0].Year != 1910 {
		t.Fatalf("unexpected result: %#v", results)
	}
}

func TestGutendexSearchMapsTopicsAndDownloads(t *testing.T) {
	client := fakeClient(t, func(r *http.Request) string {
		if r.URL.Query().Get("topic") != "biology" {
			t.Errorf("topic = %q", r.URL.Query().Get("topic"))
		}
		return `{"next":null,"results":[{"id":42,"title":"Biology","authors":[{"name":"Ada Author"}],"subjects":["Biology","Non-Fiction"],"bookshelves":["Science"],"languages":["en"],"download_count":900,"formats":{"text/plain":"https://example.test/book.txt"}}]}`
	})
	provider := newGutendexProvider(client, "https://gutendex.test/books")
	results, err := provider.Search(context.Background(), providerRequest())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Popularity == nil || results[0].Popularity.Value != 900 || len(results[0].AccessURLs) != 1 {
		t.Fatalf("unexpected result: %#v", results)
	}
}

func TestGutendexSearchesEveryTopic(t *testing.T) {
	items := make([]string, 0, 30)
	for i := 0; i < 30; i++ {
		items = append(items, fmt.Sprintf(`{"id":%d,"title":"Book %d","authors":[{"name":"Ada Author"}]}`, i, i))
	}
	body := `{"next":null,"results":[` + strings.Join(items, ",") + `]}`
	var mu sync.Mutex
	var topics []string
	client := fakeClient(t, func(r *http.Request) string {
		mu.Lock()
		topics = append(topics, r.URL.Query().Get("topic"))
		mu.Unlock()
		return body
	})
	provider := newGutendexProvider(client, "https://gutendex.test/books")
	request := providerRequest()
	request.Topics = []string{"biology", "history"}
	results, err := provider.Search(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	collected := append([]string(nil), topics...)
	mu.Unlock()
	sort.Strings(collected)
	if len(collected) != 2 || collected[0] != "biology" || collected[1] != "history" {
		t.Fatalf("upstream topics = %v, want one request per topic", collected)
	}
	if len(results) != 30 {
		t.Fatalf("results = %d, want 30", len(results))
	}
}

func TestGutendexSearchesTopicsConcurrently(t *testing.T) {
	secondStarted := make(chan struct{})
	client := fakeClient(t, func(r *http.Request) string {
		if r.URL.Query().Get("topic") == "history" {
			close(secondStarted)
			return `{"next":null,"results":[]}`
		}
		select {
		case <-secondStarted:
		case <-time.After(2 * time.Second):
			t.Error("second topic did not start before the first finished; per-topic requests look sequential")
		}
		return `{"next":null,"results":[]}`
	})
	provider := newGutendexProvider(client, "https://gutendex.test/books")
	request := providerRequest()
	request.Topics = []string{"biology", "history"}
	if _, err := provider.Search(context.Background(), request); err != nil {
		t.Fatal(err)
	}
}

func TestGutendexKeepsPartialResultsWhenOneTopicFails(t *testing.T) {
	client := responseClient(func(request *http.Request) *http.Response {
		if request.URL.Query().Get("topic") == "history" {
			return &http.Response{StatusCode: http.StatusInternalServerError, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("{}")), Request: request}
		}
		return jsonResponse(request, `{"next":null,"results":[{"id":42,"title":"Biology","authors":[{"name":"Ada Author"}]}]}`)
	})
	provider := newGutendexProvider(client, "https://gutendex.test/books")
	request := providerRequest()
	request.Topics = []string{"biology", "history"}
	results, err := provider.Search(context.Background(), request)
	if err == nil {
		t.Fatal("Search() returned nil error when a topic request failed")
	}
	if len(results) != 1 || results[0].Title != "Biology" {
		t.Fatalf("results = %#v, want the successful topic's books alongside the error", results)
	}
}

func TestDOABSearchMapsMetadata(t *testing.T) {
	client := fakeClient(t, func(r *http.Request) string {
		if !strings.Contains(r.URL.Query().Get("query"), "biology") || r.URL.Query().Get("page") != "0" || r.URL.Query().Get("size") == "" {
			t.Errorf("unexpected query: %s", r.URL.String())
		}
		return `{"_embedded":{"searchResult":{"_embedded":{"objects":[{"_embedded":{"indexableObject":{"uuid":"uuid-1","handle":"20.500.12071/123","name":"Open Biology","metadata":{"dc.title":[{"value":"Open Biology"}],"dc.contributor.author":[{"value":"Ada Author"}],"dc.subject":[{"value":"Biology"}],"dc.date.issued":[{"value":"2020"}],"dc.language.iso":[{"value":"en"}],"dc.rights.uri":[{"value":"https://creativecommons.org/licenses/by/4.0/"}],"dc.identifier.uri":[{"value":"https://directory.doabooks.org/handle/20.500.12071/123"}]}}}}]}}}}`
	})
	provider := newDOABProvider(client, "https://doab.test/rest/api/discover/search/objects")
	results, err := provider.Search(context.Background(), providerRequest())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Title != "Open Biology" || results[0].License == "" || results[0].Year == nil {
		t.Fatalf("unexpected result: %#v", results)
	}
	if results[0].URL != "https://directory.doabooks.org/handle/20.500.12071/123" {
		t.Fatalf("unexpected URL: %s", results[0].URL)
	}
}

func TestInternetArchiveSearchMapsNativeMetrics(t *testing.T) {
	client := fakeClient(t, func(r *http.Request) string {
		if !strings.Contains(r.URL.Query().Get("q"), "biology") {
			t.Errorf("q = %q", r.URL.Query().Get("q"))
		}
		return `{"response":{"docs":[{"identifier":"ia-1","title":"Biology","creator":["Ada Author"],"year":"1990","subject":["Biology","Non-Fiction"],"language":["eng"],"downloads":120,"avg_rating":4.2,"num_reviews":9}]}}`
	})
	provider := newInternetArchiveProvider(client, "https://archive.test/advancedsearch.php")
	results, err := provider.Search(context.Background(), providerRequest())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Popularity == nil || results[0].Popularity.Value != 120 || results[0].Rating == nil {
		t.Fatalf("unexpected result: %#v", results)
	}
}

func TestLibraryOfCongressSearchMapsBookRecord(t *testing.T) {
	client := fakeClient(t, func(r *http.Request) string {
		if r.URL.Query().Get("fo") != "json" || !strings.Contains(r.URL.Query().Get("q"), "biology") {
			t.Errorf("unexpected query: %s", r.URL.String())
		}
		return `{"results":[{"id":"https://www.loc.gov/item/123/","title":"Biology","contributor":["Ada Author"],"date":"2001","subject":["Biology","Non-Fiction"],"language":["English"],"image_url":"https://example.test/cover.jpg"}]}`
	})
	provider := newLibraryOfCongressProvider(client, "https://loc.test/books/")
	results, err := provider.Search(context.Background(), providerRequest())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Title != "Biology" || results[0].Year == nil || *results[0].Year != 2001 {
		t.Fatalf("unexpected result: %#v", results)
	}
}

func TestWikidataSearchMapsEntities(t *testing.T) {
	client := fakeClient(t, func(r *http.Request) string {
		query := r.URL.Query()
		switch query.Get("action") {
		case "query":
			if !strings.Contains(query.Get("srsearch"), "biology") || !strings.Contains(query.Get("srsearch"), "haswbstatement:P31=Q571") {
				t.Errorf("unexpected srsearch: %q", query.Get("srsearch"))
			}
			return `{"query":{"search":[{"title":"Q1"}]}}`
		case "wbgetentities":
			if query.Get("props") == "labels|claims" {
				return `{"entities":{"Q1":{"id":"Q1","labels":{"en":{"value":"Biology","language":"en"}},"claims":{"P577":[{"mainsnak":{"datavalue":{"value":{"time":"+2001-01-01T00:00:00Z","precision":9}}}}],"P50":[{"mainsnak":{"datavalue":{"value":{"entity-type":"item","numeric-id":42,"id":"Q42"}}}}],"P921":[{"mainsnak":{"datavalue":{"value":{"id":"Q111"}}}}],"P212":[{"mainsnak":{"datavalue":{"value":"9780123456789"}}}]}}}}`
			}
			if !strings.Contains(query.Get("ids"), "Q42") || !strings.Contains(query.Get("ids"), "Q111") {
				t.Errorf("unexpected label ids: %q", query.Get("ids"))
			}
			return `{"entities":{"Q42":{"id":"Q42","labels":{"en":{"value":"Ada Author","language":"en"}}},"Q111":{"id":"Q111","labels":{"en":{"value":"Biology","language":"en"}}}}}`
		}
		t.Errorf("unexpected action: %s", r.URL.String())
		return "{}"
	})
	provider := newWikidataProvider(client, "https://wikidata.test/w/api.php")
	results, err := provider.Search(context.Background(), providerRequest())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Title != "Biology" || results[0].Year == nil || *results[0].Year != 2001 {
		t.Fatalf("unexpected result: %#v", results)
	}
	book := results[0]
	if len(book.Authors) != 1 || book.Authors[0] != "Ada Author" || len(book.Subjects) != 1 || book.Subjects[0] != "Biology" || len(book.ISBNs) != 1 {
		t.Fatalf("unexpected metadata: %#v", book)
	}
	if book.Source.Provider != "wikidata" || book.URL != "https://www.wikidata.org/wiki/Q1" {
		t.Fatalf("unexpected source: %#v", book.Source)
	}
}

func TestDefaultProviderClientHasNoTimeoutCap(t *testing.T) {
	client := defaultClient(nil)
	if client.Timeout != 0 {
		t.Fatalf("default client timeout = %v, want 0; per-provider context deadlines must govern instead", client.Timeout)
	}
}

func TestProjectGutenbergFeedClientDefaultsToLongTimeout(t *testing.T) {
	provider := newProjectGutenbergProvider(nil, nil, "https://gutenberg.test/catalog.csv.gz")
	if provider.feedClient == nil || provider.feedClient.Timeout != gutenbergFeedTimeout {
		t.Fatalf("feed client timeout = %v, want %v (the catalog download must not share the search budgets)", provider.feedClient, gutenbergFeedTimeout)
	}
}

func TestProjectGutenbergRefreshesOfficialCompressedCatalog(t *testing.T) {
	var compressed strings.Builder
	writer := gzip.NewWriter(&compressed)
	_, _ = writer.Write([]byte("Text#,Type,Issued,Title,Language,Authors,Subjects,Bookshelves,Downloads\n123,Text,1899-01-01,Science and Life,en,Ada Author,Science; Biology,Science,42\n"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	client := responseClient(func(request *http.Request) *http.Response {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(compressed.String())), Request: request}
	})
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := store.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	provider := newProjectGutenbergProvider(db, client, "https://gutenberg.test/catalog.csv.gz")
	if err := provider.refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	results, err := provider.Search(context.Background(), books.SearchRequest{Topics: []string{"biology"}, Genre: "non-fiction", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Popularity == nil || results[0].Popularity.Value != 42 || results[0].YearKind != "gutenberg_release_year" {
		t.Fatalf("unexpected result: %#v", results)
	}
}

func TestDefaultProviderSetAdvertisesSevenSources(t *testing.T) {
	deps := NewDependencies(&DependenciesConfig{DB: mustOpenSQLite(t)})
	if got := len(deps.All()); got != 7 {
		t.Fatalf("provider count = %d, want 7", got)
	}
	want := map[string]bool{"open_library": true, "doab": true, "gutendex": true, "internet_archive": true, "library_of_congress": true, "project_gutenberg": true, "wikidata": true}
	for _, provider := range deps.All() {
		delete(want, provider.Capabilities().ID)
	}
	if len(want) != 0 {
		t.Fatalf("providers not registered: %v", want)
	}
}

func mustOpenSQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
