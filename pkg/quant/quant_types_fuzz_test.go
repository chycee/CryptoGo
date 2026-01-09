package quant

import (
	"testing"
)

// FuzzToPriceMicros tests price conversion with fuzzing.
func FuzzToPriceMicros(f *testing.F) {
	f.Add(0.0)
	f.Add(1.23)
	f.Add(-1.23)
	f.Add(0.000001)
	f.Add(9999999.999999)

	f.Fuzz(func(t *testing.T, val float64) {
		// This should never panic, just validate it doesn't crash
		_ = ToPriceMicros(val)
	})
}

// FuzzToQtySats tests quantity conversion with fuzzing.
func FuzzToQtySats(f *testing.F) {
	f.Add(0.0)
	f.Add(1.0)
	f.Add(0.00000001)
	f.Add(21000000.0) // Max BTC supply

	f.Fuzz(func(t *testing.T, val float64) {
		_ = ToQtySats(val)
	})
}

// FuzzParseTimeStamp tests timestamp parsing with fuzzing.
func FuzzParseTimeStamp(f *testing.F) {
	f.Add("0")
	f.Add("1704067200000") // 2024-01-01 00:00:00 UTC in ms
	f.Add("-1")
	f.Add("9223372036854775807")

	f.Fuzz(func(t *testing.T, s string) {
		// Should handle invalid input gracefully (return error, not panic)
		_, _ = ParseTimeStamp(s)
	})
}
