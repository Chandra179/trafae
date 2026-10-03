package books

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"go.uber.org/zap"
)

const rrfK = 60

func (d *dependencies) Capabilities() []ProviderCapability {
	capabilities := make([]ProviderCapability, 0, len(d.providers))
	for _, provider := range d.providers {
		capabilities = append(capabilities, provider.Capabilities())
	}
	sort.Slice(capabilities, func(i, j int) bool { return capabilities[i].ID < capabilities[j].ID })
	return capabilities
}

func (d *dependencies) Search(ctx context.Context, request SearchRequest) (SearchResponse, error) {
	started := time.Now()
	defer func() { d.metrics.RecordSearchDuration(time.Since(started)) }()
	request = d.normalizeRequest(request)
	if entry, ok := d.cache.get(request, time.Now()); ok {
		d.metrics.RecordSearch(true)
		return poolResponse(entry, request), nil
	}
	d.metrics.RecordSearch(false)
	response := SearchResponse{Results: []SearchResult{}, Providers: []ProviderStatus{}, Limit: request.Limit, Page: request.Page}
	selected := d.selectedProviders(request.Providers)
	if len(selected) == 0 {
		return response, errors.New("no configured providers matched the request")
	}

	// Providers fetch the requested page plus the lookahead so the next few
	// Next-clicks slice the cached pool instead of re-fanning-out; going
	// deeper than the cached pool re-fetches with fresh lookahead.
	fetchRequest := request
	fetchLimit := min(request.Page+poolLookaheadPages, MaxSearchPage) * request.Limit
	fetchRequest.Limit = fetchLimit
	// Genre-only browse: keyword-search the genre so every provider can
	// contribute, instead of post-filtering each provider's default batch.
	if len(fetchRequest.Topics) == 0 && !strings.EqualFold(request.Genre, DefaultGenre) {
		fetchRequest.Topics = []string{request.Genre}
	}

	type providerResult struct {
		capability ProviderCapability
		books      []Book
		err        error
		skipped    string
	}
	results := make(chan providerResult, len(selected))
	var wg sync.WaitGroup
	for _, provider := range selected {
		provider := provider
		capability := provider.Capabilities()
		if reason := unsupportedFilter(capability, request.RequiredFilters()); reason != "" {
			results <- providerResult{capability: capability, skipped: reason}
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			providerCtx, cancel := context.WithTimeout(ctx, d.searchTimeoutFor(capability.ID))
			defer cancel()
			books, err := provider.Search(providerCtx, fetchRequest)
			results <- providerResult{capability: capability, books: books, err: err}
		}()
	}
	wg.Wait()
	close(results)

	merged := make(map[string]*SearchResult)
	providerSuccesses := 0
	for result := range results {
		status := ProviderStatus{Provider: result.capability.ID}
		switch {
		case result.skipped != "":
			status.Status = "skipped"
			status.Reason = result.skipped
		case result.err != nil && len(result.books) == 0:
			status.Status = "error"
			status.Reason = result.err.Error()
			d.logger.Warn("book provider search failed", zap.String("provider", result.capability.ID), zap.Error(result.err))
		case result.err != nil:
			// The provider served some topic results before one upstream
			// request failed; keep the partial set and surface the failure.
			status.Status = "partial"
			status.Reason = result.err.Error()
			status.Count = len(result.books)
			providerSuccesses++
			d.logger.Warn("book provider returned partial results", zap.String("provider", result.capability.ID), zap.Error(result.err))
			mergeProviderBooks(result.books, request, result.capability.ID, merged)
		default:
			status.Status = "ok"
			providerSuccesses++
			status.Count = len(result.books)
			mergeProviderBooks(result.books, request, result.capability.ID, merged)
		}
		response.Providers = append(response.Providers, status)
	}
	sort.Slice(response.Providers, func(i, j int) bool { return response.Providers[i].Provider < response.Providers[j].Provider })
	for _, status := range response.Providers {
		d.metrics.RecordProviderStatus(status.Provider, status.Status)
	}
	if providerSuccesses == 0 {
		return response, errors.New("all eligible book providers failed")
	}

	ordered := make([]SearchResult, 0, len(merged))
	for _, result := range merged {
		ordered = append(ordered, *result)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if math.Abs(ordered[i].RRFScore-ordered[j].RRFScore) > 1e-12 {
			return ordered[i].RRFScore > ordered[j].RRFScore
		}
		return strings.ToLower(ordered[i].Book.Title) < strings.ToLower(ordered[j].Book.Title)
	})
	d.cache.put(request, ordered, response.Providers, fetchLimit, time.Now())
	return poolResponse(searchCacheEntry{pool: ordered, providers: response.Providers}, request), nil
}

// poolResponse slices the cached pool for the requested page. The pool and
// statuses are shared cache state and treated as read-only.
func poolResponse(entry searchCacheEntry, request SearchRequest) SearchResponse {
	response := SearchResponse{Results: []SearchResult{}, Providers: entry.providers, Limit: request.Limit, Page: request.Page}
	start := (request.Page - 1) * request.Limit
	if start > len(entry.pool) {
		start = len(entry.pool)
	}
	end := start + request.Limit
	if end > len(entry.pool) {
		end = len(entry.pool)
	}
	response.HasMore = end < len(entry.pool)
	response.Results = entry.pool[start:end]
	return response
}

// mergeProviderBooks folds one provider's ranked results into the fusion pool,
// scoring each book by its rank and collapsing duplicates across providers.
func mergeProviderBooks(providerBooks []Book, request SearchRequest, providerID string, merged map[string]*SearchResult) {
	for rank, book := range providerBooks {
		if !matchesRequest(book, request) {
			continue
		}
		key := dedupeKey(book)
		if key == "" {
			key = providerID + ":" + normalized(book.ID)
		}
		entry, exists := merged[key]
		if !exists {
			entry = &SearchResult{Book: book, Sources: []BookSource{}, RRFScore: 0}
			merged[key] = entry
		}
		entry.RRFScore += 1 / float64(rrfK+rank+1)
		if !sourcePresent(entry.Sources, book.Source.Provider, book.Source.ID) {
			source := book.Source
			source.Popularity = book.Popularity
			source.Rating = book.Rating
			entry.Sources = append(entry.Sources, source)
		}
		mergeBookMetadata(&entry.Book, book)
	}
}

// searchTimeoutFor returns the configured call budget for one provider,
// falling back to the shared search timeout.
func (d *dependencies) searchTimeoutFor(providerID string) time.Duration {
	if timeout, ok := d.searchTimeouts[providerID]; ok && timeout > 0 {
		return timeout
	}
	return d.searchTimeout
}

func (d *dependencies) selectedProviders(ids []string) []Provider {
	if len(ids) == 0 {
		return append([]Provider(nil), d.providers...)
	}
	selected := make([]Provider, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		id = strings.ToLower(strings.TrimSpace(id))
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if provider, ok := d.byID[id]; ok {
			selected = append(selected, provider)
		}
	}
	return selected
}

func (d *dependencies) normalizeRequest(request SearchRequest) SearchRequest {
	if strings.TrimSpace(request.Genre) == "" {
		request.Genre = d.defaultGenre
	}
	request.Genre = strings.TrimSpace(request.Genre)
	if request.Limit <= 0 {
		request.Limit = d.defaultLimit
	}
	if request.Limit > MaxResultLimit {
		request.Limit = MaxResultLimit
	}
	if request.Page <= 0 {
		request.Page = 1
	}
	if request.Page > MaxSearchPage {
		request.Page = MaxSearchPage
	}
	request.Language = strings.ToLower(strings.TrimSpace(request.Language))
	for i := range request.Topics {
		request.Topics[i] = strings.TrimSpace(request.Topics[i])
	}
	return request
}

func unsupportedFilter(capability ProviderCapability, requested []string) string {
	supported := make(map[string]struct{}, len(capability.Filters))
	for _, filter := range capability.Filters {
		supported[strings.ToLower(filter)] = struct{}{}
	}
	for _, filter := range requested {
		if _, ok := supported[filter]; !ok {
			return fmt.Sprintf("provider does not support requested %s filter", filter)
		}
	}
	return ""
}

func matchesRequest(book Book, request SearchRequest) bool {
	if request.MinYear != nil && (book.Year == nil || *book.Year < *request.MinYear) {
		return false
	}
	if request.MaxYear != nil && (book.Year == nil || *book.Year > *request.MaxYear) {
		return false
	}
	if request.Language != "" && !languageMatches(book.Languages, request.Language) {
		return false
	}
	if request.MinPopularity != nil && (book.Popularity == nil || book.Popularity.Value < *request.MinPopularity) {
		return false
	}
	if request.MinRating != nil && (book.Rating == nil || book.Rating.Value < *request.MinRating) {
		return false
	}
	if len(request.Topics) > 0 && !slices.ContainsFunc(request.Topics, func(topic string) bool {
		return containsTerm(book.Subjects, topic) || containsTerm(book.Genres, topic) ||
			strings.Contains(strings.ToLower(book.Title), strings.ToLower(topic)) ||
			strings.Contains(strings.ToLower(book.Description), strings.ToLower(topic))
	}) {
		return false
	}
	return request.Genre == "" || genreMatches(book, request.Genre)
}

// DefaultGenreTopics lists the non-fiction genres the genre matcher accepts as
// evidence that a book is non-fiction. The Project Gutenberg provider reuses
// the same list as fallback search terms for default-genre browses, so the
// two stay in sync.
var DefaultGenreTopics = []string{"history", "science", "psychology", "philosophy", "biography", "economics", "business", "politics", "sociology", "education", "travel", "health", "technology", "religion"}

func genreMatches(book Book, genre string) bool {
	genre = normalized(genre)
	candidates := append(append([]string(nil), book.Genres...), book.Subjects...)
	if containsAnyPhrase(candidates, []string{genre}) {
		return true
	}
	if genre != "non fiction" && genre != "nonfiction" {
		return false
	}
	// Any fiction mention without a "non" qualifier disqualifies the book,
	// even when another subject supplies non-fiction evidence.
	for _, item := range candidates {
		value := normalized(item)
		if strings.Contains(value, "fiction") && !strings.Contains(value, "non fiction") && !strings.Contains(value, "nonfiction") {
			return false
		}
	}
	return containsAnyPhrase(candidates, DefaultGenreTopics)
}

// containsAnyPhrase reports whether any haystack contains any of the phrases
// as whole words.
func containsAnyPhrase(haystacks, phrases []string) bool {
	for _, haystack := range haystacks {
		value := normalized(haystack)
		for _, phrase := range phrases {
			if containsPhrase(value, phrase) {
				return true
			}
		}
	}
	return false
}

// containsPhrase reports whether phrase appears in haystack as whole words.
// Both sides are normalized (lowercase, single-space separated), so padding
// each with spaces and substring-matching is exact whole-word matching —
// "art" cannot match "cartography", "history" cannot match "prehistory".
func containsPhrase(haystack, phrase string) bool {
	if phrase == "" {
		return false
	}
	return strings.Contains(" "+haystack+" ", " "+phrase+" ")
}

func dedupeKey(book Book) string {
	if len(book.ISBNs) > 0 {
		for _, isbn := range book.ISBNs {
			clean := strings.NewReplacer("-", "", " ", "").Replace(strings.ToLower(isbn))
			if clean != "" {
				return "isbn:" + clean
			}
		}
	}
	if len(book.Authors) == 0 || strings.TrimSpace(book.Title) == "" {
		return ""
	}
	return "title-author:" + normalized(book.Title) + ":" + normalized(book.Authors[0])
}

func sourcePresent(sources []BookSource, provider, id string) bool {
	for _, source := range sources {
		if source.Provider == provider && source.ID == id {
			return true
		}
	}
	return false
}

func mergeBookMetadata(destination *Book, incoming Book) {
	if destination.Description == "" {
		destination.Description = incoming.Description
	}
	if destination.Year == nil {
		destination.Year = incoming.Year
		destination.YearKind = incoming.YearKind
	}
	if len(destination.Genres) == 0 {
		destination.Genres = incoming.Genres
	}
	if len(destination.Subjects) == 0 {
		destination.Subjects = incoming.Subjects
	}
	if len(destination.Languages) == 0 {
		destination.Languages = incoming.Languages
	}
	if len(destination.ISBNs) == 0 {
		destination.ISBNs = incoming.ISBNs
	}
	if destination.URL == "" {
		destination.URL = incoming.URL
	}
	if destination.CoverURL == "" {
		destination.CoverURL = incoming.CoverURL
	}
	if destination.License == "" {
		destination.License = incoming.License
	}
	if len(destination.AccessURLs) == 0 {
		destination.AccessURLs = incoming.AccessURLs
	}
}

// CanonicalLanguageCode collapses accepted language spellings to one code, so
// "eng", "english" and "en" compare equal without listing every pair.
// Unknown spellings pass through lowercased. Providers reuse this so the
// language knowledge lives in exactly one place.
func CanonicalLanguageCode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if canonical, ok := languageCanonical[value]; ok {
		return canonical
	}
	return value
}

var languageCanonical = map[string]string{
	"en": "en", "eng": "en", "english": "en",
	"fr": "fr", "fre": "fr", "fra": "fr", "french": "fr",
	"de": "de", "ger": "de", "deu": "de", "german": "de",
	"es": "es", "spa": "es", "spanish": "es",
	"it": "it", "ita": "it", "italian": "it",
	"pt": "pt", "por": "pt", "portuguese": "pt",
}

// languageMatches reports whether one of the book's language values matches
// the requested language. Values may be compound ("fre/ger"), so every
// separator-delimited token is compared in canonical form.
func languageMatches(values []string, target string) bool {
	target = CanonicalLanguageCode(target)
	for _, value := range values {
		for _, token := range strings.FieldsFunc(strings.ToLower(value), isLanguageSeparator) {
			if CanonicalLanguageCode(token) == target {
				return true
			}
		}
	}
	return false
}

func isLanguageSeparator(r rune) bool { return r == '/' || r == ':' }

func containsTerm(values []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), target) {
			return true
		}
	}
	return false
}

func normalized(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	space := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteRune(r)
			space = false
		} else {
			space = true
		}
	}
	return b.String()
}
