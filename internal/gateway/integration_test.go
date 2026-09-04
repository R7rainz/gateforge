package gateway

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/r7rainz/gateforge/internal/config"
	"github.com/r7rainz/gateforge/internal/loadbalancer"
)

func TestUsersRouteIntegration(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("users backend"))
	}))

	defer backend.Close()

	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	lb := loadbalancer.NewRoundRobin([]*url.URL{backendURL})
	gw := NewGateway(lb, 0, 3, 2*time.Second)

	routes := []config.Route{{
		Path:    "/api/users",
		Service: "users",
	}}

	gateways := map[string]http.Handler{"users": gw}

	router, err := NewRouter(routes, gateways)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if rec.Body.String() != "users backend" {
		t.Fatalf("expected body %q, got %q", "users backend", rec.Body.String())
	}
}
