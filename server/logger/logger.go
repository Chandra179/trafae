package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
)

func NewLogger(appEnvironment, level string, samplingInitial, samplingThereafter int) (*zap.Logger, error) {
	var cfg zap.Config
	if appEnvironment == "dev" {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
		cfg.Sampling = &zap.SamplingConfig{
			Initial:    samplingInitial,
			Thereafter: samplingThereafter,
		}
	}

	if err := cfg.Level.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(level)))); err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", level, err)
	}

	return cfg.Build()
}
