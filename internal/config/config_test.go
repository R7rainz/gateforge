package config

import (
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

func TestLoadValidConfig(t *testing.T) {
	path := writeTempConfig(t, `{
		"listen_address": ":8080",
		"backend_urls": [
			"http://localhost:9000",
			"http://localhost:9001"
		],
		"health_check_interval": "5s",
		"request_timeout": "2s",
		"shutdown_timeout": "5s"
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

	if len(cfg.BackendURLs) != 2 {
		t.Fatalf(
			"expected 2 backends, got %d",
			len(cfg.BackendURLs),
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
}

func TestLoadInvalidJSON(t *testing.T) {
	path := writeTempConfig(t, `{
		"listen_address": ":8080",
		"backend_urls":
	}`)

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadInvalidBackendURL(t *testing.T) {
	path := writeTempConfig(t, `{
		"listen_address": ":8080",
		"backend_urls": [
			"not-a-url"
		],
		"health_check_interval": "5s",
		"request_timeout": "2s",
		"shutdown_timeout": "5s"
	}`)

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected error for invalid backend URL")
	}
}

func TestLoadInvalidDuration(t *testing.T) {
	path := writeTempConfig(t, `{
		"listen_address": ":8080",
		"backend_urls": [
			"http://localhost:9000"
		],
		"health_check_interval": "hello",
		"request_timeout": "2s",
		"shutdown_timeout": "5s"
	}`)

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}

func TestLoadInvalidTimeout(t *testing.T) {
	path := writeTempConfig(t, `{
		"listen_address": ":8080",
		"backend_urls": [
			"http://localhost:9000"
		],
		"health_check_interval": "5s",
		"request_timeout": "0s",
		"shutdown_timeout": "5s"
	}`)

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected error for non-positive timeout")
	}
}
