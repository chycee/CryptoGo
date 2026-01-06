package safe

import (
	"math"
	"testing"
)

func TestSafeMath(t *testing.T) {
	tests := []struct {
		name string
		val1 int64
		val2 int64
		want int64
	}{
		{"Normal Add", 10, 20, 30},
		{"Add Boundary", math.MaxInt64 - 1, 1, math.MaxInt64},
		{"Normal Sub", 30, 10, 20},
		{"Normal Mul", 5, 6, 30},
		{"Normal Div", 100, 4, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got int64
			switch tt.name {
			case "Normal Add", "Add Boundary":
				got = SafeAdd(tt.val1, tt.val2)
			case "Normal Sub":
				got = SafeSub(tt.val1, tt.val2)
			case "Normal Mul":
				got = SafeMul(tt.val1, tt.val2)
			case "Normal Div":
				got = SafeDiv(tt.val1, tt.val2)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMathPanic(t *testing.T) {
	t.Run("Add Overflow", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Should have panicked")
			}
		}()
		SafeAdd(math.MaxInt64, 1)
	})

	t.Run("Div By Zero", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Should have panicked")
			}
		}()
		SafeDiv(10, 0)
	})
}
