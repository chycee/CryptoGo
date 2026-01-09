package infra

import (
	"log/slog"
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	StateClosed   State = iota // Normal operation
	StateOpen                  // Failing, reject requests
	StateHalfOpen              // Testing recovery
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker implements the circuit breaker pattern for fault isolation.
// Thread-safe for concurrent use.
type CircuitBreaker struct {
	name string
	mu   sync.RWMutex

	state        State
	failureCount int
	successCount int
	lastFailure  time.Time

	// Configuration
	failureThreshold int           // Failures before opening
	successThreshold int           // Successes before closing (in half-open)
	timeout          time.Duration // Time before trying half-open
}

// CircuitBreakerConfig holds configuration for creating a circuit breaker.
type CircuitBreakerConfig struct {
	Name             string
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:             name,
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	}
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		name:             cfg.Name,
		state:            StateClosed,
		failureThreshold: cfg.FailureThreshold,
		successThreshold: cfg.SuccessThreshold,
		timeout:          cfg.Timeout,
	}
}

// Allow checks if a request should be allowed.
// Returns true if the request can proceed, false if it should be rejected.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = StateHalfOpen
			cb.successCount = 0
			slog.Info("Circuit breaker transitioning to HALF_OPEN",
				slog.String("name", cb.name))
			return true
		}
		return false

	case StateHalfOpen:
		// Allow limited requests in half-open state
		return true

	default:
		return false
	}
}

// RecordSuccess records a successful operation.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.failureCount = 0

	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.state = StateClosed
			cb.failureCount = 0
			cb.successCount = 0
			slog.Info("Circuit breaker CLOSED (recovered)",
				slog.String("name", cb.name))
		}
	}
}

// RecordFailure records a failed operation.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailure = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
			slog.Warn("Circuit breaker OPEN (failures exceeded threshold)",
				slog.String("name", cb.name),
				slog.Int("failures", cb.failureCount))
		}

	case StateHalfOpen:
		// Any failure in half-open returns to open
		cb.state = StateOpen
		cb.successCount = 0
		slog.Warn("Circuit breaker OPEN (half-open test failed)",
			slog.String("name", cb.name))
	}
}

// GetState returns the current state (for monitoring).
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset forces the circuit breaker to closed state (for testing/admin).
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	slog.Info("Circuit breaker RESET", slog.String("name", cb.name))
}
