package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Chandra179/lux/server/internal/books"
)

type libraryOfCongressProvider struct {
	client   *http.Client
	endpoint string
}

func newLibraryOfCongressProvider(client *http.Client, endpoint string) *libraryOfCongressProvider {
	return &libraryOfCongressProvider{client: client, endpoint: endpoint}
}

func (p *libraryOfCongressProvider) Capabilities() books.ProviderCapability {
	return books.ProviderCapability{ID: "library_of_congress", Name: "Library of Congress", Filters: []string{books.FilterTopics, books.FilterGenre, books.FilterYear, books.FilterLanguage}, Notes: []string{"The books endpoint covers digital collections, not the full LoC book catalog; reader ratings and engagement counts are not supplied as book metrics."}}
}

func (p *libraryOfCongressProvider) Search(ctx context.Context, request books.SearchRequest) ([]books.Book, error) {
	terms := queryTerms(request)
	if len(terms) == 0 {
		terms = []string{""}
	}
	termResults, termErrs := searchTerms(ctx, terms, func(ctx context.Context, term string) ([]books.Book, error) {
		return p.searchTerm(ctx, request, term)
	})
	var found bookAccumulator
	for _, termBooks := range termResults {
		for _, book := range termBooks {
			found.add(book)
		}
	}
	return found.all(), termFailure(termErrs)
}

func (p *libraryOfCongressProvider) searchTerm(ctx context.Context, request books.SearchRequest, term string) ([]books.Book, error) {
	values := url.Values{}
	if term != "" {
		values.Set("q", term)
	}
	values.Set("fo", "json")
	values.Set("at", "results")
	values.Set("c", fmt.Sprint(withLimit(request, 100)))
	facets := []string{"original-format:books"}
	if request.Language != "" {
		facets = append(facets, "language:"+locLanguage(request.Language))
	}
	values.Set("fa", strings.Join(facets, "|"))
	var response struct {
		Results []map[string]any `json:"results"`
	}
	if err := requestJSON(ctx, p.client, p.endpoint, values, "TrafaeBookDiscovery/1.0", &response); err != nil {
		return nil, err
	}
	var found bookAccumulator
	for _, item := range response.Results {
		id := firstString(stringValue(item["id"]), stringValue(item["url"]))
		title := stringValue(item["title"])
		if title == "" {
			continue
		}
		book := books.Book{ID: id, Title: title, Authors: stringValues(item["contributor"]), Description: stringValue(item["description"]),
			Subjects: stringValues(item["subject"]), Languages: stringValues(item["language"]), Year: parseYear(item["date"]), YearKind: "publication_year",
			URL: firstString(stringValue(item["id"]), stringValue(item["url"])), CoverURL: firstString(stringValue(item["image_url"]))}
		book.Genres = append([]string(nil), book.Subjects...)
		book.ISBNs = stringValues(item["isbn"])
		book.Source = books.BookSource{Provider: "library_of_congress", ID: id, URL: book.URL}
		found.add(book)
	}
	return found.all(), nil
}
