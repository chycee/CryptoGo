package execution

import (
	"context"
	"crypto_go/internal/domain"
	"crypto_go/pkg/quant"
	"testing"
)

func TestPaperExecution_Buy(t *testing.T) {
	paper := NewPaperExecution(0)

	// Setup: deposit 10000 USDT
	paper.Deposit("USDT", 10000_000000)        // 10000 USDT in Sats
	paper.UpdatePrice("BTCUSDT", 50000_000000) // 50000 USDT/BTC

	// Buy 0.1 BTC
	order := domain.Order{
		ID:      "order-1",
		Symbol:  "BTCUSDT",
		Side:    "BUY",
		Type:    "MARKET",
		QtySats: 10_000000, // 0.1 BTC in Sats
	}

	err := paper.ExecuteOrder(context.Background(), order)
	if err != nil {
		t.Fatalf("ExecuteOrder failed: %v", err)
	}

	// Verify BTC balance
	btcBalance := paper.GetBalance("BTC")
	if btcBalance.AmountSats != 10_000000 {
		t.Errorf("Expected 10000000 BTC sats, got %d", btcBalance.AmountSats)
	}

	// Verify USDT balance (should be 10000 - 5000 = 5000)
	// 0.1 BTC * 50000 = 5000 USDT
	usdtBalance := paper.GetBalance("USDT")
	expectedUSDT := int64(10000_000000 - 5000_000000)
	if usdtBalance.AmountSats != expectedUSDT {
		t.Errorf("Expected %d USDT sats, got %d", expectedUSDT, usdtBalance.AmountSats)
	}

	// Verify fills
	fills := paper.GetFills()
	if len(fills) != 1 {
		t.Fatalf("Expected 1 fill, got %d", len(fills))
	}
	if fills[0].Side != "BUY" {
		t.Errorf("Expected BUY, got %s", fills[0].Side)
	}
}

func TestPaperExecution_Sell(t *testing.T) {
	paper := NewPaperExecution(0)

	// Setup: deposit 1 BTC
	paper.Deposit("BTC", 100_000000)           // 1 BTC in Sats
	paper.UpdatePrice("BTCUSDT", 50000_000000) // 50000 USDT/BTC

	// Sell 0.5 BTC
	order := domain.Order{
		ID:      "order-2",
		Symbol:  "BTCUSDT",
		Side:    "SELL",
		Type:    "MARKET",
		QtySats: 50_000000, // 0.5 BTC in Sats
	}

	err := paper.ExecuteOrder(context.Background(), order)
	if err != nil {
		t.Fatalf("ExecuteOrder failed: %v", err)
	}

	// Verify BTC balance (should be 0.5 BTC left)
	btcBalance := paper.GetBalance("BTC")
	if btcBalance.AmountSats != 50_000000 {
		t.Errorf("Expected 50000000 BTC sats, got %d", btcBalance.AmountSats)
	}

	// Verify USDT balance (should be 25000 USDT)
	usdtBalance := paper.GetBalance("USDT")
	expectedUSDT := int64(25000_000000)
	if usdtBalance.AmountSats != expectedUSDT {
		t.Errorf("Expected %d USDT sats, got %d", expectedUSDT, usdtBalance.AmountSats)
	}
}

func TestPaperExecution_InsufficientBalance(t *testing.T) {
	paper := NewPaperExecution(quant.ToPriceMicros(100_000_000))

	// Setup: deposit only 100 USDT
	paper.Deposit("USDT", 100_000000)
	paper.UpdatePrice("BTCUSDT", 50000_000000)

	// Try to buy 1 BTC (need 50000 USDT)
	order := domain.Order{
		ID:      "order-3",
		Symbol:  "BTCUSDT",
		Side:    "BUY",
		Type:    "MARKET",
		QtySats: 100_000000, // 1 BTC
	}

	err := paper.ExecuteOrder(context.Background(), order)
	if err == nil {
		t.Fatal("Expected error for insufficient balance, got nil")
	}
}

func TestPaperExecution_ImplementsInterface(t *testing.T) {
	var _ domain.Execution = (*PaperExecution)(nil)
}
