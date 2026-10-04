package server

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestReadinessHandler(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/ready", nil)

	readinessHandler(db)(context)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestListenAddress(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		port string
		want string
	}{
		{port: "8081", want: ":8081"},
		{port: ":8081", want: ":8081"},
		{port: "127.0.0.1:8081", want: "127.0.0.1:8081"},
	} {
		if got := listenAddress(tt.port); got != tt.want {
			t.Errorf("listenAddress(%q) = %q, want %q", tt.port, got, tt.want)
		}
	}
}
