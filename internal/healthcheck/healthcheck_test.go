package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/r7rainz/gateforge/internal/loadbalancer"
)

func TestCheckHealthyBackend(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/health" {
				t.Errorf("expected /health, got %s", r.URL.Path)
			}

			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	backend := &loadbalancer.Backend{
		URL: serverURL,
	}

	checker := NewChecker(5 * time.Second)

	if !checker.Check(backend) {
		t.Fatal("expected backend to be healthy")
	}

}

func TestCheckBackend_500Response(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)

	defer server.Close()

	backendURL, err := url.Parse(server.URL)

	if err != nil {
		t.Fatal(err)
	}

	backend := &loadbalancer.Backend{
		URL: backendURL,
	}

	checker := NewChecker(5 * time.Second)
	healthy := checker.Check(backend)

	if healthy {
		t.Fatalf("expected backend to be unhealthy")
	}
}
