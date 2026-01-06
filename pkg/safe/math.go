package safe

import (
	"math"
)

// SafeAdd performs int64 addition and panics on overflow/underflow.
func SafeAdd(a, b int64) int64 {
	if (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b) {
		panic("CORE_SAFE_ADD_OVERFLOW")
	}
	return a + b
}

// SafeSub performs int64 subtraction and panics on overflow/underflow.
func SafeSub(a, b int64) int64 {
	if (b > 0 && a < math.MinInt64+b) || (b < 0 && a > math.MaxInt64+b) {
		panic("CORE_SAFE_SUB_OVERFLOW")
	}
	return a - b
}

// SafeMul performs int64 multiplication and panics on overflow/underflow.
func SafeMul(a, b int64) int64 {
	if a == 0 || b == 0 {
		return 0
	}
	if a > 0 {
		if b > 0 {
			if a > math.MaxInt64/b {
				panic("CORE_SAFE_MUL_OVERFLOW")
			}
		} else {
			if b < math.MinInt64/a {
				panic("CORE_SAFE_MUL_OVERFLOW")
			}
		}
	} else {
		if b > 0 {
			if a < math.MinInt64/b {
				panic("CORE_SAFE_MUL_OVERFLOW")
			}
		} else {
			if a < math.MaxInt64/b {
				panic("CORE_SAFE_MUL_OVERFLOW")
			}
		}
	}
	return a * b
}

// SafeDiv performs int64 division and panics on division by zero.
func SafeDiv(a, b int64) int64 {
	if b == 0 {
		panic("CORE_SAFE_DIV_BY_ZERO")
	}
	// Note: int64 MinInt64 / -1 also overflows, but it's rare.
	if a == math.MinInt64 && b == -1 {
		panic("CORE_SAFE_DIV_OVERFLOW")
	}
	return a / b
}
