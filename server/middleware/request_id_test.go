package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDReusesValidHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID)
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, GetRequestID(c.Request.Context()))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(headerKey, "client-123")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if got := res.Header().Get(headerKey); got != "client-123" {
		t.Errorf("response request ID = %q, want client-123", got)
	}
	if got := res.Body.String(); got != "client-123" {
		t.Errorf("context request ID = %q, want client-123", got)
	}
}

func TestRequestIDReplacesInvalidHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID)
	r.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(headerKey, strings.Repeat("x", maxRequestIDLength+1))
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	id := res.Header().Get(headerKey)
	if id == "" || len(id) != 32 || !validRequestID(id) {
		t.Errorf("generated request ID = %q, want a 32-character safe ID", id)
	}
}
