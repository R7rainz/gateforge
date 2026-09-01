// Package loadbalancer distributes requests across backend instances.
package loadbalancer

import (
	"net/url"
	"sync"
)

type Backend struct {
	URL     *url.URL
	Healthy bool
}

type RoundRobin struct {
	backends []*Backend
	next     int
	mu       sync.Mutex
}

func NewRoundRobin(urls []*url.URL) *RoundRobin {
	backends := make([]*Backend, 0, len(urls))

	for _, url := range urls {
		backends = append(backends, &Backend{
			URL:     url,
			Healthy: true,
		})
	}

	return &RoundRobin{
		backends: backends,
	}
}

func (rr *RoundRobin) Next() *Backend {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	for i := 0; i < len(rr.backends); i++ {

		backend := rr.backends[rr.next]
		rr.next = (rr.next + 1) % len(rr.backends)

		if backend.Healthy {
			return backend
		}
	}
	return nil
}

func (rr *RoundRobin) SetHealth(url *url.URL, healthy bool) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	for _, backend := range rr.backends {
		if backend.URL.String() == url.String() {
			backend.Healthy = healthy
			return
		}
	}
}

func (rr *RoundRobin) Backends() []*Backend {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	return rr.backends
}
