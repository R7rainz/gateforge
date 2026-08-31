package circuitbreaker

import (
	"fmt"
	"sync"
	"time"
)

type State int

const (
	Closed State = iota
	Open
	HalfOpen
)

type CircuitBreaker struct {
	threshold int
	cooldown  time.Duration

	mu          sync.Mutex
	state       State
	failures    int
	openedAt    time.Time
	probeActive bool
}

func New(threshold int, cooldown time.Duration) (*CircuitBreaker, error) {
	if threshold <= 0 {
		return nil, fmt.Errorf("threshold must be greater than zero")
	}

	if cooldown <= 0 {
		return nil, fmt.Errorf("cooldown must be greater than zero")
	}

	return &CircuitBreaker{
		threshold: threshold,
		cooldown:  cooldown,
		state:     Closed,
	}, nil
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case Closed:
		return true

	case Open:
		if time.Since(cb.openedAt) < cb.cooldown {
			return false
		}

		cb.state = HalfOpen
		cb.probeActive = true

		return true

	case HalfOpen:
		if cb.probeActive {
			return false
		}

		cb.probeActive = true

		return true
	}
	return false
}

func (cb *CircuitBreaker) Record(statusCode int, err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.recordFailure()
		return
	}

	if statusCode >= 500 && statusCode <= 599 {
		cb.recordFailure()
		return
	}

	if statusCode >= 400 && statusCode <= 499 {
		if cb.state == HalfOpen {
			cb.recordSuccess()
		}
		return
	}

	cb.recordSuccess()
}

func (cb *CircuitBreaker) recordFailure() {
	switch cb.state {
	case Closed:
		cb.failures++

		if cb.failures >= cb.threshold {
			cb.state = Open
			cb.openedAt = time.Now()
		}
	case HalfOpen:
		cb.state = Open
		cb.openedAt = time.Now()
		cb.probeActive = false
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	switch cb.state {
	case Closed:
		cb.failures = 0

	case HalfOpen:
		cb.state = Closed
		cb.failures = 0
		cb.probeActive = false
	}
}
