package strategy

import (
	"crypto_go/internal/domain"
)

// Strategy defines the interface for trading logic.
type Strategy interface {
	// OnMarketUpdate is called when new market data (price/qty) arrives.
	// It returns the number of signals written to the 'out' buffer.
	// Zero-Alloc: Caller provides the 'out' slice to avoid heap allocations.
	OnMarketUpdate(state domain.MarketState, out []domain.Order) int

	// OnOrderUpdate is called when an order status changes (Filled, Canceled, etc).
	OnOrderUpdate(order domain.Order)
}
