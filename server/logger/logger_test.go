package logger

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewLoggerHonorsConfiguredLevel(t *testing.T) {
	log, err := NewLogger("prd", "error", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = log.Sync() }()

	if log.Check(zap.InfoLevel, "not emitted") != nil {
		t.Fatal("info log was enabled at error level")
	}
	if log.Check(zap.ErrorLevel, "emitted") == nil {
		t.Fatal("error log was disabled at error level")
	}
}

func TestNewLoggerRejectsInvalidLevel(t *testing.T) {
	if _, err := NewLogger("prd", "verbose", 1, 1); err == nil {
		t.Fatal("NewLogger() accepted an invalid level")
	}
}
