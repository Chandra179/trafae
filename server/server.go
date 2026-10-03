package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Chandra179/trafae/server/config"
	"github.com/Chandra179/trafae/server/internal/books"
	"github.com/Chandra179/trafae/server/internal/example"
	"github.com/Chandra179/trafae/server/internal/providers"
	"github.com/Chandra179/trafae/server/logger"
	"github.com/Chandra179/trafae/server/middleware"
	"github.com/Chandra179/trafae/server/router"
	"github.com/Chandra179/trafae/server/store"
)

// RunHttpServer starts the configured HTTP server and blocks until it receives
// a termination signal or the server fails.
func RunHttpServer() {
	if err := runHTTPServer(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runHTTPServer() error {
	appEnvironment := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENVIRONMENT")))
	if appEnvironment == "" {
		appEnvironment = "dev"
	}

	configPath := filepath.Join("server", "config", "config_"+appEnvironment+".yaml")
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config %q: %w", configPath, err)
	}

	log, err := logger.NewLogger(
		appEnvironment,
		cfg.Logger.Level,
		cfg.Logger.SamplingInitial,
		cfg.Logger.SamplingThereafter,
	)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer func() { _ = log.Sync() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.NewSQLite(context.Background(), cfg.SQLite.DSN)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	badgerDB, err := store.NewBadger(cfg.Badger.Dir)
	if err != nil {
		return err
	}
	defer func() { _ = badgerDB.Close() }()

	exampleDeps := example.NewDependencies(&example.DependenciesConfig{
		Logger: log,
		DB:     db,
	})
	// Provider calls are bounded by the per-provider context deadlines from
	// providers.search_timeouts_in_second, not by a client-level Timeout —
	// http.Client.Timeout also covers reading the body, so it would silently
	// cap those configured budgets.
	providerDeps := providers.NewDependencies(&providers.DependenciesConfig{
		Logger:              log,
		DB:                  db,
		HTTPClient:          &http.Client{},
		OpenLibraryEmail:    cfg.Providers.OpenLibraryContactEmail,
		GutenbergCatalogURL: cfg.Providers.GutenbergCatalogURL,
	})
	providerDeps.Start(ctx)
	searchTimeouts := make(map[string]time.Duration, len(cfg.Providers.SearchTimeoutsInSec))
	for id, timeoutInSec := range cfg.Providers.SearchTimeoutsInSec {
		searchTimeouts[id] = seconds(timeoutInSec)
	}
	booksDeps := books.NewDependencies(&books.DependenciesConfig{
		Logger:         log,
		Providers:      providerDeps.All(),
		DefaultGenre:   cfg.Books.DefaultGenre,
		DefaultLimit:   cfg.Books.DefaultLimit,
		SearchTimeout:  seconds(cfg.Providers.SearchTimeoutInSec),
		SearchTimeouts: searchTimeouts,
	})
	middlewareDeps := middleware.NewDependencies(log)

	engine := router.NewDependencies(&router.DependenciesConfig{
		Logger:           log,
		RequestLog:       middlewareDeps.RequestLog(cfg.Middleware.RequestLog),
		RequestBodyLimit: middleware.RequestBodyLimit(cfg.HTTP.MaxBodySizeInBytes),
		Readiness:        readinessHandler(db, badgerDB),
		Example:          exampleDeps.HandleExample,
		BookSearch:       booksDeps.HandleSearch,
		BookProviders:    booksDeps.HandleProviders,
	}).New()

	httpServer := &http.Server{
		Addr:         listenAddress(cfg.HTTP.Port),
		Handler:      engine,
		ReadTimeout:  seconds(cfg.HTTP.ReadTimeoutInSec),
		WriteTimeout: seconds(cfg.HTTP.WriteTimeoutInSec),
		IdleTimeout:  seconds(cfg.HTTP.IdleTimeoutInSec),
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			seconds(cfg.HTTP.ShutdownTimeoutInSec),
		)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		return nil
	}
}

type readinessStore interface {
	IsClosed() bool
}

func readinessHandler(db *sql.DB, badgerDB readinessStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := db.PingContext(c.Request.Context()); err != nil || badgerDB.IsClosed() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}

func listenAddress(port string) string {
	port = strings.TrimSpace(port)
	if strings.Contains(port, ":") {
		return port
	}
	return ":" + port
}

func seconds(value int) time.Duration {
	return time.Duration(value) * time.Second
}
