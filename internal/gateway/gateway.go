package gateway

import (
	"net/http"
	"net/http/httputil"

	"github.com/r7rainz/gateforge/internal/loadbalancer"
)

type Gateway struct {
	lb *loadbalancer.RoundRobin
}

func NewGateway(lb *loadbalancer.RoundRobin) *Gateway {
	return &Gateway{
		lb: lb,
	}
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend := g.lb.Next()

	if backend == nil {
		http.Error(
			w,
			"No healthy backends available",
			http.StatusBadGateway,
		)
		return
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(proxyReq *httputil.ProxyRequest) {
			proxyReq.SetURL(backend.URL)
			proxyReq.SetXForwarded()
		},
	}

	proxy.ServeHTTP(w, r)
}
