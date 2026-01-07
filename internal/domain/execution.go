package domain

import "context"

// Execution interface defines the contract for order execution systems.
// It abstracts away the difference between Paper Trading, Demo, and Real exchanges.
type Execution interface {
	// ExecuteOrder submits an order to the execution venue.
	ExecuteOrder(ctx context.Context, order Order) error

	// CancelOrder cancels an existing order.
	CancelOrder(ctx context.Context, orderID string, symbol string) error

	// Close cleans up resources and wipes secrets.
	Close() error
}
