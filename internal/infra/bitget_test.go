package infra

import (
	"testing"
)

// =====================================================
// Bitget Precision Tests
// =====================================================

func TestCalculateBitgetBackoff(t *testing.T) {
	tests := []struct {
		retryCount int
		minDelay   int64 // milliseconds
		maxDelay   int64 // milliseconds
	}{
		{0, 1000, 1000},     // 1s
		{1, 2000, 2000},     // 2s
		{2, 4000, 4000},     // 4s
		{3, 8000, 8000},     // 8s
		{10, 60000, 60000},  // max 60s
		{100, 60000, 60000}, // still max 60s
	}

	for _, tt := range tests {
		delay := calculateBitgetBackoff(tt.retryCount)
		delayMs := delay.Milliseconds()
		if delayMs < tt.minDelay || delayMs > tt.maxDelay {
			t.Errorf("calculateBitgetBackoff(%d) = %dms, want between %d and %d",
				tt.retryCount, delayMs, tt.minDelay, tt.maxDelay)
		}
	}
}
