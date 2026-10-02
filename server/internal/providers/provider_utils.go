package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Chandra179/lux/server/internal/books"
)

const maxResponseBytes = 16 << 20

func requestJSON(ctx context.Context, client *http.Client, endpoint string, values url.Values, userAgent string, target any) error {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("parse provider URL: %w", err)
	}
	query := parsed.Query()
	for key, list := range values {
		for _, value := range list {
			query.Add(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return fmt.Errorf("create provider request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	if userAgent != "" {
		request.Header.Set("User-Agent", userAgent)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request %s: %w", parsed.Host, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("%s returned HTTP %d", parsed.Host, response.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes))
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode %s response: %w", parsed.Host, err)
	}
	return nil
}

func queryTerms(request books.SearchRequest) []string {
	terms := make([]string, 0, len(request.Topics)+1)
	seen := map[string]struct{}{}
	for _, term := range request.Topics {
		term = strings.TrimSpace(term)
		if term != "" {
			key := strings.ToLower(term)
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				terms = append(terms, term)
			}
		}
	}
	return terms
}

func parseYear(value any) *int {
	var parsed int
	switch v := value.(type) {
	case int:
		parsed = v
	case int64:
		parsed = int(v)
	case float64:
		parsed = int(v)
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return nil
		}
		parsed = int(n)
	case string:
		v = strings.TrimSpace(v)
		var year strings.Builder
		for i, r := range v {
			if i == 0 && (r == '+' || r == '-') {
				year.WriteRune(r)
				continue
			}
			if r < '0' || r > '9' {
				break
			}
			year.WriteRune(r)
			if year.Len() >= 5 {
				break
			}
		}
		n, err := strconv.Atoi(year.String())
		if err != nil {
			return nil
		}
		parsed = n
	default:
		return nil
	}
	if parsed < -3000 || parsed > 3000 {
		return nil
	}
	return &parsed
}

func stringValues(value any) []string {
	values := make([]string, 0)
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) != "" {
			values = append(values, strings.TrimSpace(v))
		}
	case []string:
		for _, item := range v {
			if strings.TrimSpace(item) != "" {
				values = append(values, strings.TrimSpace(item))
			}
		}
	case []any:
		for _, item := range v {
			switch item := item.(type) {
			case string:
				if strings.TrimSpace(item) != "" {
					values = append(values, strings.TrimSpace(item))
				}
			case map[string]any:
				for _, key := range []string{"name", "value", "label"} {
					if name, ok := item[key].(string); ok && strings.TrimSpace(name) != "" {
						values = append(values, strings.TrimSpace(name))
						break
					}
				}
			}
		}
	}
	return values
}

func number(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		n, err := v.Float64()
		return n, err == nil
	case string:
		n, err := strconv.ParseFloat(v, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func withLimit(request books.SearchRequest, max int) int {
	limit := request.Limit * 3
	if limit < 30 {
		limit = 30
	}
	if limit > max {
		limit = max
	}
	return limit
}

func defaultClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: 18 * time.Second}
}

func addUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, current := range values {
		if strings.EqualFold(current, value) {
			return values
		}
	}
	return append(values, value)
}

func firstString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func safeNextURL(base, next string) string {
	baseURL, baseErr := url.Parse(base)
	nextURL, nextErr := url.Parse(next)
	if baseErr != nil || nextErr != nil || next == "" {
		return ""
	}
	if !nextURL.IsAbs() {
		nextURL = baseURL.ResolveReference(nextURL)
	}
	if (nextURL.Scheme != "http" && nextURL.Scheme != "https") || !strings.EqualFold(nextURL.Host, baseURL.Host) {
		return ""
	}
	return nextURL.String()
}

func escapeQuery(value string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(strings.TrimSpace(value))
}

func gutendexLanguage(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, pair := range [][2]string{{"english", "en"}, {"eng", "en"}, {"french", "fr"}, {"fre", "fr"}, {"fra", "fr"}, {"german", "de"}, {"ger", "de"}, {"deu", "de"}, {"spanish", "es"}, {"spa", "es"}, {"italian", "it"}, {"ita", "it"}, {"portuguese", "pt"}, {"por", "pt"}} {
		if value == pair[0] {
			return pair[1]
		}
	}
	return value
}

func locLanguage(value string) string {
	code := gutendexLanguage(value)
	for _, pair := range [][2]string{{"en", "English"}, {"fr", "French"}, {"de", "German"}, {"es", "Spanish"}, {"it", "Italian"}, {"pt", "Portuguese"}} {
		if code == pair[0] {
			return pair[1]
		}
	}
	return strings.TrimSpace(value)
}

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	default:
		return ""
	}
}

type bookAccumulator struct {
	items []books.Book
	seen  map[string]int
}

func (a *bookAccumulator) add(book books.Book) {
	if a.seen == nil {
		a.seen = make(map[string]int)
	}
	key := strings.ToLower(book.Source.Provider + ":" + book.Source.ID)
	if book.Source.ID == "" {
		key = strings.ToLower(book.Title + ":" + strings.Join(book.Authors, ":"))
	}
	if index, ok := a.seen[key]; ok {
		current := &a.items[index]
		current.Authors = mergeStrings(current.Authors, book.Authors)
		current.Subjects = mergeStrings(current.Subjects, book.Subjects)
		current.Genres = mergeStrings(current.Genres, book.Genres)
		current.Languages = mergeStrings(current.Languages, book.Languages)
		current.ISBNs = mergeStrings(current.ISBNs, book.ISBNs)
		current.AccessURLs = mergeStrings(current.AccessURLs, book.AccessURLs)
		if current.Description == "" {
			current.Description = book.Description
		}
		if current.Year == nil {
			current.Year = book.Year
			current.YearKind = book.YearKind
		}
		if current.Popularity == nil {
			current.Popularity = book.Popularity
		}
		if current.Rating == nil {
			current.Rating = book.Rating
		}
		return
	}
	a.seen[key] = len(a.items)
	a.items = append(a.items, book)
}

func (a *bookAccumulator) all() []books.Book { return a.items }

func mergeStrings(left, right []string) []string {
	for _, value := range right {
		left = addUnique(left, value)
	}
	return left
}
