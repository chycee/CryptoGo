package infra

import (
	"testing"
	"time"
)

// =====================================================
// Infra Backoff Tests
// =====================================================

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		retryCount int
		minDelay   time.Duration
		maxDelay   time.Duration
	}{
		{0, 1 * time.Second, 1 * time.Second},     // 1s
		{1, 2 * time.Second, 2 * time.Second},     // 2s
		{2, 4 * time.Second, 4 * time.Second},     // 4s
		{3, 8 * time.Second, 8 * time.Second},     // 8s
		{10, 60 * time.Second, 60 * time.Second},  // max 60s
		{100, 60 * time.Second, 60 * time.Second}, // still max 60s
	}

	for _, tt := range tests {
		delay := CalculateBackoff(tt.retryCount)
		if delay < tt.minDelay || delay > tt.maxDelay {
			t.Errorf("CalculateBackoff(%d) = %s, want between %s and %s",
				tt.retryCount, delay, tt.minDelay, tt.maxDelay)
		}
	}
}
