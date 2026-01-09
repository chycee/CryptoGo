package execution

import (
	"context"
	"crypto_go/internal/domain"
)

// Execution defines the interface for order execution.
type Execution interface {
	// SubmitOrder sends a new order to the exchange.
	SubmitOrder(ctx context.Context, order domain.Order) error

	// CancelOrder cancels an existing order by ID.
	CancelOrder(ctx context.Context, orderID string) error
}
