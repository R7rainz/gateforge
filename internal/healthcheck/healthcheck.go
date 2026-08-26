package healthcheck

import (
	"net/http"
	"time"

	"github.com/r7rainz/gateforge/internal/loadbalancer"
)

type Checker struct {
	client   *http.Client
	interval time.Duration
}

func NewChecker(interval time.Duration) *Checker {
	return &Checker{
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
		interval: interval,
	}
}

func (c *Checker) Start(rr *loadbalancer.RoundRobin) {
	ticker := time.NewTicker(c.interval)

	go func() {
		for range ticker.C {
			for _, backend := range rr.Backends() {
				healthy := c.Check(backend)

				rr.SetHealth(backend.URL, healthy)
			}
		}
	}()
}

func (c *Checker) Check(backend *loadbalancer.Backend) bool {
	resp, err := c.client.Get(backend.URL.String() + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
