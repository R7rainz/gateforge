// Package main starts the GateForge HTTP gateway.
package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/r7rainz/gateforge/internal/admin"
	"github.com/r7rainz/gateforge/internal/config"
	"github.com/r7rainz/gateforge/internal/gateway"
	"github.com/r7rainz/gateforge/internal/healthcheck"
	"github.com/r7rainz/gateforge/internal/loadbalancer"
	"github.com/r7rainz/gateforge/internal/metrics"
	"github.com/r7rainz/gateforge/internal/middleware"
	"github.com/r7rainz/gateforge/internal/ratelimit"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	gateways := make(
		map[string]http.Handler,
		len(cfg.Services),
	)

	checkers := make(
		[]*healthcheck.Checker,
		0,
		len(cfg.Services),
	)

	loadBalancers := make(
		map[string]*loadbalancer.RoundRobin,
		len(cfg.Services),
	)

	// Create one load balancer and one health checker per service.
	for serviceName, rawURLs := range cfg.Services {
		urls := make([]*url.URL, 0, len(rawURLs))

		for _, rawURL := range rawURLs {
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
		loadBalancers[serviceName] = lb

		checker := healthcheck.NewChecker(
			cfg.HealthCheckInterval,
		)
		checker.Start(lb)
		checkers = append(checkers, checker)

		gateways[serviceName] = gateway.NewGateway(
			lb,
			cfg.MaxRetries,
			cfg.CircuitBreakerThreshold,
			cfg.CircuitBreakerCooldown,
		)
	}

	// Build routes from config.
	mux, err := gateway.NewRouter(
		cfg.Routes,
		gateways,
	)
	if err != nil {
		log.Fatalf("failed to create router: %v", err)
	}

	metricCollector := metrics.New()

	// API middleware.
	timeoutMux := http.TimeoutHandler(
		mux,
		cfg.RequestTimeout,
		"Gateway request timeout",
	)

	// Rate limiter: configured requests per window per client IP.
	rateLimiter, err := ratelimit.New(
		cfg.RateLimit,
		cfg.RateLimitWindow,
	)
	if err != nil {
		log.Fatalf("failed to create rate limiter: %v", err)
	}

	rateLimitedAPI := middleware.RateLimit(
		rateLimiter,
	)(timeoutMux)

	// Outer mux for admin endpoints and API.
	outerMux := http.NewServeMux()

	outerMux.HandleFunc(
		"/health",
		admin.HealthHandler,
	)
	outerMux.Handle(
		"/ready",
		admin.ReadyHandler(loadBalancers),
	)
	outerMux.Handle(
		"/metrics",
		metricCollector.Handler(),
	)
	outerMux.Handle(
		"/api/",
		rateLimitedAPI,
	)

	// Request logging middleware.
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelInfo,
			},
		),
	)

	metricMux := middleware.Metrics(
		metricCollector,
	)(outerMux)

	loggedMux := middleware.RequestLog(
		logger,
	)(metricMux)

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
		log.Printf(
			"GateForge Proxy starting at %s...",
			cfg.ListenAddress,
		)

		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-signalCtx.Done()

	log.Println("Shutdown signal received")

	// Stop all health-check goroutines.
	log.Println("Stopping health checkers...")

	for _, checker := range checkers {
		checker.Stop()
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		cfg.ShutdownTimeout,
	)
	defer cancel()

	log.Println("Shutting down HTTP server...")

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf(
			"HTTP server shutdown error: %v",
			err,
		)
	}

	log.Println("GateForge stopped")
}
