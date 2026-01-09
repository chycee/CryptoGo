package infra

import (
	"time"
)

const (
	// Standard backoff constants
	baseDelay = 1 * time.Second
	maxDelay  = 60 * time.Second
)

// CalculateBackoff returns the exponential backoff duration for a given retry count.
// Logic: baseDelay * 2^retryCount, capped at maxDelay.
// If retryCount is negative, it returns baseDelay.
func CalculateBackoff(retryCount int) time.Duration {
	if retryCount < 0 {
		return baseDelay
	}

	// 2^retryCount
	// To prevent overflow with bit shifting, we check explicitly or cap it early.
	// 2^30 is already > 1 billion seconds > maxDelay.
	if retryCount > 30 {
		return maxDelay
	}

	backoff := baseDelay * time.Duration(1<<retryCount)

	if backoff > maxDelay {
		return maxDelay
	}

	return backoff
}
