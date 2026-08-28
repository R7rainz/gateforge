package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestLog_GenerateRequestID(t *testing.T) {
	var logs bytes.Buffer

	logger := slog.New(
		slog.NewTextHandler(&logs, nil),
	)

	handler := RequestLog(logger)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	requestID := rec.Header().Get("X-Request-ID")

	if requestID == "" {
		t.Fatal("expected generated X-Request-ID, got empty value")
	}
}

func TestRequestLog_PreservesRequestID(t *testing.T) {
	var logs bytes.Buffer

	logger := slog.New(
		slog.NewTextHandler(&logs, nil),
	)

	handler := RequestLog(logger)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/users/",
		nil,
	)

	req.Header.Set("X-Request-ID", "test-request-123")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	requestID := rec.Header().Get("X-Request-ID")

	if requestID != "test-request-123" {
		t.Fatalf("expected request ID %q, got %q", "test-request-123", requestID)
	}
}

func TestRequestLog_CapturesStatusAndLatency(t *testing.T) {
	var logs bytes.Buffer

	logger := slog.New(
		slog.NewTextHandler(&logs, nil),
	)

	handler := RequestLog(logger)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusCreated)
		}),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/users/",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			rec.Code,
		)
	}

	output := logs.String()

	if !strings.Contains(output, "status=201") {
		t.Fatalf(
			"expected logged status=201, got:\n%s",
			output,
		)
	}

	if !strings.Contains(output, "latency") {
		t.Fatalf(
			"expected latency in log output, got:\n%s",
			output,
		)
	}
}
