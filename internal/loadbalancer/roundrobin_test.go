package loadbalancer

import (
	"net/url"
	"sync"
	"testing"
)

func TestRoundRobinNext(t *testing.T) {
	// Parsing 3 backend URLs
	u1, _ := url.Parse("http://localhost:9000")
	u2, _ := url.Parse("http://localhost:9001")
	u3, _ := url.Parse("http://localhost:9002")

	// Creating round robin with them
	backends := []*url.URL{u1, u2, u3}
	rr := NewRoundRobin(backends)

	expected := []string{
		"http://localhost:9000",
		"http://localhost:9001",
		"http://localhost:9002",
		"http://localhost:9000",
	}

	for i, exp := range expected {
		backend := rr.Next()

		if backend == nil {
			t.Fatalf("Call %d: expected backend, got nil", i+1)
		}

		got := backend.URL.String()

		if got != exp {
			t.Fatalf(
				"Call %d: expected %s, but got %s",
				i+1,
				exp,
				got,
			)
		}
	}
}

func TestRoundRobinConcurrent(t *testing.T) {
	u1, _ := url.Parse("http://b1")

	rr := NewRoundRobin([]*url.URL{u1})

	const workers = 1000

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()

			backend := rr.Next()

			if backend == nil {
				t.Errorf("expected backend, got nil")
			}
		}()
	}

	wg.Wait()
}
