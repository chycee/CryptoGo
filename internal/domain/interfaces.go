package domain

import (
	"context"

	"crypto_go/pkg/quant"
)

// ExchangeWorker defines the interface for exchange WebSocket connectors
type ExchangeWorker interface {
	Connect(ctx context.Context) error
	Disconnect()
	IsConnected() bool
}

// ExchangeRateProvider defines the interface for currency exchange rate sources
type ExchangeRateProvider interface {
	Start(ctx context.Context) error
	GetRate() quant.PriceMicros
}

// MarketDataRepository defines how to access market data (for future persistence or memory storage)
type MarketDataRepository interface {
	Save(data *MarketData) error
	FindAll() ([]*MarketData, error)
	FindBySymbol(symbol string) (*MarketData, error)
}
