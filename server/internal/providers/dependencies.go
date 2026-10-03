package providers

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Chandra179/trafae/server/internal/books"
)

type DependenciesConfig struct {
	Logger     *zap.Logger
	DB         *sql.DB
	HTTPClient *http.Client
	// GutenbergFeedClient downloads the daily catalog feed; when nil a
	// dedicated client with a long timeout is built, since the download is
	// far larger than any per-search call.
	GutenbergFeedClient      *http.Client
	OpenLibraryEmail         string
	OpenLibraryBaseURL       string
	DOABBaseURL              string
	GutendexBaseURL          string
	InternetArchiveBaseURL   string
	LibraryOfCongressBaseURL string
	WikidataBaseURL          string
	GutenbergCatalogURL      string
}

type dependencies struct {
	logger    *zap.Logger
	client    *http.Client
	providers []books.Provider
	gutenberg *projectGutenbergProvider
}

func NewDependencies(cfg *DependenciesConfig) *dependencies {
	if cfg == nil {
		cfg = &DependenciesConfig{}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	client := defaultClient(cfg.HTTPClient)
	openLibrary := newOpenLibraryProvider(client, baseOr(cfg.OpenLibraryBaseURL, "https://openlibrary.org"), cfg.OpenLibraryEmail)
	doab := newDOABProvider(client, baseOr(cfg.DOABBaseURL, "https://directory.doabooks.org/rest/api/discover/search/objects"))
	gutendex := newGutendexProvider(client, baseOr(cfg.GutendexBaseURL, "https://gutendex.com/books"))
	internetArchive := newInternetArchiveProvider(client, baseOr(cfg.InternetArchiveBaseURL, "https://archive.org/advancedsearch.php"))
	loc := newLibraryOfCongressProvider(client, baseOr(cfg.LibraryOfCongressBaseURL, "https://www.loc.gov/books/"))
	var pg *projectGutenbergProvider
	if cfg.DB != nil {
		feedClient := cfg.GutenbergFeedClient
		if feedClient == nil {
			feedClient = &http.Client{Timeout: gutenbergFeedTimeout}
		}
		pg = newProjectGutenbergProvider(cfg.DB, feedClient, baseOr(cfg.GutenbergCatalogURL, "https://www.gutenberg.org/cache/epub/feeds/pg_catalog.csv.gz"))
	}
	wikidata := newWikidataProvider(client, baseOr(cfg.WikidataBaseURL, "https://www.wikidata.org/w/api.php"))
	providers := []books.Provider{openLibrary, doab, gutendex, internetArchive, loc, wikidata}
	if pg != nil {
		providers = append(providers, pg)
	}
	return &dependencies{logger: logger, client: client, providers: providers, gutenberg: pg}
}

func (d *dependencies) All() []books.Provider { return append([]books.Provider(nil), d.providers...) }

// Start begins the daily refresh loop for the official Project Gutenberg catalog.
func (d *dependencies) Start(ctx context.Context) {
	if d.gutenberg == nil {
		return
	}
	go func() {
		if err := d.gutenberg.refreshIfNeeded(ctx); err != nil && ctx.Err() == nil {
			d.logger.Warn("initial Project Gutenberg catalog refresh failed", zap.Error(err))
		}
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := d.gutenberg.refresh(ctx); err != nil && ctx.Err() == nil {
					d.logger.Warn("Project Gutenberg catalog refresh failed", zap.Error(err))
				}
			}
		}
	}()
}

func baseOr(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}
