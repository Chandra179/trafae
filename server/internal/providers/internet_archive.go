package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Chandra179/trafae/server/internal/books"
)

type internetArchiveProvider struct {
	client   *http.Client
	endpoint string
}

func newInternetArchiveProvider(client *http.Client, endpoint string) *internetArchiveProvider {
	return &internetArchiveProvider{client: client, endpoint: endpoint}
}

func (p *internetArchiveProvider) Capabilities() books.ProviderCapability {
	return books.ProviderCapability{ID: "internet_archive", Name: "Internet Archive", Filters: []string{books.FilterTopics, books.FilterGenre, books.FilterYear, books.FilterLanguage, books.FilterPopularity, books.FilterRating}, PopularityMetric: "downloads", RatingScale: 5, Notes: []string{"Ratings and download counts are present only on some archive items."}}
}

func (p *internetArchiveProvider) Search(ctx context.Context, request books.SearchRequest) ([]books.Book, error) {
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

func (p *internetArchiveProvider) searchTerm(ctx context.Context, request books.SearchRequest, term string) ([]books.Book, error) {
	values := url.Values{}
	if term == "" {
		values.Set("q", "mediatype:texts")
		values.Add("sort[]", "downloads desc")
	} else {
		values.Set("q", `mediatype:texts AND (title:"`+escapeQuery(term)+`" OR subject:"`+escapeQuery(term)+`")`)
	}
	for _, field := range []string{"title", "creator", "year", "description", "subject", "language", "downloads", "avg_rating", "num_reviews", "licenseurl", "identifier"} {
		values.Add("fl[]", field)
	}
	values.Set("rows", fmt.Sprint(withLimit(request, 200)))
	values.Set("page", "1")
	values.Set("output", "json")
	var response struct {
		Response struct {
			Docs []map[string]any `json:"docs"`
		} `json:"response"`
	}
	if err := requestJSON(ctx, p.client, p.endpoint, values, "TrafaeBookDiscovery/1.0", &response); err != nil {
		return nil, err
	}
	var found bookAccumulator
	for _, doc := range response.Response.Docs {
		id := stringValue(doc["identifier"])
		title := firstString(stringValue(doc["title"]), id)
		if id == "" || title == "" {
			continue
		}
		book := books.Book{ID: id, Title: title, Authors: stringValues(doc["creator"]), Description: stringValue(doc["description"]),
			Subjects: stringValues(doc["subject"]), Languages: stringValues(doc["language"]), Year: parseYear(doc["year"]), YearKind: "publication_year",
			URL: "https://archive.org/details/" + url.PathEscape(id), License: stringValue(doc["licenseurl"])}
		book.Genres = append([]string(nil), book.Subjects...)
		if count, ok := number(doc["downloads"]); ok {
			book.Popularity = &books.Metric{Value: count, Metric: "downloads"}
		}
		if rating, ok := number(doc["avg_rating"]); ok {
			count, _ := number(doc["num_reviews"])
			book.Rating = &books.Rating{Value: rating, Scale: 5, Count: int(count)}
		}
		book.Source = books.BookSource{Provider: "internet_archive", ID: id, URL: book.URL}
		found.add(book)
	}
	return found.all(), nil
}
