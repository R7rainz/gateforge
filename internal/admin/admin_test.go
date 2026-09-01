package admin

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/r7rainz/gateforge/internal/loadbalancer"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	rec := httptest.NewRecorder()

	HealthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status 200, got %d",
			rec.Code,
		)
	}
}

func TestReadyHandlerAllServicesHealthy(t *testing.T) {
	usersURL, err := url.Parse("http://localhost:9000")
	if err != nil {
		t.Fatal(err)
	}

	ordersURL, err := url.Parse("http://localhost:9002")
	if err != nil {
		t.Fatal(err)
	}

	loadBalancers := map[string]*loadbalancer.RoundRobin{
		"users": loadbalancer.NewRoundRobin(
			[]*url.URL{usersURL},
		),
		"orders": loadbalancer.NewRoundRobin(
			[]*url.URL{ordersURL},
		),
	}

	handler := ReadyHandler(loadBalancers)

	req := httptest.NewRequest(
		http.MethodGet,
		"/ready",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status 200, got %d",
			rec.Code,
		)
	}
}

func TestReadyHandlerServiceUnhealthy(t *testing.T) {
	usersURL, err := url.Parse("http://localhost:9000")
	if err != nil {
		t.Fatal(err)
	}

	ordersURL, err := url.Parse("http://localhost:9002")
	if err != nil {
		t.Fatal(err)
	}

	usersLB := loadbalancer.NewRoundRobin(
		[]*url.URL{usersURL},
	)

	ordersLB := loadbalancer.NewRoundRobin(
		[]*url.URL{ordersURL},
	)

	ordersLB.SetHealth(ordersURL, false)

	loadBalancers := map[string]*loadbalancer.RoundRobin{
		"users":  usersLB,
		"orders": ordersLB,
	}

	handler := ReadyHandler(loadBalancers)

	req := httptest.NewRequest(
		http.MethodGet,
		"/ready",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status 503, got %d",
			rec.Code,
		)
	}
}
