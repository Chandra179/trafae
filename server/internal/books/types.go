package books

import (
	"context"
	"strings"
	"time"
)

const (
	FilterTopics       = "topics"
	FilterGenre        = "genre"
	FilterYear         = "year"
	FilterLanguage     = "language"
	FilterPopularity   = "popularity"
	FilterRating       = "rating"
	DefaultGenre       = "non-fiction"
	DefaultResultLimit = 20
	MaxResultLimit     = 50
	MaxSearchTopics    = 8
	MaxSearchPage      = 20
	// MaxParamLength bounds free-text query parameters (topic, genre, language,
	// provider) before they are interpolated into upstream URLs and SQL patterns.
	MaxParamLength = 100
	// DefaultSearchTimeout bounds each provider call when the wiring does not
	// supply providers.search_timeout_in_second.
	DefaultSearchTimeout = 30 * time.Second
)

// SearchRequest is the normalized query passed to every provider.
type SearchRequest struct {
	Topics        []string
	Genre         string
	Providers     []string
	MinYear       *int
	MaxYear       *int
	Language      string
	MinPopularity *float64
	MinRating     *float64
	Limit         int
	Page          int
}

// ProviderCapability describes data and filters a provider can honor.
type ProviderCapability struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Filters          []string `json:"filters"`
	PopularityMetric string   `json:"popularity_metric,omitempty"`
	RatingScale      float64  `json:"rating_scale,omitempty"`
	Notes            []string `json:"notes,omitempty"`
}

// Provider is the boundary between book discovery and external catalogs.
type Provider interface {
	Capabilities() ProviderCapability
	Search(context.Context, SearchRequest) ([]Book, error)
}

type Book struct {
	ID          string     `json:"id,omitempty"`
	Title       string     `json:"title"`
	Authors     []string   `json:"authors,omitempty"`
	Description string     `json:"description,omitempty"`
	Year        *int       `json:"year,omitempty"`
	YearKind    string     `json:"year_kind,omitempty"`
	Genres      []string   `json:"genres,omitempty"`
	Subjects    []string   `json:"subjects,omitempty"`
	Languages   []string   `json:"languages,omitempty"`
	ISBNs       []string   `json:"isbns,omitempty"`
	URL         string     `json:"url,omitempty"`
	CoverURL    string     `json:"cover_url,omitempty"`
	License     string     `json:"license,omitempty"`
	AccessURLs  []string   `json:"access_urls,omitempty"`
	Popularity  *Metric    `json:"popularity,omitempty"`
	Rating      *Rating    `json:"rating,omitempty"`
	Source      BookSource `json:"source"`
}

type Metric struct {
	Value  float64 `json:"value"`
	Metric string  `json:"metric"`
}

type Rating struct {
	Value float64 `json:"value"`
	Scale float64 `json:"scale"`
	Count int     `json:"count,omitempty"`
}

type BookSource struct {
	Provider   string  `json:"provider"`
	ID         string  `json:"id,omitempty"`
	URL        string  `json:"url,omitempty"`
	Popularity *Metric `json:"popularity,omitempty"`
	Rating     *Rating `json:"rating,omitempty"`
}

type SearchResult struct {
	Book     Book         `json:"book"`
	Sources  []BookSource `json:"sources"`
	RRFScore float64      `json:"rrf_score"`
}

type ProviderStatus struct {
	Provider string `json:"provider"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
	Count    int    `json:"count,omitempty"`
}

type SearchResponse struct {
	Results   []SearchResult   `json:"results"`
	Providers []ProviderStatus `json:"providers"`
	Limit     int              `json:"limit"`
	Page      int              `json:"page"`
	HasMore   bool             `json:"has_more"`
}

func (r SearchRequest) RequiredFilters() []string {
	filters := make([]string, 0, 6)
	if len(r.Topics) > 0 {
		filters = append(filters, FilterTopics)
	}
	if strings.TrimSpace(r.Genre) != "" {
		filters = append(filters, FilterGenre)
	}
	if r.MinYear != nil || r.MaxYear != nil {
		filters = append(filters, FilterYear)
	}
	if strings.TrimSpace(r.Language) != "" {
		filters = append(filters, FilterLanguage)
	}
	if r.MinPopularity != nil {
		filters = append(filters, FilterPopularity)
	}
	if r.MinRating != nil {
		filters = append(filters, FilterRating)
	}
	return filters
}
