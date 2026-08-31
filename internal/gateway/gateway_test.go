package gateway

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/r7rainz/gateforge/internal/loadbalancer"
)

func TestGatewayRetriesGETOnBackendFailure(t *testing.T) {
	//create a backend server and close it immediately
	//its url will now represent a failed backend
	failedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("failed backend should not receive any request")
	}))

	failedURL, err := url.Parse(failedServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	failedServer.Close()

	var requestCount atomic.Int32

	healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))

	defer healthyServer.Close()

	healthyURL, err := url.Parse(healthyServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	//failed backend comes first
	lb := loadbalancer.NewRoundRobin(
		[]*url.URL{failedURL, healthyURL},
	)

	gw := NewGateway(lb, 1)

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/api/users/123", nil)

	rec := httptest.NewRecorder()

	gw.ServeHTTP(rec, req)

	//the gateway should retry the failed backend and successfully reach the healthy backend
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 after retry, got %d", rec.Code)
	}

	//healthy backend should receive exactly one request
	if got := requestCount.Load(); got != 1 {
		t.Fatalf("expected healthy backend to receive 1 request, got %d", got)
	}
}

func TestGatewayDoesNotRetryPOST(t *testing.T) {
	//failed first backend
	failedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("failed backend should not receive a successfull request")
	}))

	failedURL, err := url.Parse(failedServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	failedServer.Close()

	//healthy second backend
	var requestCount atomic.Int32

	healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))

	defer healthyServer.Close()

	healthyURL, err := url.Parse(healthyServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	lb := loadbalancer.NewRoundRobin([]*url.URL{
		failedURL, healthyURL,
	})

	gw := NewGateway(lb, 1)

	req := httptest.NewRequest(http.MethodPost, "http://gateway.local/api/users", nil)

	rec := httptest.NewRecorder()

	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expeceted status 502, got %d", rec.Code)
	}

	if got := requestCount.Load(); got != 0 {
		t.Fatalf("expected healthy backend to receive 0 requests, got %d", got)
	}
}

func TestGatewayDoesNotRetryOnBackend500(t *testing.T) {
	var secondRequestCount atomic.Int32

	firstServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	defer firstServer.Close()

	secondServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secondRequestCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))

	defer secondServer.Close()

	firstURL, err := url.Parse(firstServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	secondURL, err := url.Parse(secondServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	lb := loadbalancer.NewRoundRobin([]*url.URL{
		firstURL, secondURL,
	})

	gw := NewGateway(lb, 1)

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/api/users", nil)
	rec := httptest.NewRecorder()

	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	if got := secondRequestCount.Load(); got != 0 {
		t.Fatalf("expected second backend to receive 0 requests, got %d", got)
	}
}
