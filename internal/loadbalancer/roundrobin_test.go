package loadbalancer

import (
	"net/url"
	"sync"
	"testing"
)

func TestRoundRobinNext(t *testing.T) {
	//parsing 3 backend urls
	u1, _ := url.Parse("http://localhost:9000")
	u2, _ := url.Parse("http://localhost:9001")
	u3, _ := url.Parse("http://localhost:9002")

	//creating roundrobins with them
	backends := []*url.URL{u1, u2, u3}
	rr := NewRoundRobin(backends)

	expected := []string{
		"http://localhost:9000",
		"http://localhost:9001",
		"http://localhost:9002",
		"http://localhost:9000",
	}

	for i, exp := range expected {
		got := rr.Next().String()
		if got != exp {
			t.Fatalf("Call %d: expected %s, but got %s", i+1, exp, got)
		}
	}

}

func TestRoundRobin_Concurrent(t *testing.T) {
	u1, _ := url.Parse("http://b1")
	rr := NewRoundRobin([]*url.URL{u1})

	const workers = 1000
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			rr.Next() // Without a mutex, this will likely cause a race on rr.next
		}()
	}
	wg.Wait()
}
