// Package admin provides GateForge health and readiness handlers.
package admin

import (
	"fmt"
	"net/http"

	"github.com/r7rainz/gateforge/internal/loadbalancer"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func ReadyHandler(loadBalancers map[string]*loadbalancer.RoundRobin) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for serviceName, lb := range loadBalancers {
			if !lb.HasHealthyBackend() {
				http.Error(
					w,
					fmt.Sprintf("service %q is not ready", serviceName),
					http.StatusServiceUnavailable,
				)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	}
}
