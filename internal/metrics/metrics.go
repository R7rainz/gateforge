// Package metrics collects and exposes GateForge request metrics.
package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/r7rainz/gateforge/internal/circuitbreaker"
	"github.com/r7rainz/gateforge/internal/gateway"
	"github.com/r7rainz/gateforge/internal/loadbalancer"
)

type Metrics struct {
	totalRequests       atomic.Uint64
	errors5xx           atomic.Uint64
	rateLimitedRequests atomic.Uint64
	totalLatencyNanos   atomic.Uint64
	loadBalancers       map[string]*loadbalancer.RoundRobin
	gatewayInstances    map[string]*gateway.Gateway
}

type Snapshot struct {
	TotalRequests       uint64
	Errors5xx           uint64
	RateLimitedRequests uint64
	TotalLatencyNanos   uint64
}

func New(loadbalancer map[string]*loadbalancer.RoundRobin, gatewayInstances map[string]*gateway.Gateway) *Metrics {
	return &Metrics{
		loadBalancers:    loadbalancer,
		gatewayInstances: gatewayInstances,
	}
}

func (m *Metrics) Record(status int, latency time.Duration) {
	m.totalRequests.Add(1)
	m.totalLatencyNanos.Add(uint64(latency.Nanoseconds()))

	if status >= 500 && status <= 599 {
		m.errors5xx.Add(1)
	}

	if status == http.StatusTooManyRequests {
		m.rateLimitedRequests.Add(1)
	}
}

func (m *Metrics) Snapshot() Snapshot {
	return Snapshot{
		TotalRequests:       uint64(m.totalRequests.Load()),
		Errors5xx:           uint64(m.errors5xx.Load()),
		RateLimitedRequests: uint64(m.rateLimitedRequests.Load()),
		TotalLatencyNanos:   uint64(m.totalLatencyNanos.Load()),
	}
}

func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snapshot := m.Snapshot()

		w.Header().Set(
			"Content-Type",
			"text/plain; version=0.0.4; charset=utf-8",
		)

		fmt.Fprintf(
			w,
			"# HELP gateforge_requests_total Total number of requests\n"+
				"# TYPE gateforge_requests_total counter\n"+
				"gateforge_requests_total %d\n\n"+
				"# HELP gateforge_errors_5xx_total Total number of 5xx responses\n"+
				"# TYPE gateforge_errors_5xx_total counter\n"+
				"gateforge_errors_5xx_total %d\n\n"+
				"# HELP gateforge_rate_limited_requests_total Total number of rate-limited requests\n"+
				"# TYPE gateforge_rate_limited_requests_total counter\n"+
				"gateforge_rate_limited_requests_total %d\n\n"+
				"# HELP gateforge_request_latency_nanoseconds_total Total request latency in nanoseconds\n"+
				"# TYPE gateforge_request_latency_nanoseconds_total counter\n"+
				"gateforge_request_latency_nanoseconds_total %d\n",
			snapshot.TotalRequests,
			snapshot.Errors5xx,
			snapshot.RateLimitedRequests,
			snapshot.TotalLatencyNanos,
		)

		fmt.Fprintf(
			w,
			"\n# HELP gateforge_backend_health Health status of GateForge backends\n"+
				"# TYPE gateforge_backend_health gauge\n",
		)

		for service, loadBalancer := range m.loadBalancers {
			states := loadBalancer.Snapshot()

			for _, state := range states {
				health := 0

				if state.Healthy {
					health = 1
				}

				fmt.Fprintf(w, "gateforge_backend_health{service=%q, backend=%q} %d\n", service, state.URL, health)
			}
		}

		fmt.Fprintf(
			w,
			"\n# HELP gateforge_circuit_breaker_state Current circuit breaker state\n"+
				"# TYPE gateforge_circuit_breaker_state gauge\n",
		)

		for service, gw := range m.gatewayInstances {
			states := gw.CircuitBreakerSnapshot()

			for _, state := range states {
				value := 0

				switch state.State {
				case circuitbreaker.Closed:
					value = 0
				case circuitbreaker.Open:
					value = 1
				case circuitbreaker.HalfOpen:
					value = 2
				}

				fmt.Fprintf(w, "gateforge_circuit_breaker_state{service=%q,backend=%q} %d\n", service, state.BackendURL, value)
			}
		}
	})
}
