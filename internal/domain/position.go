package domain

// Position represents an open trading position.
// All monetary values are strictly int64.
type Position struct {
	Symbol              string
	QtySats             int64 // Positive for Long, Negative for Short.
	AvgEntryPriceMicros int64 // Weighted Average Entry Price.
	RealizedPnLMicros   int64 // Realized Profit/Loss.
}

// IsLong checks if the position is Long.
func (p *Position) IsLong() bool {
	return p.QtySats > 0
}

// IsShort checks if the position is Short.
func (p *Position) IsShort() bool {
	return p.QtySats < 0
}

// IsProductTypeCompatible check logic if needed in future
