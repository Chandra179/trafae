package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Chandra179/lux/server/internal/books"
)

type wikidataProvider struct {
	client   *http.Client
	endpoint string
}

func newWikidataProvider(client *http.Client, endpoint string) *wikidataProvider {
	return &wikidataProvider{client: client, endpoint: endpoint}
}

func (p *wikidataProvider) Capabilities() books.ProviderCapability {
	return books.ProviderCapability{ID: "wikidata", Name: "Wikidata", Filters: []string{books.FilterTopics, books.FilterGenre, books.FilterYear, books.FilterLanguage}, Notes: []string{"Metadata coverage is community supplied and can be sparse; popularity and ratings are not available."}}
}

func (p *wikidataProvider) Search(ctx context.Context, request books.SearchRequest) ([]books.Book, error) {
	terms := make([]string, 0, len(request.Topics)+1)
	for _, term := range request.Topics {
		if term = strings.TrimSpace(term); term != "" {
			terms = append(terms, term)
		}
	}
	if len(terms) == 0 {
		terms = append(terms, request.Genre)
	}
	filters := make([]string, 0, len(terms)*2)
	for _, term := range terms {
		literal := sparqlString(term)
		filters = append(filters, `(CONTAINS(LCASE(STR(?bookLabel)), LCASE(`+literal+`)) || CONTAINS(LCASE(STR(?subjectLabel)), LCASE(`+literal+`)) || CONTAINS(LCASE(STR(?genreLabel)), LCASE(`+literal+`)))`)
	}
	filterExpression := strings.Join(filters, " || ")
	query := `SELECT DISTINCT ?book ?bookLabel ?authorLabel ?publication ?languageLabel ?subjectLabel ?genreLabel ?isbn WHERE {
  ?book wdt:P31/wdt:P279* wd:Q571.
  ?book rdfs:label ?bookLabel. FILTER(LANG(?bookLabel) = "en")
  OPTIONAL { ?book wdt:P50 ?author. }
  OPTIONAL { ?author rdfs:label ?authorLabel. FILTER(LANG(?authorLabel) = "en") }
  OPTIONAL { ?book wdt:P577 ?publication. }
  OPTIONAL { ?book wdt:P407 ?language. }
  OPTIONAL { ?language rdfs:label ?languageLabel. FILTER(LANG(?languageLabel) = "en") }
  OPTIONAL { ?book wdt:P921 ?subject. }
  OPTIONAL { ?subject rdfs:label ?subjectLabel. FILTER(LANG(?subjectLabel) = "en") }
  OPTIONAL { ?book wdt:P136 ?genre. }
  OPTIONAL { ?genre rdfs:label ?genreLabel. FILTER(LANG(?genreLabel) = "en") }
  OPTIONAL { ?book wdt:P212 ?isbn. }
  FILTER(` + filterExpression + `)
} ORDER BY ?bookLabel LIMIT ` + fmt.Sprint(withLimit(request, 100))
	values := url.Values{"query": []string{query}, "format": []string{"json"}}
	var response struct {
		Results struct {
			Bindings []map[string]struct {
				Value string `json:"value"`
			} `json:"bindings"`
		} `json:"results"`
	}
	if err := requestJSON(ctx, p.client, p.endpoint, values, "TrafaeBookDiscovery/1.0 (book discovery)", &response); err != nil {
		return nil, err
	}
	var found bookAccumulator
	for _, binding := range response.Results.Bindings {
		id := binding["book"].Value
		title := binding["bookLabel"].Value
		if id == "" || title == "" {
			continue
		}
		book := books.Book{ID: id, Title: title, URL: id, Year: parseYear(binding["publication"].Value), YearKind: "publication_year"}
		if author := binding["authorLabel"].Value; author != "" {
			book.Authors = []string{author}
		}
		if language := binding["languageLabel"].Value; language != "" {
			book.Languages = []string{language}
		}
		if subject := binding["subjectLabel"].Value; subject != "" {
			book.Subjects = []string{subject}
		}
		if genre := binding["genreLabel"].Value; genre != "" {
			book.Genres = []string{genre}
		}
		if isbn := binding["isbn"].Value; isbn != "" {
			book.ISBNs = []string{isbn}
		}
		book.Source = books.BookSource{Provider: "wikidata", ID: id, URL: id}
		found.add(book)
	}
	return found.all(), nil
}

func sparqlString(value string) string {
	value = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`).Replace(value)
	return `"` + value + `"`
}
