package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(
		t.TempDir(),
		"config.json",
	)

	err := os.WriteFile(
		path,
		[]byte(contents),
		0644,
	)
	if err != nil {
		t.Fatal(err)
	}

	return path
}

func validConfigWithBreaker(threshold int, cooldown string) string {
	return fmt.Sprintf(`{
		"listen_address": ":8080",
		"routes": [
			{
				"path": "/api/users",
				"service": "users"
			}
		],
		"services": {
			"users": [
				"http://localhost:9000"
			]
		},
		"health_check_interval": "5s",
		"request_timeout": "2s",
		"shutdown_timeout": "5s",
		"rate_limit": 2,
		"rate_limit_window": "100ms",
		"max_retries": 1,
		"circuit_breaker_threshold": %d,
		"circuit_breaker_cooldown": %q
	}`, threshold, cooldown)
}

func TestLoadValidConfig(t *testing.T) {
	path := writeTempConfig(t, `{
		"listen_address": ":8080",
		"routes": [
			{
				"path": "/api/users",
				"service": "users"
			},
			{
				"path": "/api/orders",
				"service": "orders"
			}
		],
		"services": {
			"users": [
				"http://localhost:9000",
				"http://localhost:9001"
			],
			"orders": [
				"http://localhost:9002",
				"http://localhost:9003"
			]
		},
		"health_check_interval": "5s",
		"request_timeout": "2s",
		"shutdown_timeout": "5s",
		"rate_limit": 2,
		"rate_limit_window": "100ms",
		"max_retries": 1,
		"circuit_breaker_threshold": 3,
		"circuit_breaker_cooldown": "5s"
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}

	if cfg.ListenAddress != ":8080" {
		t.Fatalf(
			"expected :8080, got %s",
			cfg.ListenAddress,
		)
	}

	if len(cfg.Routes) != 2 {
		t.Fatalf(
			"expected 2 routes, got %d",
			len(cfg.Routes),
		)
	}

	if len(cfg.Services) != 2 {
		t.Fatalf(
			"expected 2 services, got %d",
			len(cfg.Services),
		)
	}

	if len(cfg.Services["users"]) != 2 {
		t.Fatalf(
			"expected 2 users backends, got %d",
			len(cfg.Services["users"]),
		)
	}

	if len(cfg.Services["orders"]) != 2 {
		t.Fatalf(
			"expected 2 orders backends, got %d",
			len(cfg.Services["orders"]),
		)
	}

	if cfg.RateLimit != 2 {
		t.Fatalf("expected rate limit 2, got %d", cfg.RateLimit)
	}

	if cfg.RateLimitWindow != 100*time.Millisecond {
		t.Fatalf(
			"expected 100ms rate-limit window, got %v",
			cfg.RateLimitWindow,
		)
	}

	if cfg.HealthCheckInterval != 5*time.Second {
		t.Fatalf(
			"expected 5s health interval, got %v",
			cfg.HealthCheckInterval,
		)
	}

	if cfg.RequestTimeout != 2*time.Second {
		t.Fatalf(
			"expected 2s request timeout, got %v",
			cfg.RequestTimeout,
		)
	}

	if cfg.ShutdownTimeout != 5*time.Second {
		t.Fatalf(
			"expected 5s shutdown timeout, got %v",
			cfg.ShutdownTimeout,
		)
	}

	if cfg.CircuitBreakerThreshold != 3 {
		t.Fatalf(
			"expected circuit breaker threshold 3, got %d",
			cfg.CircuitBreakerThreshold,
		)
	}

	if cfg.CircuitBreakerCooldown != 5*time.Second {
		t.Fatalf(
			"expected 5s circuit breaker cooldown, got %v",
			cfg.CircuitBreakerCooldown,
		)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	path := writeTempConfig(t, `{
		"listen_address": ":8080",
		"routes":
	}`)

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadInvalidCircuitBreakerThreshold(t *testing.T) {
	path := writeTempConfig(t, validConfigWithBreaker(0, "5s"))

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected error for invalid circuit breaker threshold")
	}
}

func TestLoadInvalidCircuitBreakerCooldown(t *testing.T) {
	path := writeTempConfig(t, validConfigWithBreaker(3, "0s"))

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected error for invalid circuit breaker cooldown")
	}
}

func TestLoadInvalidBackendURL(t *testing.T) {
	path := writeTempConfig(t, `{
		"listen_address": ":8080",
		"routes": [
			{
				"path": "/api/users",
				"service": "users"
			}
		],
		"services": {
			"users": [
				"not-a-url"
			]
		},
		"health_check_interval": "5s",
		"request_timeout": "2s",
		"shutdown_timeout": "5s",
		"rate_limit": 2,
		"rate_limit_window": "100ms",
		"max_retries": 1,
		"circuit_breaker_threshold": 3,
		"circuit_breaker_cooldown": "5s"
	}`)

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected error for invalid backend URL")
	}
}

func TestLoadInvalidBackendScheme(t *testing.T) {
	path := writeTempConfig(t, `{
		"listen_address": ":8080",
		"routes": [
			{
				"path": "/api/users",
				"service": "users"
			}
		],
		"services": {
			"users": [
				"ftp://localhost:9000"
			]
		},
		"health_check_interval": "5s",
		"request_timeout": "2s",
		"shutdown_timeout": "5s",
		"rate_limit": 2,
		"rate_limit_window": "100ms",
		"max_retries": 1,
		"circuit_breaker_threshold": 3,
		"circuit_breaker_cooldown": "5s"
	}`)

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected error for invalid backend scheme")
	}
}

func TestLoadInvalidDuration(t *testing.T) {
	path := writeTempConfig(t, `{
		"listen_address": ":8080",
		"routes": [
			{
				"path": "/api/users",
				"service": "users"
			}
		],
		"services": {
			"users": [
				"http://localhost:9000"
			]
		},
		"health_check_interval": "hello",
		"request_timeout": "2s",
		"shutdown_timeout": "5s",
		"rate_limit": 2,
		"rate_limit_window": "100ms",
		"max_retries": 1,
		"circuit_breaker_threshold": 3,
		"circuit_breaker_cooldown": "5s"
	}`)

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}

func TestLoadInvalidTimeout(t *testing.T) {
	path := writeTempConfig(t, `{
		"listen_address": ":8080",
		"routes": [
			{
				"path": "/api/users",
				"service": "users"
			}
		],
		"services": {
			"users": [
				"http://localhost:9000"
			]
		},
		"health_check_interval": "5s",
		"request_timeout": "0s",
		"shutdown_timeout": "5s",
		"rate_limit": 2,
		"rate_limit_window": "100ms",
		"max_retries": 1,
		"circuit_breaker_threshold": 3,
		"circuit_breaker_cooldown": "5s"
	}`)

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected error for non-positive timeout")
	}
}

func TestLoadRouteWithUnknownService(t *testing.T) {
	path := writeTempConfig(t, `{
		"listen_address": ":8080",
		"routes": [
			{
				"path": "/api/users",
				"service": "missing"
			}
		],
		"services": {
			"users": [
				"http://localhost:9000"
			]
		},
		"health_check_interval": "5s",
		"request_timeout": "2s",
		"shutdown_timeout": "5s",
		"rate_limit": 2,
		"rate_limit_window": "100ms",
		"max_retries": 1,
		"circuit_breaker_threshold": 3,
		"circuit_breaker_cooldown": "5s"
	}`)

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected error for route referencing unknown service")
	}
}

func TestLoadNegativeMaxRetries(t *testing.T) {
	path := writeTempConfig(t, `{
		"listen_address": ":8080",
		"routes": [
			{
				"path": "/api/users",
				"service": "users"
			}
		],
		"services": {
			"users": [
				"http://localhost:9000"
			]
		},
		"health_check_interval": "5s",
		"request_timeout": "2s",
		"shutdown_timeout": "5s",
		"rate_limit": 2,
		"rate_limit_window": "100ms",
		"max_retries": -1,
		"circuit_breaker_threshold": 3,
		"circuit_breaker_cooldown": "5s"
	}`)

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected error for negative max_retries")
	}
}

func TestLoadZeroMaxRetries(t *testing.T) {
	path := writeTempConfig(t, `{
		"listen_address": ":8080",
		"routes": [
			{
				"path": "/api/users",
				"service": "users"
			}
		],
		"services": {
			"users": [
				"http://localhost:9000"
			]
		},
		"health_check_interval": "5s",
		"request_timeout": "2s",
		"shutdown_timeout": "5s",
		"rate_limit": 2,
		"rate_limit_window": "100ms",
		"max_retries": 0,
		"circuit_breaker_threshold": 3,
		"circuit_breaker_cooldown": "5s"
	}`)

	cfg, err := Load(path)

	if err != nil {
		t.Fatalf(
			"expected zero max_retries to be valid, got error: %v",
			err,
		)
	}

	if cfg.MaxRetries != 0 {
		t.Fatalf(
			"expected max_retries 0, got %d",
			cfg.MaxRetries,
		)
	}
}
