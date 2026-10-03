package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/Chandra179/trafae/server/internal/books"
)

// wikidataSearchLimit is the CirrusSearch page size; the Action API allows at
// most 50 results per search request without elevated rights.
const wikidataSearchLimit = 50

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

func (p *wikidataProvider) searchTerm(ctx context.Context, request books.SearchRequest, term string) ([]books.Book, error) {
	searchValues := url.Values{
		"action":   []string{"query"},
		"list":     []string{"search"},
		"format":   []string{"json"},
		"srsearch": []string{wikidataBookQuery(term, request.Language)},
		"srlimit":  []string{strconv.Itoa(wikidataSearchLimit)},
		"srprop":   []string{""},
	}
	var search struct {
		Query struct {
			Search []struct {
				Title string `json:"title"`
			} `json:"search"`
		} `json:"query"`
	}
	if err := requestJSON(ctx, p.client, p.endpoint, searchValues, "TrafaeBookDiscovery/1.0 (book discovery)", &search); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(search.Query.Search))
	for _, hit := range search.Query.Search {
		if id := strings.TrimSpace(hit.Title); id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	entities, err := p.entities(ctx, ids)
	if err != nil {
		return nil, err
	}

	references := map[string]struct{}{}
	for _, id := range ids {
		entity, ok := entities[id]
		if !ok {
			continue
		}
		for _, property := range []string{"P50", "P407", "P921", "P136"} {
			for _, value := range entity.claimIDs(property) {
				references[value] = struct{}{}
			}
		}
	}
	referenceIDs := make([]string, 0, len(references))
	for id := range references {
		referenceIDs = append(referenceIDs, id)
	}
	sort.Strings(referenceIDs)
	referenceLabels, err := p.labels(ctx, referenceIDs)
	if err != nil {
		return nil, err
	}

	matched := make([]books.Book, 0, len(ids))
	for _, id := range ids {
		entity, ok := entities[id]
		if !ok {
			continue
		}
		title := entity.label()
		if title == "" {
			continue
		}
		book := books.Book{ID: id, Title: title, URL: "https://www.wikidata.org/wiki/" + id,
			Year: entity.claimYear("P577"), YearKind: "publication_year", ISBNs: entity.claimStrings("P212")}
		book.Authors = referenceLabels.values(entity.claimIDs("P50"))
		book.Languages = referenceLabels.values(entity.claimIDs("P407"))
		book.Subjects = referenceLabels.values(entity.claimIDs("P921"))
		book.Genres = referenceLabels.values(entity.claimIDs("P136"))
		book.Source = books.BookSource{Provider: "wikidata", ID: id, URL: book.URL}
		matched = append(matched, book)
	}
	return matched, nil
}

// wikidataBookQuery builds the entity-search string: books (P31=Q571) for the
// term, plus the language-of-work constraint (P407) when a known language is
// requested. Unknown language spellings simply omit the constraint rather
// than sending a statement Wikidata cannot resolve.
func wikidataBookQuery(term, language string) string {
	query := strings.TrimSpace(term) + " haswbstatement:P31=Q571"
	if qid := wikidataLanguage(language); qid != "" {
		query += " haswbstatement:P407=" + qid
	}
	return query
}

func (p *wikidataProvider) entities(ctx context.Context, ids []string) (map[string]wikidataEntity, error) {
	values := url.Values{
		"action":           []string{"wbgetentities"},
		"format":           []string{"json"},
		"props":            []string{"labels|claims"},
		"languages":        []string{"en"},
		"languagefallback": []string{"1"},
		"ids":              []string{strings.Join(ids, "|")},
	}
	var response struct {
		Entities map[string]wikidataEntity `json:"entities"`
	}
	if err := requestJSON(ctx, p.client, p.endpoint, values, "TrafaeBookDiscovery/1.0 (book discovery)", &response); err != nil {
		return nil, err
	}
	return response.Entities, nil
}

func (p *wikidataProvider) labels(ctx context.Context, ids []string) (wikidataLabels, error) {
	labels := wikidataLabels{}
	for start := 0; start < len(ids); start += 50 {
		end := start + 50
		if end > len(ids) {
			end = len(ids)
		}
		values := url.Values{
			"action":           []string{"wbgetentities"},
			"format":           []string{"json"},
			"props":            []string{"labels"},
			"languages":        []string{"en"},
			"languagefallback": []string{"1"},
			"ids":              []string{strings.Join(ids[start:end], "|")},
		}
		var response struct {
			Entities map[string]wikidataEntity `json:"entities"`
		}
		if err := requestJSON(ctx, p.client, p.endpoint, values, "TrafaeBookDiscovery/1.0 (book discovery)", &response); err != nil {
			return nil, err
		}
		for id, entity := range response.Entities {
			if label := entity.label(); label != "" {
				labels[id] = label
			}
		}
	}
	return labels, nil
}

type wikidataLabels map[string]string

func (l wikidataLabels) values(ids []string) []string {
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		if label := l[id]; label != "" {
			values = addUnique(values, label)
		}
	}
	return values
}

type wikidataEntity struct {
	ID     string `json:"id"`
	Labels map[string]struct {
		Value string `json:"value"`
	} `json:"labels"`
	Claims map[string][]struct {
		Mainsnak struct {
			Datavalue *struct {
				Value json.RawMessage `json:"value"`
			} `json:"datavalue"`
		} `json:"mainsnak"`
	} `json:"claims"`
}

func (e wikidataEntity) label() string {
	if en, ok := e.Labels["en"]; ok {
		return strings.TrimSpace(en.Value)
	}
	for _, label := range e.Labels {
		if value := strings.TrimSpace(label.Value); value != "" {
			return value
		}
	}
	return ""
}

func (e wikidataEntity) claimRaw(property string) []json.RawMessage {
	claims := e.Claims[property]
	values := make([]json.RawMessage, 0, len(claims))
	for _, claim := range claims {
		if claim.Mainsnak.Datavalue != nil {
			values = append(values, claim.Mainsnak.Datavalue.Value)
		}
	}
	return values
}

func (e wikidataEntity) claimIDs(property string) []string {
	values := make([]string, 0)
	for _, raw := range e.claimRaw(property) {
		var entity struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(raw, &entity) == nil && entity.ID != "" {
			values = append(values, entity.ID)
		}
	}
	return values
}

func (e wikidataEntity) claimStrings(property string) []string {
	values := make([]string, 0)
	for _, raw := range e.claimRaw(property) {
		var value string
		if json.Unmarshal(raw, &value) == nil && strings.TrimSpace(value) != "" {
			values = append(values, strings.TrimSpace(value))
		}
	}
	return values
}

func (e wikidataEntity) claimYear(property string) *int {
	for _, raw := range e.claimRaw(property) {
		var moment struct {
			Time string `json:"time"`
		}
		if json.Unmarshal(raw, &moment) == nil {
			if year := parseYear(moment.Time); year != nil {
				return year
			}
		}
	}
	return nil
}
