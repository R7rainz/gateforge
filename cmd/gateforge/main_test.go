package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/r7rainz/gateforge/internal/gateway"
	"github.com/r7rainz/gateforge/internal/loadbalancer"
)

func TestGatewayTimeout(t *testing.T) {
	slowBackend := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("slow response"))
		},
	))
	defer slowBackend.Close()

	backendURL, err := url.Parse(slowBackend.URL)
	if err != nil {
		t.Fatal(err)
	}

	lb := loadbalancer.NewRoundRobin([]*url.URL{
		backendURL,
	})

	gw := gateway.NewGateway(lb, 0)

	mux := http.NewServeMux()

	// This was missing.
	mux.Handle("/api/users/", gw)

	timeoutMux := http.TimeoutHandler(
		mux,
		2*time.Second,
		"Gateway request timeout",
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"http://localhost:8080/api/users/slow",
		nil,
	)

	rec := httptest.NewRecorder()

	start := time.Now()

	timeoutMux.ServeHTTP(rec, req)

	elapsed := time.Since(start)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status 503, got %d",
			rec.Code,
		)
	}

	if !strings.Contains(
		rec.Body.String(),
		"Gateway request timeout",
	) {
		t.Fatalf(
			"expected timeout message, got %q",
			rec.Body.String(),
		)
	}

	if elapsed > 3*time.Second {
		t.Fatalf(
			"request took too long: %v",
			elapsed,
		)
	}
}
