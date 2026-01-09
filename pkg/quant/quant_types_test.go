package quant

import (
	"testing"
)

func TestToPriceMicros(t *testing.T) {
	tests := []struct {
		input    float64
		expected PriceMicros
	}{
		{1.23, 1230000},
		{0.000001, 1},
		{0.0, 0},
		{-1.23, -1230000},
	}

	for _, tt := range tests {
		got := ToPriceMicros(tt.input)
		if got != tt.expected {
			t.Errorf("ToPriceMicros(%f) = %d; want %d", tt.input, got, tt.expected)
		}
	}
}

func TestPriceMicros_String(t *testing.T) {
	p := PriceMicros(1230000)
	expected := "1.230000"
	if p.String() != expected {
		t.Errorf("PriceMicros(1230000).String() = %s; want %s", p.String(), expected)
	}
}
