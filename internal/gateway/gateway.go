// Package gateway routes requests and proxies them to backend services.
package gateway

import (
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/r7rainz/gateforge/internal/circuitbreaker"
	"github.com/r7rainz/gateforge/internal/loadbalancer"
)

type Gateway struct {
	lb       *loadbalancer.RoundRobin
	retries  int
	breakers map[*loadbalancer.Backend]*circuitbreaker.CircuitBreaker
}

func NewGateway(
	lb *loadbalancer.RoundRobin,
	retries int,
	breakerThreshold int,
	breakerCooldown time.Duration,
) *Gateway {
	breakers := make(map[*loadbalancer.Backend]*circuitbreaker.CircuitBreaker)
	for _, backend := range lb.Backends() {
		breaker, err := circuitbreaker.New(
			breakerThreshold,
			breakerCooldown,
		)
		if err != nil {
			panic(err)
		}

		breakers[backend] = breaker
	}

	return &Gateway{
		lb:       lb,
		retries:  retries,
		breakers: breakers,
	}
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	attempt := 0

	for {
		backend := g.nextAvailableBackend()

		if backend == nil {
			http.Error(
				w,
				"No healthy backends available",
				http.StatusBadGateway,
			)
			return
		}

		var proxyErr error

		proxy := &httputil.ReverseProxy{
			Rewrite: func(proxyReq *httputil.ProxyRequest) {
				proxyReq.SetURL(backend.URL)
				proxyReq.SetXForwarded()
			},

			ModifyResponse: func(resp *http.Response) error {
				g.breakers[backend].Record(resp.StatusCode, nil)
				return nil
			},

			// Capture transport errors instead of immediately
			// writing a 502 response.
			ErrorHandler: func(
				w http.ResponseWriter,
				r *http.Request,
				err error,
			) {
				g.breakers[backend].Record(0, err)
				proxyErr = err
			},
		}

		proxy.ServeHTTP(w, r)

		// No transport error.
		// This includes backend responses like 200, 404, 500, etc.
		if proxyErr == nil {
			return
		}

		// Only GET, HEAD, and OPTIONS are retryable.
		if !isRetryableMethod(r.Method) {
			http.Error(
				w,
				"Bad Gateway",
				http.StatusBadGateway,
			)
			return
		}

		// We have used all retry attempts.
		if attempt >= g.retries {
			http.Error(
				w,
				"Bad Gateway",
				http.StatusBadGateway,
			)
			return
		}

		attempt++
	}
}

func (g *Gateway) nextAvailableBackend() *loadbalancer.Backend {
	backends := g.lb.Backends()

	for range backends {
		backend := g.lb.Next()

		if backend == nil {
			return nil
		}

		if g.breakers[backend].Allow() {
			return backend
		}
	}
	return nil
}

func isRetryableMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}
