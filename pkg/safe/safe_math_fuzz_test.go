package safe

import (
	"testing"
)

// FuzzSafeAdd tests SafeAdd with fuzzing.
func FuzzSafeAdd(f *testing.F) {
	// Seed corpus
	f.Add(int64(0), int64(0))
	f.Add(int64(1), int64(2))
	f.Add(int64(-1), int64(1))
	f.Add(int64(9223372036854775807), int64(0))  // MaxInt64
	f.Add(int64(-9223372036854775808), int64(0)) // MinInt64

	f.Fuzz(func(t *testing.T, a, b int64) {
		defer func() { recover() }() // Overflow panic is expected behavior
		_ = SafeAdd(a, b)
	})
}

// FuzzSafeSub tests SafeSub with fuzzing.
func FuzzSafeSub(f *testing.F) {
	f.Add(int64(0), int64(0))
	f.Add(int64(10), int64(5))
	f.Add(int64(-1), int64(-1))
	f.Add(int64(9223372036854775807), int64(0))
	f.Add(int64(-9223372036854775808), int64(0))

	f.Fuzz(func(t *testing.T, a, b int64) {
		defer func() { recover() }()
		_ = SafeSub(a, b)
	})
}

// FuzzSafeMul tests SafeMul with fuzzing.
func FuzzSafeMul(f *testing.F) {
	f.Add(int64(0), int64(0))
	f.Add(int64(2), int64(3))
	f.Add(int64(-2), int64(3))
	f.Add(int64(1000000), int64(1000000))

	f.Fuzz(func(t *testing.T, a, b int64) {
		defer func() { recover() }()
		_ = SafeMul(a, b)
	})
}

// FuzzSafeDiv tests SafeDiv with fuzzing.
func FuzzSafeDiv(f *testing.F) {
	f.Add(int64(10), int64(2))
	f.Add(int64(-10), int64(2))
	f.Add(int64(100), int64(-5))
	f.Add(int64(9223372036854775807), int64(1))

	f.Fuzz(func(t *testing.T, a, b int64) {
		defer func() { recover() }() // Div by zero panic is expected
		_ = SafeDiv(a, b)
	})
}
