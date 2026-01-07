package infra

import (
	"testing"
	"time"
)

func TestCircuitBreaker_AllowInClosed(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig("test"))

	if !cb.Allow() {
		t.Error("Expected Allow() to return true in CLOSED state")
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state CLOSED, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(cfg)

	// Record failures up to threshold
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.GetState() != StateClosed {
		t.Error("Should still be CLOSED after 2 failures")
	}

	cb.RecordFailure() // 3rd failure

	if cb.GetState() != StateOpen {
		t.Errorf("Expected OPEN after 3 failures, got %s", cb.GetState())
	}

	// Should reject requests when open
	if cb.Allow() {
		t.Error("Expected Allow() to return false in OPEN state")
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:          50 * time.Millisecond,
	}
	cb := NewCircuitBreaker(cfg)

	// Open the breaker
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.GetState() != StateOpen {
		t.Fatal("Expected OPEN state")
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open
	if !cb.Allow() {
		t.Error("Expected Allow() to return true after timeout (half-open)")
	}

	if cb.GetState() != StateHalfOpen {
		t.Errorf("Expected HALF_OPEN, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_ClosesOnSuccess(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          10 * time.Millisecond,
	}
	cb := NewCircuitBreaker(cfg)

	// Open the breaker
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait and transition to half-open
	time.Sleep(15 * time.Millisecond)
	cb.Allow()

	// Record successes
	cb.RecordSuccess()
	if cb.GetState() != StateHalfOpen {
		t.Error("Should still be HALF_OPEN after 1 success")
	}

	cb.RecordSuccess()
	if cb.GetState() != StateClosed {
		t.Errorf("Expected CLOSED after 2 successes, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig("test")
	cb := NewCircuitBreaker(cfg)

	// Open the breaker
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	if cb.GetState() != StateOpen {
		t.Fatal("Expected OPEN state")
	}

	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Errorf("Expected CLOSED after Reset, got %s", cb.GetState())
	}

	if !cb.Allow() {
		t.Error("Expected Allow() to return true after Reset")
	}
}
