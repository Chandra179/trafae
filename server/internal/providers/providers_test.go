package providers

import (
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Chandra179/lux/server/internal/books"
	_ "modernc.org/sqlite"
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

func TestDOABSearchMapsMetadata(t *testing.T) {
	client := fakeClient(t, func(r *http.Request) string {
		if r.URL.Query().Get("query") == "" || r.URL.Query().Get("expand") != "metadata,bitstreams" {
			t.Errorf("unexpected query: %s", r.URL.String())
		}
		return `{"items":[{"uuid":"uuid-1","handle":"20.500/123","metadata":[{"key":"dc.title","value":"Open Biology"},{"key":"dc.contributor.author","value":"Ada Author"},{"key":"dc.subject","value":"Biology"},{"key":"dc.date.issued","value":"2020"},{"key":"dc.language.iso","value":"en"},{"key":"dc.rights.uri","value":"https://creativecommons.org/licenses/by/4.0/"}]}]}`
	})
	provider := newDOABProvider(client, "https://doab.test/rest/search")
	results, err := provider.Search(context.Background(), providerRequest())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Title != "Open Biology" || results[0].License == "" || results[0].Year == nil {
		t.Fatalf("unexpected result: %#v", results)
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

func TestWikidataSearchMapsBindings(t *testing.T) {
	client := fakeClient(t, func(r *http.Request) string {
		query := r.URL.Query().Get("query")
		if !strings.Contains(query, "biology") || !strings.Contains(query, "ORDER BY") {
			t.Errorf("unexpected SPARQL query: %s", query)
		}
		return `{"results":{"bindings":[{"book":{"value":"http://www.wikidata.org/entity/Q1"},"bookLabel":{"value":"Biology"},"authorLabel":{"value":"Ada Author"},"publication":{"value":"+2001-01-01T00:00:00Z"},"languageLabel":{"value":"English"},"subjectLabel":{"value":"Biology"},"genreLabel":{"value":"non-fiction"},"isbn":{"value":"9780123456789"}}]}}`
	})
	provider := newWikidataProvider(client, "https://wikidata.test/sparql")
	results, err := provider.Search(context.Background(), providerRequest())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Year == nil || *results[0].Year != 2001 || results[0].Source.Provider != "wikidata" {
		t.Fatalf("unexpected result: %#v", results)
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
