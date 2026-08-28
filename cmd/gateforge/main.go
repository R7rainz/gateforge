package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/r7rainz/gateforge/internal/gateway"
	"github.com/r7rainz/gateforge/internal/healthcheck"
	"github.com/r7rainz/gateforge/internal/loadbalancer"
	"github.com/r7rainz/gateforge/internal/middleware"
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

	timeoutMux := http.TimeoutHandler(
		mux, 2*time.Second, "Gateway request timeout",
	)

	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelInfo,
			},
		),
	)
	loggedMux := middleware.RequestLog(logger)(timeoutMux)

	server := &http.Server{
		Addr:    ":8080",
		Handler: loggedMux,
	}

	signalCtx, stopSignal := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stopSignal()

	go func() {
		log.Println("GateForge Proxy starting at :8080...")

		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-signalCtx.Done()

	log.Println("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	log.Println("Shutting down HTTP server...")

	err := server.Shutdown(shutdownCtx)
	if err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Stopping health checker...")
	checker.Stop()

	log.Println("GateForge Stopped")
}
