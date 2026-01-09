package infra

import (
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter.
// Thread-safe and suitable for concurrent API calls.
type RateLimiter struct {
	mu          sync.Mutex
	tokens      float64
	maxTokens   float64
	refillRate  float64 // tokens per second
	lastRefill  time.Time
	minInterval time.Duration // minimum time between requests
	lastRequest time.Time
}

// NewRateLimiter creates a new rate limiter.
// maxRequests: maximum burst size
// perSecond: refill rate (requests per second)
func NewRateLimiter(maxRequests int, perSecond float64) *RateLimiter {
	now := time.Now()
	return &RateLimiter{
		tokens:      float64(maxRequests),
		maxTokens:   float64(maxRequests),
		refillRate:  perSecond,
		lastRefill:  now,
		minInterval: time.Duration(float64(time.Second) / perSecond),
		lastRequest: now.Add(-time.Hour), // Allow immediate first request
	}
}

// Wait blocks until a token is available.
// Returns immediately if a token is available.
func (r *RateLimiter) Wait() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()

	for r.tokens < 1 {
		// Calculate wait time for next token
		waitTime := time.Duration(float64(time.Second) / r.refillRate)
		r.mu.Unlock()
		time.Sleep(waitTime)
		r.mu.Lock()
		r.refill()
	}

	r.tokens--
	r.lastRequest = time.Now()
}

// TryAcquire attempts to acquire a token without blocking.
// Returns true if a token was acquired, false otherwise.
func (r *RateLimiter) TryAcquire() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()

	if r.tokens >= 1 {
		r.tokens--
		r.lastRequest = time.Now()
		return true
	}
	return false
}

// refill adds tokens based on elapsed time.
// Must be called with mutex held.
func (r *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens += elapsed * r.refillRate

	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}

	r.lastRefill = now
}

// BitgetRateLimiter provides pre-configured rate limiters for Bitget API.
// Bitget limits: 10 requests/second for most endpoints.
var (
	bitgetOrderLimiter   *RateLimiter
	bitgetAccountLimiter *RateLimiter
	bitgetMarketLimiter  *RateLimiter
	rateLimiterOnce      sync.Once
)

// GetBitgetOrderLimiter returns the rate limiter for order endpoints.
// Limit: 10 requests/second with burst of 5.
func GetBitgetOrderLimiter() *RateLimiter {
	rateLimiterOnce.Do(initBitgetLimiters)
	return bitgetOrderLimiter
}

// GetBitgetAccountLimiter returns the rate limiter for account endpoints.
// Limit: 10 requests/second with burst of 5.
func GetBitgetAccountLimiter() *RateLimiter {
	rateLimiterOnce.Do(initBitgetLimiters)
	return bitgetAccountLimiter
}

// GetBitgetMarketLimiter returns the rate limiter for market data endpoints.
// Limit: 20 requests/second with burst of 10.
func GetBitgetMarketLimiter() *RateLimiter {
	rateLimiterOnce.Do(initBitgetLimiters)
	return bitgetMarketLimiter
}

func initBitgetLimiters() {
	// Conservative limits to avoid IP bans
	bitgetOrderLimiter = NewRateLimiter(5, 10)   // 10 req/s, burst 5
	bitgetAccountLimiter = NewRateLimiter(5, 10) // 10 req/s, burst 5
	bitgetMarketLimiter = NewRateLimiter(10, 20) // 20 req/s, burst 10
}
