package server

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type readinessStoreStub struct {
	closed bool
}

func (s readinessStoreStub) IsClosed() bool {
	return s.closed
}

func TestReadinessHandler(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	tests := []struct {
		name       string
		store      readinessStoreStub
		wantStatus int
	}{
		{name: "ready", wantStatus: http.StatusOK},
		{name: "badger closed", store: readinessStoreStub{closed: true}, wantStatus: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest(http.MethodGet, "/ready", nil)

			readinessHandler(db, tt.store)(context)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
		})
	}
}

func TestListenAddress(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		port string
		want string
	}{
		{port: "8080", want: ":8080"},
		{port: ":8080", want: ":8080"},
		{port: "127.0.0.1:8080", want: "127.0.0.1:8080"},
	} {
		if got := listenAddress(tt.port); got != tt.want {
			t.Errorf("listenAddress(%q) = %q, want %q", tt.port, got, tt.want)
		}
	}
}
