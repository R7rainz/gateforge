package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"
)

type Config struct {
	ListenAddress       string
	BackendURLs         []string
	HealthCheckInterval time.Duration
	RequestTimeout      time.Duration
	ShutdownTimeout     time.Duration
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config

	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsed config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func (c *Config) UnmarshalJSON(data []byte) error {
	var raw struct {
		ListenAddress       string   `json:"listen_address"`
		BackendURLs         []string `json:"backend_urls"`
		HealthCheckInterval string   `json:"health_check_interval"`
		RequestTimeout      string   `json:"request_timeout"`
		ShutdownTimeout     string   `json:"shutdown_timeout"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	healthCheckInterval, err := time.ParseDuration(
		raw.HealthCheckInterval,
	)
	if err != nil {
		return fmt.Errorf(
			"invalid health_check_interval: %w",
			err,
		)
	}

	requestTimeout, err := time.ParseDuration(raw.RequestTimeout)
	if err != nil {
		return fmt.Errorf("invalid request_timeout: %w", err)
	}

	shutdownTimeout, err := time.ParseDuration(
		raw.ShutdownTimeout,
	)
	if err != nil {
		return fmt.Errorf("invalid shutdown_timeout: %w", err)
	}

	c.ListenAddress = raw.ListenAddress
	c.BackendURLs = raw.BackendURLs
	c.HealthCheckInterval = healthCheckInterval
	c.RequestTimeout = requestTimeout
	c.ShutdownTimeout = shutdownTimeout

	return nil
}

func (c *Config) validate() error {
	if c.ListenAddress == "" {
		return fmt.Errorf("listen_address is required")
	}

	if len(c.BackendURLs) == 0 {
		return fmt.Errorf("backend_urls cannot be empty")
	}

	for i, rawURL := range c.BackendURLs {
		parsedURL, err := url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("backend_urls[%d] is invalid: %w", i, err)
		}
		if parsedURL.Scheme == "" || parsedURL.Host == "" {
			return fmt.Errorf("backend_urls[%d] must be an absolute URL: %q", i, rawURL)
		}
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf("backend_urls[%d] must use http or https: %q", i, rawURL)
		}
	}

	if c.HealthCheckInterval <= 0 {
		return fmt.Errorf("health_check_interval must be greater than zero")
	}

	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request_timeout must be greater than zero")
	}

	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown_timeout should be greater than zero")
	}

	return nil
}
