package middleware

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Chandra179/trafae/server/config"
)

// RequestLog returns Gin middleware that logs one canonical line per HTTP
// request. Level follows the response status (Info 2xx/3xx, Warn 4xx,
// Error 5xx); a handler's c.Error(err) is attached for 4xx/5xx. Request
// and response bodies are never logged. Details: docs/logging.md.
func (d *dependencies) RequestLog(cfg config.RequestLogConfig) gin.HandlerFunc {
	queryAllowlist := make(map[string]struct{}, len(cfg.QueryAllowlist))
	for _, key := range cfg.QueryAllowlist {
		queryAllowlist[key] = struct{}{}
	}

	return func(c *gin.Context) {
		if slices.Contains(cfg.SkipPaths, c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		route := c.FullPath()
		if route == "" {
			route = "<unmatched>"
		}

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", route),
			zap.Int("status", status),
			zap.Int64("duration_ms", duration.Milliseconds()),
		}

		if reqID := GetRequestID(c.Request.Context()); reqID != "" {
			fields = append(fields, zap.String("request_id", reqID))
		}

		if cfg.LogQuery {
			if query := queryForLog(c.Request.URL.Query(), queryAllowlist); query != "" {
				fields = append(fields, zap.String("query", query))
			}
		}

		switch {
		case status >= http.StatusInternalServerError:
			d.logger.Error("request completed", withLastError(c, fields)...)
		case status >= http.StatusBadRequest:
			d.logger.Warn("request completed", withLastError(c, fields)...)
		default:
			d.logger.Info("request completed", fields...)
		}
	}
}

func queryForLog(query url.Values, allowlist map[string]struct{}) string {
	filtered := make(url.Values)
	for key, values := range query {
		if _, allowed := allowlist[key]; !allowed {
			continue
		}

		for _, value := range values {
			if isSensitiveQueryKey(key) {
				value = "[REDACTED]"
			}
			filtered.Add(key, value)
		}
	}

	return filtered.Encode()
}

func isSensitiveQueryKey(key string) bool {
	key = strings.ToLower(key)
	for _, fragment := range []string{"auth", "code", "key", "password", "secret", "signature", "state", "token"} {
		if strings.Contains(key, fragment) {
			return true
		}
	}
	return false
}

// withLastError appends the last error recorded via c.Error(err) to fields,
// if any. Only called for 4xx/5xx responses; 2xx/3xx never carry an error
// field.
func withLastError(c *gin.Context, fields []zap.Field) []zap.Field {
	if err := c.Errors.Last(); err != nil {
		return append(fields, zap.Error(err.Err))
	}
	return fields
}
