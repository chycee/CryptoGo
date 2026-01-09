package execution

import (
	"context"
	"testing"

	"crypto_go/internal/domain"
)

func TestMockExecution_ImplementsInterface(t *testing.T) {
	var _ Execution = (*MockExecution)(nil) // Compile-time check
}

func TestMockExecution_SubmitOrder(t *testing.T) {
	mock := NewMockExecution()
	order := domain.Order{
		ID:          "test-order-1",
		Symbol:      "BTCUSDT",
		PriceMicros: 100000000,
		QtySats:     10000,
	}

	if err := mock.SubmitOrder(context.Background(), order); err != nil {
		t.Errorf("SubmitOrder failed: %v", err)
	}
}
