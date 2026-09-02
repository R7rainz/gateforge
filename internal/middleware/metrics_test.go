package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/r7rainz/gateforge/internal/metrics"
)

func TestMetricsRecords200(t *testing.T) {
	m := metrics.New()

	handler := Metrics(m)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/users",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	snapshot := m.Snapshot()

	if snapshot.TotalRequests != 1 {
		t.Fatalf(
			"expected 1 request, got %d",
			snapshot.TotalRequests,
		)
	}

	if snapshot.Errors5xx != 0 {
		t.Fatalf(
			"expected 0 5xx errors, got %d",
			snapshot.Errors5xx,
		)
	}

	if snapshot.RateLimitedRequests != 0 {
		t.Fatalf(
			"expected 0 rate-limited requests, got %d",
			snapshot.RateLimitedRequests,
		)
	}

	if snapshot.TotalLatencyNanos == 0 {
		t.Fatal("expected latency to be recorded")
	}
}

func TestMetricsRecords429(t *testing.T) {
	m := metrics.New()

	handler := Metrics(m)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/users",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	snapshot := m.Snapshot()

	if snapshot.TotalRequests != 1 {
		t.Fatalf(
			"expected 1 request, got %d",
			snapshot.TotalRequests,
		)
	}

	if snapshot.RateLimitedRequests != 1 {
		t.Fatalf(
			"expected 1 rate-limited request, got %d",
			snapshot.RateLimitedRequests,
		)
	}

	if snapshot.Errors5xx != 0 {
		t.Fatalf(
			"expected 0 5xx errors, got %d",
			snapshot.Errors5xx,
		)
	}
}

func TestMetricsRecords500(t *testing.T) {
	m := metrics.New()

	handler := Metrics(m)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/users",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	snapshot := m.Snapshot()

	if snapshot.TotalRequests != 1 {
		t.Fatalf(
			"expected 1 request, got %d",
			snapshot.TotalRequests,
		)
	}

	if snapshot.Errors5xx != 1 {
		t.Fatalf(
			"expected 1 5xx error, got %d",
			snapshot.Errors5xx,
		)
	}

	if snapshot.RateLimitedRequests != 0 {
		t.Fatalf(
			"expected 0 rate-limited requests, got %d",
			snapshot.RateLimitedRequests,
		)
	}
}
