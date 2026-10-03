package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigValidateRejectsMissingRequiredValues(t *testing.T) {
	t.Parallel()

	cfg := Config{
		HTTP: HTTPConfig{
			Port:                 "8080",
			ReadTimeoutInSec:     1,
			WriteTimeoutInSec:    1,
			IdleTimeoutInSec:     1,
			ShutdownTimeoutInSec: 1,
			MaxBodySizeInBytes:   1,
		},
		Logger: LoggerConfig{Level: "info", SamplingInitial: 1, SamplingThereafter: 1},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil for an invalid config")
	}
	for _, want := range []string{"sqlite.dsn is required"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Validate() error = %q, want %q", err, want)
		}
	}
}

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	contents := []byte(`
http:
  port: "8080"
  read_timeout_in_second: 1
  write_timeout_in_second: 1
  idle_timeout_in_second: 1
  shutdown_timeout_in_second: 1
  max_body_size_in_bytes: 1024
logger:
  level: info
  sampling_initial: 1
  sampling_thereafter: 1
sqlite:
  dsn: file.yaml
`)
	if err := os.WriteFile(configPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SQLITE_DSN", "file.env")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SQLite.DSN != "file.env" {
		t.Errorf("SQLite.DSN = %q, want file.env", cfg.SQLite.DSN)
	}
}

func TestLoadRejectsExplicitlyEmptyRequiredOverride(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	contents := []byte(`
http:
  port: "8080"
  read_timeout_in_second: 1
  write_timeout_in_second: 1
  idle_timeout_in_second: 1
  shutdown_timeout_in_second: 1
  max_body_size_in_bytes: 1024
logger:
  level: info
  sampling_initial: 1
  sampling_thereafter: 1
sqlite:
  dsn: file.yaml
`)
	if err := os.WriteFile(configPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SQLITE_DSN", "")

	if _, err := Load(configPath); err == nil {
		t.Fatal("Load() returned nil for an explicitly empty required override")
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	contents := []byte(`
http:
  port: "8080"
  read_timeout_in_second: 1
  write_timeout_in_second: 1
  idle_timeout_in_second: 1
  shutdown_timeout_in_second: 1
  max_body_size_in_bytes: 1024
logger:
  level: info
  sampling_initial: 1
  sampling_thereafter: 1
sqlite:
  dsn: file.yaml
unexpected: true
`)
	if err := os.WriteFile(configPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(configPath); err == nil {
		t.Fatal("Load() accepted an unknown config field")
	}
}
