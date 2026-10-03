package books

import (
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Chandra179/trafae/server/internal/metrics"
)

type DependenciesConfig struct {
	Logger       *zap.Logger
	Providers    []Provider
	DefaultGenre string
	DefaultLimit int
	// SearchTimeout bounds each provider call; SearchTimeouts overrides it per
	// provider capability ID.
	SearchTimeout  time.Duration
	SearchTimeouts map[string]time.Duration
	// SearchCacheTTL serves identical successful searches from memory for
	// that long; zero disables the cache.
	SearchCacheTTL time.Duration
	// Metrics receives the launch funnel counters; nil disables recording.
	Metrics *metrics.Metrics
}

type dependencies struct {
	logger         *zap.Logger
	providers      []Provider
	byID           map[string]Provider
	defaultGenre   string
	defaultLimit   int
	searchTimeout  time.Duration
	searchTimeouts map[string]time.Duration
	cache          *searchCache
	metrics        *metrics.Metrics
}

func NewDependencies(cfg *DependenciesConfig) *dependencies {
	if cfg == nil {
		cfg = &DependenciesConfig{}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	defaultGenre := strings.TrimSpace(cfg.DefaultGenre)
	if defaultGenre == "" {
		defaultGenre = DefaultGenre
	}
	defaultLimit := cfg.DefaultLimit
	if defaultLimit <= 0 {
		defaultLimit = DefaultResultLimit
	}
	searchTimeout := cfg.SearchTimeout
	if searchTimeout <= 0 {
		searchTimeout = DefaultSearchTimeout
	}
	searchTimeouts := make(map[string]time.Duration, len(cfg.SearchTimeouts))
	for id, timeout := range cfg.SearchTimeouts {
		if timeout <= 0 {
			continue
		}
		searchTimeouts[strings.ToLower(strings.TrimSpace(id))] = timeout
	}
	deps := &dependencies{
		logger:         logger,
		providers:      append([]Provider(nil), cfg.Providers...),
		byID:           make(map[string]Provider, len(cfg.Providers)),
		defaultGenre:   defaultGenre,
		defaultLimit:   defaultLimit,
		searchTimeout:  searchTimeout,
		searchTimeouts: searchTimeouts,
		cache:          newSearchCache(cfg.SearchCacheTTL),
		metrics:        cfg.Metrics,
	}
	for _, provider := range deps.providers {
		id := strings.ToLower(strings.TrimSpace(provider.Capabilities().ID))
		if id != "" {
			deps.byID[id] = provider
		}
	}
	return deps
}
