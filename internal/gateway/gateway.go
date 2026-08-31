package gateway

import (
	"net/http"
	"net/http/httputil"

	"github.com/r7rainz/gateforge/internal/loadbalancer"
)

type Gateway struct {
	lb      *loadbalancer.RoundRobin
	retries int
}

func NewGateway(lb *loadbalancer.RoundRobin, retries int) *Gateway {
	return &Gateway{
		lb:      lb,
		retries: retries,
	}
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	attempt := 0

	for {
		backend := g.lb.Next()

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

			// Capture transport errors instead of immediately
			// writing a 502 response.
			ErrorHandler: func(
				w http.ResponseWriter,
				r *http.Request,
				err error,
			) {
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

func isRetryableMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}
