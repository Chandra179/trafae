package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSnapshotCountsSearchesProvidersAndClicks(t *testing.T) {
	t.Parallel()

	m := New()
	m.RecordSearch(false)
	m.RecordSearch(true)
	m.RecordProviderStatus("gutendex", "ok")
	m.RecordProviderStatus("gutendex", "ok")
	m.RecordProviderStatus("doab", "error")
	m.RecordAccessClick("gutendex")
	m.RecordAccessClick("")

	data, err := json.Marshal(m.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	snapshot := string(data)
	for _, want := range []string{
		`"searches_total":1`,
		`"searches_cached_total":1`,
		`"provider_status":{"doab":{"error":1},"gutendex":{"ok":2}}`,
		`"access_clicks":{"gutendex":{"clicks":1},"unknown":{"clicks":1}}`,
	} {
		if !strings.Contains(snapshot, want) {
			t.Errorf("snapshot %s missing %s", snapshot, want)
		}
	}
}

func TestEventHandlerAcceptsClicksAndRejectsJunk(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	m := New()
	router := gin.New()
	router.POST("/books/events", m.EventHandler())

	post := func(body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/books/events", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		return recorder
	}

	if got := post(`{"type":"access_click","provider":"gutendex","topic":"history"}`); got.Code != http.StatusAccepted {
		t.Fatalf("valid event: status = %d, want 202", got.Code)
	}
	if got := post(`{"type":"pageview"}`); got.Code != http.StatusBadRequest {
		t.Fatalf("unsupported event type: status = %d, want 400", got.Code)
	}
	if got := post(`not json`); got.Code != http.StatusBadRequest {
		t.Fatalf("malformed payload: status = %d, want 400", got.Code)
	}

	data, err := json.Marshal(m.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	snapshot := string(data)
	if !strings.Contains(snapshot, `"access_clicks":{"gutendex":{"clicks":1}}`) {
		t.Errorf("snapshot %s missing the accepted click", snapshot)
	}
	if !strings.Contains(snapshot, `"events_rejected_total":2`) {
		t.Errorf("snapshot %s missing the two rejected events", snapshot)
	}
}

func TestNilMetricsIsNoop(t *testing.T) {
	t.Parallel()

	var m *Metrics
	m.RecordSearch(false)
	m.RecordProviderStatus("gutendex", "ok")
	m.RecordAccessClick("gutendex")
	m.RecordRejectedEvent()

	data, err := json.Marshal(m.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"searches_total":0`) {
		t.Errorf("nil metrics snapshot %s should be all zeros", data)
	}
}
