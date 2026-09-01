// Package healthcheck monitors backend health and updates their availability.
package healthcheck

import (
	"net/http"
	"time"

	"github.com/r7rainz/gateforge/internal/loadbalancer"
)

type Checker struct {
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
}

func NewChecker(interval time.Duration) *Checker {
	return &Checker{
		interval: interval,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

func (c *Checker) Start(rr *loadbalancer.RoundRobin) {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	ticker := time.NewTicker(c.interval)

	go func() {
		defer close(c.done)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				for _, backend := range rr.Backends() {
					healthy := check(client, backend)
					rr.SetHealth(backend.URL, healthy)
				}
			case <-c.stop:
				return
			}
		}
	}()
}

func (c *Checker) Stop() {
	close(c.stop)
	<-c.done
}

func check(client *http.Client, backend *loadbalancer.Backend) bool {
	resp, err := client.Get(
		backend.URL.String() + "/health",
	)

	if err != nil {
		return false
	}

	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
