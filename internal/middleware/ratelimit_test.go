package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/r7rainz/gateforge/internal/ratelimit"
)

func newTestRateLimitMiddleware(t *testing.T, limit int, window time.Duration) http.Handler {
	t.Helper()

	limiter, err := ratelimit.New(limit, window)
	if err != nil {
		t.Fatal(err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return RateLimit(limiter)(next)
}

func makeRequest(t *testing.T, handler http.Handler, remoteAddr string) int {
	t.Helper()

	req := httptest.NewRequest(
		http.MethodGet,
		"http://localhost/api/users",
		nil,
	)

	req.RemoteAddr = remoteAddr

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	return rec.Code
}

func TestRateLimitFirstTwoRequestsAllowed(t *testing.T) {
	handler := newTestRateLimitMiddleware(
		t,
		2,
		time.Second,
	)

	if status := makeRequest(
		t,
		handler,
		"127.0.0.1:54321",
	); status != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", status)
	}

	if status := makeRequest(
		t,
		handler,
		"127.0.0.1:54322",
	); status != http.StatusOK {
		t.Fatalf("second request: expected 200, got %d", status)
	}
}

func TestRateLimitThirdRequestReturns429(t *testing.T) {
	handler := newTestRateLimitMiddleware(
		t,
		2,
		time.Second,
	)

	makeRequest(t, handler, "127.0.0.1:54321")
	makeRequest(t, handler, "127.0.0.1:54322")

	status := makeRequest(
		t,
		handler,
		"127.0.0.1:54323",
	)

	if status != http.StatusTooManyRequests {
		t.Fatalf(
			"third request: expected 429, got %d",
			status,
		)
	}
}

func TestRateLimitSameIPDifferentPortsShareLimit(t *testing.T) {
	handler := newTestRateLimitMiddleware(
		t,
		2,
		time.Second,
	)

	if status := makeRequest(
		t,
		handler,
		"127.0.0.1:54321",
	); status != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", status)
	}

	if status := makeRequest(
		t,
		handler,
		"127.0.0.1:54322",
	); status != http.StatusOK {
		t.Fatalf("second request: expected 200, got %d", status)
	}

	if status := makeRequest(
		t,
		handler,
		"127.0.0.1:54323",
	); status != http.StatusTooManyRequests {
		t.Fatalf(
			"third request from same IP: expected 429, got %d",
			status,
		)
	}
}

func TestRateLimitDifferentIPsAreIndependent(t *testing.T) {
	handler := newTestRateLimitMiddleware(
		t,
		2,
		time.Second,
	)

	makeRequest(t, handler, "127.0.0.1:54321")
	makeRequest(t, handler, "127.0.0.1:54322")

	// First IP has reached its limit.
	if status := makeRequest(
		t,
		handler,
		"127.0.0.1:54323",
	); status != http.StatusTooManyRequests {
		t.Fatalf(
			"expected 429 for first IP, got %d",
			status,
		)
	}

	// Different IP has its own counter.
	if status := makeRequest(
		t,
		handler,
		"192.168.1.10:60000",
	); status != http.StatusOK {
		t.Fatalf(
			"expected 200 for different IP, got %d",
			status,
		)
	}
}
