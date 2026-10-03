package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newRateLimitRouter(t *testing.T, rps float64, burst int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		t.Fatal(err)
	}
	router.Use(RateLimit(rps, burst))
	router.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return router
}

func TestRateLimitAllowsBurstThenBlocks(t *testing.T) {
	t.Parallel()

	// A negligible refill rate keeps the test deterministic: exactly burst
	// requests pass, everything after is throttled.
	router := newRateLimitRouter(t, 0.001, 2)
	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ping", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i+1, recorder.Code)
		}
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("third request: status = %d, want 429", recorder.Code)
	}
	if recorder.Header().Get("Retry-After") == "" {
		t.Fatal("429 response is missing the Retry-After header")
	}
}

func TestRateLimitIgnoresSpoofedForwardedFor(t *testing.T) {
	t.Parallel()

	router := newRateLimitRouter(t, 0.001, 1)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ping", nil)
	request.Header.Set("X-Forwarded-For", "203.0.113.7")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("first request: status = %d, want 200", recorder.Code)
	}

	// A "different" client via a spoofed header over the same connection must
	// share the bucket — otherwise rotating the header defeats the limit.
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/ping", nil)
	request.Header.Set("X-Forwarded-For", "198.51.100.9")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("spoofed request: status = %d, want 429", recorder.Code)
	}
}
