package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Logger     LoggerConfig     `yaml:"logger"`
	SQLite     SQLiteConfig     `yaml:"sqlite"`
	Badger     BadgerConfig     `yaml:"badger"`
	Middleware MiddlewareConfig `yaml:"middleware"`
	HTTP       HTTPConfig       `yaml:"http"`
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
}

type RequestLogConfig struct {
	SkipPaths      []string `yaml:"skip_paths"`
	QueryAllowlist []string `yaml:"query_allowlist"`
	LogQuery       bool     `yaml:"log_query"`
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

	if len(problems) > 0 {
		return fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	return nil
}
