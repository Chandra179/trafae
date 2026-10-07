package providers

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/Chandra179/trafae/server/internal/books"
)

type gutendexProvider struct {
	client   *http.Client
	endpoint string
}

func newGutendexProvider(client *http.Client, endpoint string) *gutendexProvider {
	return &gutendexProvider{client: client, endpoint: endpoint}
}

func (p *gutendexProvider) Capabilities() books.ProviderCapability {
	return books.ProviderCapability{ID: "gutendex", Name: "Gutendex", Filters: []string{books.FilterTopics, books.FilterGenre, books.FilterLanguage, books.FilterPopularity}, PopularityMetric: "downloads", Notes: []string{"Publication year is not included in the catalog response."}}
}

func (p *gutendexProvider) Search(ctx context.Context, request books.SearchRequest) ([]books.Book, error) {
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

func (p *gutendexProvider) searchTerm(ctx context.Context, request books.SearchRequest, term string) ([]books.Book, error) {
	var found bookAccumulator
	target := withLimit(request, 200)
	maxPages := 8
	values := url.Values{}
	if term != "" {
		values.Set("topic", term)
	}
	values.Set("sort", "popular")
	if request.Language != "" {
		values.Set("languages", gutendexLanguage(request.Language))
	}
	pageURL := p.endpoint + "?" + values.Encode()
	for page, collected := 0, 0; page < maxPages && collected < target; page++ {
		var response struct {
			Next    string `json:"next"`
			Results []struct {
				ID      int    `json:"id"`
				Title   string `json:"title"`
				Authors []struct {
					Name string `json:"name"`
				} `json:"authors"`
				Summaries     []string          `json:"summaries"`
				Subjects      []string          `json:"subjects"`
				Bookshelves   []string          `json:"bookshelves"`
				Languages     []string          `json:"languages"`
				DownloadCount int64             `json:"download_count"`
				Formats       map[string]string `json:"formats"`
			} `json:"results"`
		}
		if err := requestJSON(ctx, p.client, pageURL, nil, "TrafaeBookDiscovery/1.0", &response); err != nil {
			return nil, err
		}
		for _, item := range response.Results {
			book := books.Book{ID: fmt.Sprint(item.ID), Title: item.Title, Subjects: append(item.Subjects, item.Bookshelves...), Languages: item.Languages,
				Popularity: &books.Metric{Value: float64(item.DownloadCount), Metric: "downloads"}, URL: fmt.Sprintf("https://www.gutenberg.org/ebooks/%d", item.ID)}
			for _, author := range item.Authors {
				book.Authors = append(book.Authors, author.Name)
			}
			if len(item.Summaries) > 0 {
				book.Description = item.Summaries[0]
			}
			book.Genres = append([]string(nil), book.Subjects...)
			book.AccessURLs = formatURLs(item.Formats)
			book.Source = books.BookSource{Provider: "gutendex", ID: book.ID, URL: book.URL}
			if found.add(book) {
				collected++
			}
		}
		nextURL := safeNextURL(p.endpoint, response.Next)
		if nextURL == "" {
			break
		}
		pageURL = nextURL
	}
	return found.all(), nil
}

func formatURLs(formats map[string]string) []string {
	priority := map[string]int{
		"text/html": 0, "application/xhtml+xml": 0,
		"application/pdf": 1, "text/plain": 2,
		"application/epub+zip": 3, "application/x-mobipocket-ebook": 4,
	}
	type bookFormat struct {
		url      string
		priority int
	}
	readable := make([]bookFormat, 0, len(formats))
	for contentType, value := range formats {
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			continue
		}
		rank, ok := priority[mediaType]
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		parsed, err := url.Parse(value)
		if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			continue
		}
		readable = append(readable, bookFormat{url: value, priority: rank})
	}
	sort.Slice(readable, func(i, j int) bool {
		if readable[i].priority != readable[j].priority {
			return readable[i].priority < readable[j].priority
		}
		return readable[i].url < readable[j].url
	})
	urls := make([]string, 0, len(readable))
	seen := make(map[string]struct{}, len(readable))
	for _, format := range readable {
		if _, exists := seen[format.url]; !exists {
			urls = append(urls, format.url)
			seen[format.url] = struct{}{}
		}
	}
	return urls
}
