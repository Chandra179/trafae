package middleware

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Chandra179/lux/server/config"
)

func TestRequestLogIncludesCorrelationAndSanitizesQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&output),
		zap.InfoLevel,
	)
	deps := NewDependencies(zap.New(core))

	r := gin.New()
	r.Use(RequestID, deps.RequestLog(config.RequestLogConfig{
		LogQuery:       true,
		QueryAllowlist: []string{"page", "token"},
	}))
	r.GET("/items", func(c *gin.Context) {
		_ = c.Error(errors.New("expected failure"))
		c.Status(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/items?page=2&token=secret&ignored=value", nil)
	req.Header.Set(headerKey, "request-123")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	logLine := output.String()
	for _, want := range []string{"request-123", "/items", "expected failure", "page=2", "REDACTED"} {
		if !strings.Contains(logLine, want) {
			t.Errorf("log = %q, missing %q", logLine, want)
		}
	}
	if strings.Contains(logLine, "secret") || strings.Contains(logLine, "ignored=value") {
		t.Errorf("log leaked query data: %q", logLine)
	}
}
