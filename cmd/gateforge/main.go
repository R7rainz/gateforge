package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/r7rainz/gateforge/internal/gateway"
	"github.com/r7rainz/gateforge/internal/healthcheck"
	"github.com/r7rainz/gateforge/internal/loadbalancer"
)

func main() {
	ports := []string{
		"9000", "9001", "9002", "9003",
	}

	urls := make([]*url.URL, 0, len(ports))

	for _, port := range ports {
		backendURL, err := url.Parse(
			"http://localhost:" + port,
		)
		if err != nil {
			panic(err)
		}
		urls = append(urls, backendURL)
	}

	lb := loadbalancer.NewRoundRobin(urls)
	checker := healthcheck.NewChecker(5 * time.Second)
	checker.Start(lb)

	gw := gateway.NewGateway(lb)

	mux := http.NewServeMux()

	mux.Handle("/api/users/", gw)

	mux.HandleFunc("/api/orders/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Orders endpoint")
	})

	log.Println("GateForge Proxy starting on :8080...")
	timeoutMux := http.TimeoutHandler(mux, 2*time.Second, "Gateway request timeout")
	err := http.ListenAndServe(":8080", timeoutMux)
	if err != nil {
		log.Fatal(err)
	}
}
