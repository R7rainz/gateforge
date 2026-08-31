package circuitbreaker

import (
	"testing"
	"time"
)

func newTestBreaker(t *testing.T) *CircuitBreaker {
	t.Helper()

	breaker, err := New(3, 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	return breaker
}

func TestClosedAllowRequests(t *testing.T) {
	breaker := newTestBreaker(t)

	if !breaker.Allow() {
		t.Fatal("closed breaker should allow requests")
	}

	if !breaker.Allow() {
		t.Fatal("closed breaker should allow requests")
	}
}

func TestThresholdOpensBreaker(t *testing.T) {
	breaker := newTestBreaker(t)

	breaker.Record(500, nil)
	breaker.Record(500, nil)

	if !breaker.Allow() {
		t.Fatal("breaker should still allow requests before threshold")
	}

	breaker.Record(500, nil)

	if breaker.Allow() {
		t.Fatal("breaker should reject requests after threshold")
	}
}

func TestOpenRejectsDuringCooldown(t *testing.T) {
	breaker := newTestBreaker(t)

	breaker.Record(500, nil)
	breaker.Record(500, nil)
	breaker.Record(500, nil)

	if breaker.Allow() {
		t.Fatal("open breaker should reject requests during cooldown")
	}
}
