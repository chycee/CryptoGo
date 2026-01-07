package bitget

import (
	"crypto_go/internal/infra"
	"log/slog"
)

// Constants for Bitget API URLs
const (
	MainnetURL = "https://api.bitget.com"
	TestnetURL = "https://api.bitget.com" // Bitget V2 uses same endpoint but different keys? Or verify docs.
	// Typically V2 Mix URL: https://api.bitget.com
)

// Client handles Bitget REST API communication for Order Execution
type Client struct {
	config    *infra.Config
	isTestnet bool
	baseURL   string
}

// NewClient creates a new Bitget REST client
func NewClient(cfg *infra.Config, isTestnet bool) *Client {
	baseURL := MainnetURL
	// In the future, if Testnet uses different URL, switch here.
	// For Bitget, sometimes it's driven by the keys or specific Simulation endpoints.
	// For now, we assume MainnetURL but will handle logic based on isTestnet flag.

	return &Client{
		config:    cfg,
		isTestnet: isTestnet,
		baseURL:   baseURL,
	}
}

// PlaceOrder submits an order to Bitget
// TODO: Implement actual API call
func (c *Client) PlaceOrder(symbol string, side string, price, qty int64) error {
	mode := "REAL"
	if c.isTestnet {
		mode = "TESTNET"
	}
	slog.Info("Bitget Client: PlaceOrder", "mode", mode, "symbol", symbol, "side", side)
	return nil
}

// CancelOrder cancels an order on Bitget
// TODO: Implement actual API call
func (c *Client) CancelOrder(orderID string) error {
	slog.Info("Bitget Client: CancelOrder", "orderID", orderID)
	return nil
}
