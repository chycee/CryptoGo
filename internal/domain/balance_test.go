package domain

import (
	"testing"
)

func TestBalance_CreditDebit(t *testing.T) {
	b := &Balance{Symbol: "BTC"}

	// Credit 100 sats
	b.Credit(100, 1)
	if b.AmountSats != 100 {
		t.Errorf("expected 100, got %d", b.AmountSats)
	}

	// Debit 30 sats
	b.Debit(30, 2)
	if b.AmountSats != 70 {
		t.Errorf("expected 70, got %d", b.AmountSats)
	}

	// Invariant should pass
	b.VerifyInvariant()
}

func TestBalance_Reserve(t *testing.T) {
	b := &Balance{Symbol: "ETH", AmountSats: 1000}

	// Reserve 400
	b.Reserve(400, 1)
	if b.ReservedSats != 400 {
		t.Errorf("expected reserved 400, got %d", b.ReservedSats)
	}
	if b.AvailableSats() != 600 {
		t.Errorf("expected available 600, got %d", b.AvailableSats())
	}

	// Release 200
	b.Release(200, 2)
	if b.ReservedSats != 200 {
		t.Errorf("expected reserved 200, got %d", b.ReservedSats)
	}

	b.VerifyInvariant()
}

func TestBalance_InvariantPanic_NegativeAmount(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for negative amount")
		}
	}()

	b := &Balance{Symbol: "BTC", AmountSats: -1}
	b.VerifyInvariant()
}

func TestBalance_InvariantPanic_ReservedExceedsAmount(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when reserved > amount")
		}
	}()

	b := &Balance{Symbol: "BTC", AmountSats: 100, ReservedSats: 200}
	b.VerifyInvariant()
}

func TestBalance_DebitPanic_Insufficient(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for insufficient balance")
		}
	}()

	b := &Balance{Symbol: "BTC", AmountSats: 50}
	b.Debit(100, 1) // Should panic
}

func TestBalanceBook(t *testing.T) {
	bb := NewBalanceBook()

	btc := bb.Get("BTC")
	btc.Credit(1000, 1)

	eth := bb.Get("ETH")
	eth.Credit(5000, 2)

	// Verify all
	bb.VerifyAll()

	// Snapshot
	snap := bb.Snapshot()
	if len(snap) != 2 {
		t.Errorf("expected 2 balances, got %d", len(snap))
	}
}
