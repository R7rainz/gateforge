package main

import (
	"log"
	"net/http"
	"net/url"

	"github.com/r7rainz/gateforge/internal/gateway"
	"github.com/r7rainz/gateforge/internal/loadbalancer"
)

func main() {
	u1, _ := url.Parse("http://localhost:9000")
	u2, _ := url.Parse("http://localhost:9001")
	u3, _ := url.Parse("http://localhost:9002")

	backends := []*url.URL{u1, u2, u3}

	lb := loadbalancer.NewRoundRobin(backends)

	gw := gateway.NewGateway(lb)

	mux := http.NewServeMux()

	mux.Handle("/api/users/", gw)

	log.Println("GateForge Proxy starting on :8080...")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
