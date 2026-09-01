// Package config loads and validates GateForge configuration.
package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

type Route struct {
	Path    string `json:"path"`
	Service string `json:"service"`
}

type Services map[string][]string

type Config struct {
	ListenAddress           string   `json:"listen_address"`
	Routes                  []Route  `json:"routes"`
	Services                Services `json:"services"`
	HealthCheckInterval     time.Duration
	RequestTimeout          time.Duration
	ShutdownTimeout         time.Duration
	RateLimit               int
	RateLimitWindow         time.Duration
	MaxRetries              int
	CircuitBreakerThreshold int
	CircuitBreakerCooldown  time.Duration
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config

	if err = json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if err = cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func (c *Config) UnmarshalJSON(data []byte) error {
	var raw struct {
		ListenAddress           string   `json:"listen_address"`
		Routes                  []Route  `json:"routes"`
		Services                Services `json:"services"`
		HealthCheckInterval     string   `json:"health_check_interval"`
		RequestTimeout          string   `json:"request_timeout"`
		ShutdownTimeout         string   `json:"shutdown_timeout"`
		RateLimit               int      `json:"rate_limit"`
		RateLimitWindow         string   `json:"rate_limit_window"`
		MaxRetries              int      `json:"max_retries"`
		CircuitBreakerThreshold int      `json:"circuit_breaker_threshold"`
		CircuitBreakerCooldown  string   `json:"circuit_breaker_cooldown"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	healthCheckInterval, err := time.ParseDuration(raw.HealthCheckInterval)
	if err != nil {
		return fmt.Errorf("invalid health_check_interval: %w", err)
	}

	requestTimeout, err := time.ParseDuration(raw.RequestTimeout)
	if err != nil {
		return fmt.Errorf("invalid request_timeout: %w", err)
	}

	shutdownTimeout, err := time.ParseDuration(raw.ShutdownTimeout)
	if err != nil {
		return fmt.Errorf("invalid shutdown_timeout: %w", err)
	}

	rateLimitWindow, err := time.ParseDuration(raw.RateLimitWindow)
	if err != nil {
		return fmt.Errorf("invalid rate_limit_window: %w", err)
	}

	circuitBreakerCooldown, err := time.ParseDuration(
		raw.CircuitBreakerCooldown,
	)
	if err != nil {
		return fmt.Errorf("invalid circuit_breaker_cooldown: %w", err)
	}

	c.ListenAddress = raw.ListenAddress
	c.Routes = raw.Routes
	c.Services = raw.Services
	c.HealthCheckInterval = healthCheckInterval
	c.RequestTimeout = requestTimeout
	c.ShutdownTimeout = shutdownTimeout
	c.RateLimit = raw.RateLimit
	c.RateLimitWindow = rateLimitWindow
	c.MaxRetries = raw.MaxRetries
	c.CircuitBreakerThreshold = raw.CircuitBreakerThreshold
	c.CircuitBreakerCooldown = circuitBreakerCooldown

	return nil
}

func (c *Config) validate() error {
	if c.ListenAddress == "" {
		return fmt.Errorf("listen_address is required")
	}

	if len(c.Services) == 0 {
		return fmt.Errorf("services cannot be empty")
	}

	for serviceName, backendURLs := range c.Services {
		if strings.TrimSpace(serviceName) == "" {
			return fmt.Errorf("service name cannot be empty")
		}

		if len(backendURLs) == 0 {
			return fmt.Errorf("service %q must have at least one backend", serviceName)
		}

		for i, rawURL := range backendURLs {
			parsedURL, err := url.Parse(rawURL)
			if err != nil {
				return fmt.Errorf("services[%q][%d] is invalid: %w", serviceName, i, err)
			}

			if parsedURL.Scheme == "" || parsedURL.Host == "" {
				return fmt.Errorf("service[%q][%d] must be an absolute URL: %q", serviceName, i, rawURL)
			}

			if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
				return fmt.Errorf("services[%q][%d] must use http or https: %q", serviceName, i, rawURL)
			}

		}
	}

	if len(c.Routes) == 0 {
		return fmt.Errorf("routes cannot be empty")
	}

	seenPaths := make(map[string]struct{}, len(c.Routes))

	for _, route := range c.Routes {
		if route.Path == "" {
			return fmt.Errorf("route path cannot be empty")
		}

		if !strings.HasPrefix(route.Path, "/") {
			return fmt.Errorf("route path must start with '/' :%q", route.Path)
		}

		if route.Service == "" {
			return fmt.Errorf("route %q must specify a service", route.Path)
		}

		if _, exists := c.Services[route.Service]; !exists {
			return fmt.Errorf("route %q references unknown service %q", route.Path, route.Service)
		}

		if _, exists := seenPaths[route.Path]; exists {
			return fmt.Errorf("duplicate route path: %q", route.Path)
		}

		seenPaths[route.Path] = struct{}{}
	}
	if c.HealthCheckInterval <= 0 {
		return fmt.Errorf("health_check_interval must be greater than zero")
	}

	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request_timeout must be greater than zero")
	}

	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown_timeout must be greater than zero")
	}

	if c.RateLimit <= 0 {
		return fmt.Errorf("rate_limit must be greater than zero")
	}

	if c.RateLimitWindow <= 0 {
		return fmt.Errorf("rate_limit_window must be greater than zero")
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non negative")
	}

	if c.CircuitBreakerThreshold <= 0 {
		return fmt.Errorf(
			"circuit_breaker_threshold must be greater than zero",
		)
	}

	if c.CircuitBreakerCooldown <= 0 {
		return fmt.Errorf(
			"circuit_breaker_cooldown must be greater than zero",
		)
	}

	return nil
}
