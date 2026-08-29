package gateway

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/r7rainz/gateforge/internal/config"
)

func NewRouter(routes []config.Route, gateway map[string]http.Handler) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	for _, route := range routes {
		handler, ok := gateway[route.Service]
		if !ok {
			return nil, fmt.Errorf("no gateway found for service %q", route.Service)
		}
		pattern := route.Path

		if pattern != "/" && !strings.HasSuffix(pattern, "/") {
			pattern += "/"
		}

		mux.Handle(pattern, handler)
	}

	return mux, nil
}
