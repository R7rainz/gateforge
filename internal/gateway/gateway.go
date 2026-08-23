package gateway

import (
	"net/http"
	"net/http/httputil"

	"github.com/r7rainz/gateforge/internal/loadbalancer"
)

type Gateway struct {
	lb    *loadbalancer.RoundRobin
	proxy *httputil.ReverseProxy
}

func NewGateway(lb *loadbalancer.RoundRobin) *Gateway {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(proxyReq *httputil.ProxyRequest) {
			target := lb.Next()
			//handles scheme, host and path joining automatically
			proxyReq.SetURL(target)
			//set standard proxy headers like X-Forwarded-For
			proxyReq.SetXForwarded()

		},
	}
	return &Gateway{
		lb:    lb,
		proxy: proxy,
	}
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.proxy.ServeHTTP(w, r)
}
