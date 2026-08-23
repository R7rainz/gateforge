package loadbalancer

import (
	"net/url"
	"sync"
)

type RoundRobin struct {
	backends []*url.URL
	next     int
	mu       sync.Mutex
}

func NewRoundRobin(urls []*url.URL) *RoundRobin {
	return &RoundRobin{
		backends: urls,
	}
}

func (rr *RoundRobin) Next() *url.URL {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	backend := rr.backends[rr.next]

	rr.next = (rr.next + 1) % len(rr.backends)
	return backend
}
