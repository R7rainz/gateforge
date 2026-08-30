package ratelimit

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type clientState struct {
	count       int
	windowStart time.Time
}

type Limiter struct {
	limit  int
	window time.Duration
	client map[string]*clientState
	mu     sync.Mutex
}

func New(limit int, window time.Duration) (*Limiter, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	if window <= 0 {
		return nil, fmt.Errorf("window must be greater than zero")
	}
	return &Limiter{
		limit:  limit,
		window: window,
		client: make(map[string]*clientState),
	}, nil
}

func (l *Limiter) Allow(clientIP string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	state, exists := l.client[clientIP]

	if !exists {
		l.client[clientIP] = &clientState{
			count:       1,
			windowStart: time.Now(),
		}
		return true
	}

	if time.Since(state.windowStart) >= l.window {
		state.count = 1
		state.windowStart = time.Now()
		return true
	}

	if state.count >= l.limit {
		return false
	}

	state.count++
	return true
}

func ClientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
