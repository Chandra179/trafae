package providers

import (
	"context"
	"encoding/json"
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
	return books.ProviderCapability{ID: "doab", Name: "DOAB", Filters: []string{books.FilterTopics, books.FilterGenre, books.FilterYear, books.FilterLanguage}, Notes: []string{"DOAB catalogs scholarly open-access books; popularity and reader ratings are not provided."}}
}

func (p *doabProvider) Search(ctx context.Context, request books.SearchRequest) ([]books.Book, error) {
	terms := queryTerms(request)
	if len(terms) == 0 {
		terms = []string{"*:*"}
	}
	var found bookAccumulator
	for _, term := range terms {
		values := url.Values{}
		if term == "*:*" {
			values.Set("query", term)
		} else {
			values.Set("query", `dc.subject:"`+escapeQuery(term)+`"`)
		}
		values.Set("expand", "metadata,bitstreams")
		var response json.RawMessage
		if err := requestJSON(ctx, p.client, p.endpoint, values, "TrafaeBookDiscovery/1.0", &response); err != nil {
			return nil, err
		}
		for _, item := range findMaps(response) {
			metadata := extractMetadata(item)
			title := firstString(metadataValue(metadata, "dc.title"), metadataValue(metadata, "title"), stringValue(item["name"]))
			if title == "" {
				continue
			}
			id := firstString(stringValue(item["handle"]), stringValue(item["uuid"]), stringValue(item["id"]))
			subjects := metadataValues(metadata, "dc.subject")
			book := books.Book{ID: id, Title: title, Authors: metadataValues(metadata, "dc.contributor.author", "dc.creator"),
				Description: metadataValue(metadata, "dc.description"), Subjects: subjects, Genres: append([]string{"non-fiction"}, subjects...),
				Languages: metadataValues(metadata, "dc.language.iso", "dc.language"), ISBNs: metadataValues(metadata, "dc.identifier.isbn"),
				License: firstString(metadataValue(metadata, "dc.rights.uri"), metadataValue(metadata, "dc.rights"))}
			book.Year = parseYear(metadataValue(metadata, "dc.date.issued"))
			book.YearKind = "publication_year"
			if id != "" && strings.Contains(id, "/") {
				book.URL = "https://directory.doabooks.org/handle/" + id
			}
			if book.URL == "" {
				book.URL = firstString(stringValue(item["link"]), stringValue(item["url"]))
			}
			book.Source = books.BookSource{Provider: "doab", ID: id, URL: book.URL}
			found.add(book)
		}
	}
	return found.all(), nil
}

func escapeQuery(value string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(strings.TrimSpace(value))
}

func findMaps(raw json.RawMessage) []map[string]any {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return nil
	}
	var output []map[string]any
	var walk func(any)
	walk = func(value any) {
		switch v := value.(type) {
		case []any:
			for _, child := range v {
				walk(child)
			}
		case map[string]any:
			if _, hasMetadata := v["metadata"]; hasMetadata {
				output = append(output, v)
			}
			for _, child := range v {
				walk(child)
			}
		}
	}
	walk(value)
	return output
}

func extractMetadata(item map[string]any) map[string][]string {
	metadata := map[string][]string{}
	var walk func(any)
	walk = func(value any) {
		switch v := value.(type) {
		case []any:
			for _, child := range v {
				walk(child)
			}
		case map[string]any:
			if key, ok := v["key"].(string); ok {
				if val, ok := v["value"].(string); ok {
					metadata[key] = append(metadata[key], val)
				}
			}
			for _, child := range v {
				walk(child)
			}
		}
	}
	walk(item["metadata"])
	return metadata
}

func metadataValues(metadata map[string][]string, keys ...string) []string {
	values := []string{}
	for _, key := range keys {
		for _, value := range metadata[key] {
			values = addUnique(values, value)
		}
	}
	return values
}

func metadataValue(metadata map[string][]string, key string) string {
	if values := metadata[key]; len(values) > 0 {
		return values[0]
	}
	return ""
}
