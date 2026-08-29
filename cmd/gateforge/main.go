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

	"github.com/r7rainz/gateforge/internal/config"
	"github.com/r7rainz/gateforge/internal/gateway"
	"github.com/r7rainz/gateforge/internal/healthcheck"
	"github.com/r7rainz/gateforge/internal/loadbalancer"
	"github.com/r7rainz/gateforge/internal/middleware"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	urls := make([]*url.URL, 0, len(cfg.BackendURLs))
	for _, rawURL := range cfg.BackendURLs {
		backendURL, err := url.Parse(rawURL)
		if err != nil {
			log.Fatalf(
				"failed to parse backend URL %q: %v",
				rawURL,
				err,
			)
		}
		urls = append(urls, backendURL)
	}

	lb := loadbalancer.NewRoundRobin(urls)

	checker := healthcheck.NewChecker(cfg.HealthCheckInterval)
	checker.Start(lb)

	gw := gateway.NewGateway(lb)

	mux := http.NewServeMux()

	mux.Handle("/api/users/", gw)

	mux.HandleFunc("/api/orders/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Orders endpoint")
	})

	timeoutMux := http.TimeoutHandler(
		mux, cfg.RequestTimeout, "Gateway request timeout",
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
		Addr:    cfg.ListenAddress,
		Handler: loggedMux,
	}

	signalCtx, stopSignal := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stopSignal()

	go func() {
		log.Printf("GateForge Proxy starting at %s...", cfg.ListenAddress)

		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-signalCtx.Done()

	log.Println("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		cfg.ShutdownTimeout,
	)
	defer cancel()

	log.Println("Shutting down HTTP server...")

	err = server.Shutdown(shutdownCtx)
	if err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Stopping health checker...")
	checker.Stop()

	log.Println("GateForge Stopped")
}
