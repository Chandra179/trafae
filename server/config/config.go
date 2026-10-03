package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Chandra179/trafae/server/internal/books"
)

type Config struct {
	Logger     LoggerConfig     `yaml:"logger"`
	SQLite     SQLiteConfig     `yaml:"sqlite"`
	Badger     BadgerConfig     `yaml:"badger"`
	Middleware MiddlewareConfig `yaml:"middleware"`
	HTTP       HTTPConfig       `yaml:"http"`
	Books      BooksConfig      `yaml:"books"`
	Providers  ProvidersConfig  `yaml:"providers"`
}

type BooksConfig struct {
	DefaultGenre string `yaml:"default_genre"`
	DefaultLimit int    `yaml:"default_limit"`
	// CacheTTLInSec serves identical successful searches from memory for this
	// long; zero disables the cache.
	CacheTTLInSec int `yaml:"cache_ttl_in_second"`
}

type ProvidersConfig struct {
	OpenLibraryContactEmail string         `yaml:"open_library_contact_email"`
	GutenbergCatalogURL     string         `yaml:"gutenberg_catalog_url"`
	SearchTimeoutInSec      int            `yaml:"search_timeout_in_second"`
	SearchTimeoutsInSec     map[string]int `yaml:"search_timeouts_in_second"`
}

type HTTPConfig struct {
	Port                 string `yaml:"port"`
	ReadTimeoutInSec     int    `yaml:"read_timeout_in_second"`
	WriteTimeoutInSec    int    `yaml:"write_timeout_in_second"`
	IdleTimeoutInSec     int    `yaml:"idle_timeout_in_second"`
	ShutdownTimeoutInSec int    `yaml:"shutdown_timeout_in_second"`
	MaxBodySizeInBytes   int64  `yaml:"max_body_size_in_bytes"`
}

type MiddlewareConfig struct {
	RequestLog RequestLogConfig `yaml:"request_log"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
}

type RequestLogConfig struct {
	SkipPaths      []string `yaml:"skip_paths"`
	QueryAllowlist []string `yaml:"query_allowlist"`
	LogQuery       bool     `yaml:"log_query"`
}

// RateLimitConfig throttles API requests per client address.
type RateLimitConfig struct {
	Enabled           bool    `yaml:"enabled"`
	RequestsPerSecond float64 `yaml:"requests_per_second"`
	Burst             int     `yaml:"burst"`
}

type LoggerConfig struct {
	Level              string `yaml:"level"`
	SamplingInitial    int    `yaml:"sampling_initial"`
	SamplingThereafter int    `yaml:"sampling_thereafter"`
}

type SQLiteConfig struct {
	DSN string `yaml:"dsn"`
}

type BadgerConfig struct {
	Dir string `yaml:"dir"`
}

// Environment returns the APP_ENVIRONMENT value, defaulting to dev.
func Environment() string {
	appEnvironment := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENVIRONMENT")))
	if appEnvironment == "" {
		appEnvironment = "dev"
	}
	return appEnvironment
}

// Path returns the config file path for the given environment.
func Path(appEnvironment string) string {
	return filepath.Join("server", "config", "config_"+appEnvironment+".yaml")
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.Books.DefaultGenre) == "" {
		cfg.Books.DefaultGenre = books.DefaultGenre
	}
	if cfg.Books.DefaultLimit == 0 {
		cfg.Books.DefaultLimit = books.DefaultResultLimit
	}
	if strings.TrimSpace(cfg.Providers.GutenbergCatalogURL) == "" {
		cfg.Providers.GutenbergCatalogURL = "https://www.gutenberg.org/cache/epub/feeds/pg_catalog.csv.gz"
	}
	if cfg.Providers.SearchTimeoutInSec == 0 {
		cfg.Providers.SearchTimeoutInSec = int(books.DefaultSearchTimeout.Seconds())
	}
	timeouts := make(map[string]int, len(cfg.Providers.SearchTimeoutsInSec))
	for id, timeout := range cfg.Providers.SearchTimeoutsInSec {
		timeouts[strings.ToLower(strings.TrimSpace(id))] = timeout
	}
	cfg.Providers.SearchTimeoutsInSec = timeouts

	if dsn, ok := os.LookupEnv("SQLITE_DSN"); ok {
		cfg.SQLite.DSN = dsn
	}
	if dir, ok := os.LookupEnv("BADGER_DIR"); ok {
		cfg.Badger.Dir = dir
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// Validate checks values that are required for the server to start safely.
// Environment-specific files provide safe operational defaults, while
// deployment-specific values such as datastore locations may be overridden by
// environment variables.
func (c Config) Validate() error {
	var problems []string

	if strings.TrimSpace(c.HTTP.Port) == "" {
		problems = append(problems, "http.port is required")
	}
	if c.HTTP.ReadTimeoutInSec <= 0 {
		problems = append(problems, "http.read_timeout_in_second must be greater than zero")
	}
	if c.HTTP.WriteTimeoutInSec <= 0 {
		problems = append(problems, "http.write_timeout_in_second must be greater than zero")
	}
	if c.HTTP.IdleTimeoutInSec <= 0 {
		problems = append(problems, "http.idle_timeout_in_second must be greater than zero")
	}
	if c.HTTP.ShutdownTimeoutInSec <= 0 {
		problems = append(problems, "http.shutdown_timeout_in_second must be greater than zero")
	}
	if c.HTTP.MaxBodySizeInBytes <= 0 {
		problems = append(problems, "http.max_body_size_in_bytes must be greater than zero")
	}
	if c.Books.DefaultGenre == "" {
		problems = append(problems, "books.default_genre is required")
	}
	if c.Books.DefaultLimit <= 0 || c.Books.DefaultLimit > 50 {
		problems = append(problems, "books.default_limit must be between 1 and 50")
	}
	if c.Providers.SearchTimeoutInSec <= 0 {
		problems = append(problems, "providers.search_timeout_in_second must be greater than zero")
	}
	providerIDs := make([]string, 0, len(c.Providers.SearchTimeoutsInSec))
	for id := range c.Providers.SearchTimeoutsInSec {
		providerIDs = append(providerIDs, id)
	}
	sort.Strings(providerIDs)
	for _, id := range providerIDs {
		if c.Providers.SearchTimeoutsInSec[id] <= 0 {
			problems = append(problems, fmt.Sprintf("providers.search_timeouts_in_second.%s must be greater than zero", id))
		}
	}
	if strings.TrimSpace(c.Providers.GutenbergCatalogURL) == "" {
		problems = append(problems, "providers.gutenberg_catalog_url is required")
	}

	level := strings.ToLower(strings.TrimSpace(c.Logger.Level))
	switch level {
	case "debug", "info", "warn", "error":
	default:
		problems = append(problems, "logger.level must be one of debug, info, warn, or error")
	}
	if c.Logger.SamplingInitial <= 0 {
		problems = append(problems, "logger.sampling_initial must be greater than zero")
	}
	if c.Logger.SamplingThereafter <= 0 {
		problems = append(problems, "logger.sampling_thereafter must be greater than zero")
	}

	if strings.TrimSpace(c.SQLite.DSN) == "" {
		problems = append(problems, "sqlite.dsn is required")
	}
	if strings.TrimSpace(c.Badger.Dir) == "" {
		problems = append(problems, "badger.dir is required")
	}
	if c.Middleware.RequestLog.LogQuery && len(c.Middleware.RequestLog.QueryAllowlist) == 0 {
		problems = append(problems, "middleware.request_log.query_allowlist is required when log_query is enabled")
	}
	if c.Middleware.RateLimit.Enabled {
		if c.Middleware.RateLimit.RequestsPerSecond <= 0 {
			problems = append(problems, "middleware.rate_limit.requests_per_second must be greater than zero when enabled")
		}
		if c.Middleware.RateLimit.Burst < 1 {
			problems = append(problems, "middleware.rate_limit.burst must be at least one when enabled")
		}
	}
	if c.Books.CacheTTLInSec < 0 {
		problems = append(problems, "books.cache_ttl_in_second must not be negative")
	}

	if len(problems) > 0 {
		return fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	return nil
}
