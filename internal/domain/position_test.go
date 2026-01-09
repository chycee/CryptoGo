package domain

import "testing"

func TestPosition_Direction(t *testing.T) {
	tests := []struct {
		name    string
		qty     int64
		isLong  bool
		isShort bool
	}{
		{"Long", 100, true, false},
		{"Short", -100, false, true},
		{"Flat", 0, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Position{QtySats: tt.qty}
			if got := p.IsLong(); got != tt.isLong {
				t.Errorf("Position.IsLong() = %v, want %v", got, tt.isLong)
			}
			if got := p.IsShort(); got != tt.isShort {
				t.Errorf("Position.IsShort() = %v, want %v", got, tt.isShort)
			}
		})
	}
}
