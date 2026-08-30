package ratelimit

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestLimiter(t *testing.T, limit int, window time.Duration) *Limiter {
	t.Helper()

	limiter, err := New(limit, window)
	if err != nil {
		t.Fatal(err)
	}

	return limiter
}

func TestAllowUnderLimit(t *testing.T) {
	limiter := newTestLimiter(t, 2, 100*time.Millisecond)

	if !limiter.Allow("127.0.0.1") {
		t.Fatal("first request should be allowed")
	}

	if !limiter.Allow("127.0.0.1") {
		t.Fatal("second request should be allowed")
	}
}

func TestRejectOverLimit(t *testing.T) {
	limiter := newTestLimiter(t, 2, 100*time.Millisecond)

	if !limiter.Allow("127.0.0.1") {
		t.Fatal("first request should be allowed")
	}

	if !limiter.Allow("127.0.0.1") {
		t.Fatal("second request should be allowed")
	}

	if limiter.Allow("127.0.0.1") {
		t.Fatal("third request should be rejected")
	}
}

func TestWindowResets(t *testing.T) {
	limiter := newTestLimiter(t, 2, 100*time.Millisecond)

	if !limiter.Allow("127.0.0.1") {
		t.Fatal("first request should be allowed")
	}

	if !limiter.Allow("127.0.0.1") {
		t.Fatal("second request should be allowed")
	}

	if limiter.Allow("127.0.0.1") {
		t.Fatal("third request should be rejected")
	}

	time.Sleep(110 * time.Millisecond)

	if !limiter.Allow("127.0.0.1") {
		t.Fatal("request after window reset should be allowed")
	}
}

func TestClientsAreIndependent(t *testing.T) {
	limiter := newTestLimiter(t, 2, 100*time.Millisecond)

	if !limiter.Allow("127.0.0.1") {
		t.Fatal("first client request should be allowed")
	}

	if !limiter.Allow("127.0.0.1") {
		t.Fatal("second client request should be allowed")
	}

	if limiter.Allow("127.0.0.1") {
		t.Fatal("third request from first client should be rejected")
	}

	if !limiter.Allow("192.168.1.10") {
		t.Fatal("different client should have an independent counter")
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		remoteAddr string
		expected   string
	}{
		{
			remoteAddr: "127.0.0.1:54321",
			expected:   "127.0.0.1",
		},
		{
			remoteAddr: "127.0.0.1:54322",
			expected:   "127.0.0.1",
		},
		{
			remoteAddr: "[::1]:54321",
			expected:   "::1",
		},
	}

	for _, tt := range tests {
		got := ClientIP(tt.remoteAddr)

		if got != tt.expected {
			t.Fatalf(
				"ClientIP(%q): expected %q, got %q",
				tt.remoteAddr,
				tt.expected,
				got,
			)
		}
	}
}

func TestNewRejectsZeroLimit(t *testing.T) {
	_, err := New(0, 100*time.Millisecond)

	if err == nil {
		t.Fatal("expected error for zero limit")
	}
}

func TestNewRejectsNonPositiveWindow(t *testing.T) {
	_, err := New(2, 0)

	if err == nil {
		t.Fatal("expected error for zero window")
	}
}

func TestAllowConcurrent(t *testing.T) {
	limiter := newTestLimiter(t, 100, time.Minute)

	const workers = 1000

	var wg sync.WaitGroup
	var allowed int64
	var rejected int64

	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()

			if limiter.Allow("127.0.0.1") {
				atomic.AddInt64(&allowed, 1)
			} else {
				atomic.AddInt64(&rejected, 1)
			}
		}()
	}

	wg.Wait()

	if allowed != 100 {
		t.Fatalf(
			"expected 100 allowed requests, got %d",
			allowed,
		)
	}

	if rejected != 900 {
		t.Fatalf(
			"expected 900 rejected requests, got %d",
			rejected,
		)
	}
}
