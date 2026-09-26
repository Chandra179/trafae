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

	"github.com/Chandra179/lux/server/config"
	"github.com/Chandra179/lux/server/internal/example"
	"github.com/Chandra179/lux/server/logger"
	"github.com/Chandra179/lux/server/middleware"
	"github.com/Chandra179/lux/server/router"
	"github.com/Chandra179/lux/server/store"
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
	middlewareDeps := middleware.NewDependencies(log)

	engine := router.NewDependencies(&router.DependenciesConfig{
		Logger:           log,
		RequestLog:       middlewareDeps.RequestLog(cfg.Middleware.RequestLog),
		RequestBodyLimit: middleware.RequestBodyLimit(cfg.HTTP.MaxBodySizeInBytes),
		Readiness:        readinessHandler(db, badgerDB),
		Example:          exampleDeps.HandleExample,
	}).New()

	httpServer := &http.Server{
		Addr:         listenAddress(cfg.HTTP.Port),
		Handler:      engine,
		ReadTimeout:  seconds(cfg.HTTP.ReadTimeoutInSec),
		WriteTimeout: seconds(cfg.HTTP.WriteTimeoutInSec),
		IdleTimeout:  seconds(cfg.HTTP.IdleTimeoutInSec),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
