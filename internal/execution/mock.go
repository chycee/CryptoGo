package execution

import (
	"context"
	"log/slog"

	"crypto_go/internal/domain"
)

// MockExecution is a safe implementation that only logs orders.
type MockExecution struct {
	// Can add channels here to capture orders for testing if needed
}

func NewMockExecution() *MockExecution {
	return &MockExecution{}
}

func (m *MockExecution) SubmitOrder(ctx context.Context, order domain.Order) error {
	slog.Info("MOCK EXECUTION: Submit Order",
		slog.String("id", order.ID),
		slog.String("symbol", order.Symbol),
		slog.String("side", order.Side),
		slog.Int64("price", order.PriceMicros),
		slog.Int64("qty", order.QtySats),
	)
	return nil
}

func (m *MockExecution) CancelOrder(ctx context.Context, orderID string) error {
	slog.Info("MOCK EXECUTION: Cancel Order", slog.String("id", orderID))
	return nil
}
