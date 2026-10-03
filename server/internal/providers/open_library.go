package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Chandra179/trafae/server/internal/books"
)

type openLibraryProvider struct {
	client   *http.Client
	baseURL  string
	email    string
	mu       sync.Mutex
	nextCall time.Time
	cache    map[string]openLibraryCacheEntry
}

type openLibraryCacheEntry struct {
	books     []books.Book
	expiresAt time.Time
}

func newOpenLibraryProvider(client *http.Client, baseURL, email string) *openLibraryProvider {
	return &openLibraryProvider{client: client, baseURL: strings.TrimRight(baseURL, "/"), email: strings.TrimSpace(email), cache: make(map[string]openLibraryCacheEntry)}
}

func (p *openLibraryProvider) Capabilities() books.ProviderCapability {
	return books.ProviderCapability{ID: "open_library", Name: "Open Library", Filters: []string{books.FilterTopics, books.FilterGenre, books.FilterYear, books.FilterLanguage, books.FilterRating}, RatingScale: 5, Notes: []string{"Popularity counts are not available in search results."}}
}

func (p *openLibraryProvider) Search(ctx context.Context, request books.SearchRequest) ([]books.Book, error) {
	terms := queryTerms(request)
	if len(terms) == 0 {
		terms = []string{"non-fiction"}
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

func (p *openLibraryProvider) searchTerm(ctx context.Context, request books.SearchRequest, term string) ([]books.Book, error) {
	limit := withLimit(request, 100)
	cacheKey := strings.ToLower(term) + ":" + fmt.Sprint(limit)
	if cached, ok := p.cached(cacheKey); ok {
		return cached, nil
	}
	if err := p.wait(ctx); err != nil {
		return nil, err
	}
	values := url.Values{}
	values.Set("subject", term)
	values.Set("limit", fmt.Sprint(limit))
	values.Set("fields", "key,title,author_name,first_publish_year,subject,isbn,ratings_average,ratings_count,cover_i,language,edition_count")
	var response struct {
		Docs []map[string]any `json:"docs"`
	}
	userAgent := "TrafaeBookDiscovery/1.0"
	if p.email != "" {
		userAgent += " (" + p.email + ")"
	}
	if err := requestJSON(ctx, p.client, p.baseURL+"/search.json", values, userAgent, &response); err != nil {
		return nil, err
	}
	var termBooks bookAccumulator
	for _, doc := range response.Docs {
		book := books.Book{
			ID: stringValue(doc["key"]), Title: stringValue(doc["title"]),
			Authors: stringValues(doc["author_name"]), Subjects: stringValues(doc["subject"]),
			Languages: stringValues(doc["language"]), ISBNs: stringValues(doc["isbn"]),
			URL: openLibraryURL(stringValue(doc["key"])),
		}
		book.Genres = append([]string(nil), book.Subjects...)
		book.Year = parseYear(doc["first_publish_year"])
		book.YearKind = "first_publication_year"
		if coverID, ok := number(doc["cover_i"]); ok {
			book.CoverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%.0f-M.jpg", coverID)
		}
		if rating, ok := number(doc["ratings_average"]); ok {
			count, _ := number(doc["ratings_count"])
			book.Rating = &books.Rating{Value: rating, Scale: 5, Count: int(count)}
		}
		book.Source = books.BookSource{Provider: "open_library", ID: book.ID, URL: book.URL}
		if book.Title != "" {
			termBooks.add(book)
		}
	}
	p.cacheBooks(cacheKey, termBooks.all())
	return termBooks.all(), nil
}

func (p *openLibraryProvider) cached(key string) ([]books.Book, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	entry, ok := p.cache[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(p.cache, key)
		return nil, false
	}
	return append([]books.Book(nil), entry.books...), true
}

func (p *openLibraryProvider) cacheBooks(key string, values []books.Book) {
	p.mu.Lock()
	p.cache[key] = openLibraryCacheEntry{books: append([]books.Book(nil), values...), expiresAt: time.Now().Add(10 * time.Minute)}
	p.mu.Unlock()
}

func (p *openLibraryProvider) wait(ctx context.Context) error {
	delay := time.Second
	if p.email != "" {
		delay = time.Second / 3
	}
	p.mu.Lock()
	wait := time.Until(p.nextCall)
	if wait < 0 {
		wait = 0
	}
	p.nextCall = time.Now().Add(wait + delay)
	p.mu.Unlock()
	if wait == 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func openLibraryURL(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if strings.HasPrefix(key, "/") {
		return "https://openlibrary.org" + key
	}
	return "https://openlibrary.org/works/" + key
}
