package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandlerExportsMetrics(t *testing.T) {
	collector := New()

	collector.Record(http.StatusOK, 10*time.Millisecond)
	collector.Record(http.StatusInternalServerError, 20*time.Millisecond)
	collector.Record(http.StatusTooManyRequests, 30*time.Millisecond)

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)

	recorder := httptest.NewRecorder()

	handler := collector.Handler()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	contentType := recorder.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Fatalf("expected Content-Type to contain text/plain, got %q", contentType)
	}

	body := recorder.Body.String()

	if !strings.Contains(body, "gateforge_requests_total 3") {
		t.Fatalf("expected total requests metric, got:\n%s", body)
	}

	if !strings.Contains(body, "gateforge_errors_5xx_total 1") {
		t.Fatalf("expected 5xx errors metric, got:\n%s", body)
	}

	if !strings.Contains(body, "gateforge_rate_limited_requests_total 1") {
		t.Fatalf("expected rate-limited metric, got:\n%s", body)
	}

	if !strings.Contains(body, "gateforge_request_latency_nanoseconds_total 60000000") {
		t.Fatalf("expected total latency metric, got:\n%s", body)
	}
}
