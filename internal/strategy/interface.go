package strategy

import (
	"crypto_go/internal/domain"
)

// Strategy defines the interface for trading logic.
type Strategy interface {
	// OnMarketUpdate is called when new market data (price/qty) arrives.
	// It returns a list of new orders to be executed.
	OnMarketUpdate(state domain.MarketState) []domain.Order

	// OnOrderUpdate is called when an order status changes (Filled, Canceled, etc).
	OnOrderUpdate(order domain.Order)
}
