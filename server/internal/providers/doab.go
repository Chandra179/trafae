package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Chandra179/lux/server/internal/books"
)

type doabProvider struct {
	client   *http.Client
	endpoint string
}

func newDOABProvider(client *http.Client, endpoint string) *doabProvider {
	return &doabProvider{client: client, endpoint: endpoint}
}

func (p *doabProvider) Capabilities() books.ProviderCapability {
	return books.ProviderCapability{ID: "doab", Name: "DOAB", Filters: []string{books.FilterTopics, books.FilterGenre, books.FilterYear, books.FilterLanguage}, Notes: []string{"DOAB catalogs scholarly open-access books; links go to the DOAB record where the PDF can be downloaded."}}
}

func (p *doabProvider) Search(ctx context.Context, request books.SearchRequest) ([]books.Book, error) {
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

func (p *doabProvider) searchTerm(ctx context.Context, request books.SearchRequest, term string) ([]books.Book, error) {
	values := url.Values{}
	if term != "" {
		values.Set("query", term)
	}
	values.Set("size", fmt.Sprint(withLimit(request, 100)))
	values.Set("page", "0")
	var response struct {
		Embedded struct {
			SearchResult struct {
				Embedded struct {
					Objects []struct {
						Embedded struct {
							IndexableObject struct {
								UUID     string             `json:"uuid"`
								Handle   string             `json:"handle"`
								Name     string             `json:"name"`
								Metadata doabMetadataValues `json:"metadata"`
							} `json:"indexableObject"`
						} `json:"_embedded"`
					} `json:"objects"`
				} `json:"_embedded"`
			} `json:"searchResult"`
		} `json:"_embedded"`
	}
	if err := requestJSON(ctx, p.client, p.endpoint, values, "TrafaeBookDiscovery/1.0", &response); err != nil {
		return nil, err
	}
	var found bookAccumulator
	for _, object := range response.Embedded.SearchResult.Embedded.Objects {
		item := object.Embedded.IndexableObject
		metadata := item.Metadata.normalized()
		title := firstString(metadata.first("dc.title"), item.Name)
		if title == "" {
			continue
		}
		id := firstString(item.UUID, item.Handle)
		subjects := metadata.all("dc.subject")
		book := books.Book{ID: id, Title: title, Authors: metadata.all("dc.contributor.author", "dc.creator"),
			Description: metadata.first("dc.description.abstract", "dc.description"), Subjects: subjects,
			Genres:    append([]string{"non-fiction"}, subjects...),
			Languages: metadata.all("dc.language.iso", "dc.language"), ISBNs: metadata.all("dc.identifier.isbn"),
			License: firstString(metadata.first("dc.rights.uri"), metadata.first("dc.rights"), "Open Access")}
		book.Year = parseYear(metadata.first("dc.date.issued"))
		book.YearKind = "publication_year"
		book.URL = firstString(metadata.first("dc.identifier.uri"), doabHandleURL(item.Handle), doabItemURL(item.UUID))
		book.Source = books.BookSource{Provider: "doab", ID: id, URL: book.URL}
		found.add(book)
	}
	return found.all(), nil
}

func doabHandleURL(handle string) string {
	if handle == "" {
		return ""
	}
	return "https://directory.doabooks.org/handle/" + handle
}

func doabItemURL(uuid string) string {
	if uuid == "" {
		return ""
	}
	return "https://directory.doabooks.org/items/" + uuid
}

type doabMetadataValues map[string][]doabMetadataValue

type doabMetadataValue struct {
	Value string `json:"value"`
}

func (raw doabMetadataValues) normalized() doabMetadata {
	metadata := make(doabMetadata, len(raw))
	for key, values := range raw {
		parsed := make([]string, 0, len(values))
		for _, value := range values {
			parsed = append(parsed, value.Value)
		}
		metadata[key] = parsed
	}
	return metadata
}

type doabMetadata map[string][]string

func (m doabMetadata) first(keys ...string) string {
	for _, key := range keys {
		if values := m[key]; len(values) > 0 && strings.TrimSpace(values[0]) != "" {
			return strings.TrimSpace(values[0])
		}
	}
	return ""
}

func (m doabMetadata) all(keys ...string) []string {
	values := []string{}
	for _, key := range keys {
		for _, value := range m[key] {
			values = addUnique(values, value)
		}
	}
	return values
}
