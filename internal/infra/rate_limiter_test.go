package infra

import (
	"testing"
	"time"
)

func TestRateLimiter_TryAcquire(t *testing.T) {
	// Create limiter with 2 tokens, 10/second refill
	rl := NewRateLimiter(2, 10)

	// Should acquire first two tokens immediately
	if !rl.TryAcquire() {
		t.Error("expected first TryAcquire to succeed")
	}
	if !rl.TryAcquire() {
		t.Error("expected second TryAcquire to succeed")
	}

	// Third should fail (no tokens left)
	if rl.TryAcquire() {
		t.Error("expected third TryAcquire to fail")
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	// Create limiter with 1 token, 10/second refill
	rl := NewRateLimiter(1, 10)

	// Exhaust the token
	if !rl.TryAcquire() {
		t.Error("expected first TryAcquire to succeed")
	}

	// Should fail immediately
	if rl.TryAcquire() {
		t.Error("expected immediate TryAcquire to fail")
	}

	// Wait for refill (100ms = 1 token at 10/s)
	time.Sleep(120 * time.Millisecond)

	// Should succeed after refill
	if !rl.TryAcquire() {
		t.Error("expected TryAcquire to succeed after refill")
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	// Create limiter with 1 token, 100/second refill (fast for testing)
	rl := NewRateLimiter(1, 100)

	// Exhaust the token
	rl.Wait()

	// Second Wait should block ~10ms (1/100 second)
	start := time.Now()
	rl.Wait()
	elapsed := time.Since(start)

	// Should have waited at least 5ms (allowing some tolerance)
	if elapsed < 5*time.Millisecond {
		t.Errorf("expected Wait to block, but elapsed=%v", elapsed)
	}
}

func TestBitgetLimiters_Initialized(t *testing.T) {
	// Verify singleton initialization works
	order := GetBitgetOrderLimiter()
	account := GetBitgetAccountLimiter()
	market := GetBitgetMarketLimiter()

	if order == nil {
		t.Error("order limiter is nil")
	}
	if account == nil {
		t.Error("account limiter is nil")
	}
	if market == nil {
		t.Error("market limiter is nil")
	}

	// Verify they are different instances
	if order == account {
		t.Error("order and account limiters should be different")
	}
}
