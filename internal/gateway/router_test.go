package gateway

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/r7rainz/gateforge/internal/config"
)

func TestRouterMatchesCorrectService(t *testing.T) {
	routes := []config.Route{
		{
			Path:    "/api/users",
			Service: "users",
		},
		{
			Path:    "/api/orders",
			Service: "orders",
		},
	}

	gateway := map[string]http.Handler{
		"users": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "users")
		}),
		"orders": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "orders")
		}),
	}

	mux, err := NewRouter(routes, gateway)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path     string
		expected string
	}{{
		path:     "/api/users/123",
		expected: "users\n",
	},
		{
			path:     "/api/orders/456",
			expected: "orders\n",
		},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(
			http.MethodGet,
			tt.path,
			nil,
		)

		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf(
				"%s: expected status 200, got %d",
				tt.path,
				rec.Code,
			)
		}

		if rec.Body.String() != tt.expected {
			t.Fatalf(
				"%s: expected %q, got %q",
				tt.path,
				tt.expected,
				rec.Body.String(),
			)
		}
	}
}

func TestRouterUnknownRoute(t *testing.T) {
	routes := []config.Route{
		{
			Path:    "/api/users",
			Service: "users",
		},
	}

	gateway := map[string]http.Handler{
		"users": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "users")
		}),
	}

	mux, err := NewRouter(routes, gateway)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/products/123",
		nil,
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}
