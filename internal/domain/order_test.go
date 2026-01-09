package domain

import "testing"

func TestOrder_IsOpen(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"NEW", "NEW", true},
		{"PARTIALL_FILLED", "PARTIALLY_FILLED", true},
		{"FILLED", "FILLED", false},
		{"CANCELED", "CANCELED", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Order{Status: tt.status}
			if got := o.IsOpen(); got != tt.want {
				t.Errorf("Order.IsOpen() = %v, want %v", got, tt.want)
			}
		})
	}
}
