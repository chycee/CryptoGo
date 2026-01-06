package domain

import (
	"crypto_go/pkg/safe"
	"fmt"
)

// Balance represents account balance with invariant checking.
// This is the core structure for Balance Invariant verification.
type Balance struct {
	Symbol       string `json:"symbol"`
	AmountSats   int64  `json:"amount"`   // Current balance (Sats)
	ReservedSats int64  `json:"reserved"` // Reserved for open orders
	LastSeq      uint64 `json:"last_seq"` // Last event sequence that modified this
}

// AvailableSats returns the available balance (total - reserved).
func (b *Balance) AvailableSats() int64 {
	return safe.SafeSub(b.AmountSats, b.ReservedSats)
}

// Credit adds funds to the balance. Panics on overflow.
func (b *Balance) Credit(amountSats int64, seq uint64) {
	b.AmountSats = safe.SafeAdd(b.AmountSats, amountSats)
	b.LastSeq = seq
}

// Debit removes funds from the balance. Panics if insufficient or overflow.
func (b *Balance) Debit(amountSats int64, seq uint64) {
	if amountSats > b.AvailableSats() {
		panic(fmt.Sprintf("BALANCE_INSUFFICIENT: %s need %d, available %d",
			b.Symbol, amountSats, b.AvailableSats()))
	}
	b.AmountSats = safe.SafeSub(b.AmountSats, amountSats)
	b.LastSeq = seq
}

// Reserve locks funds for an order.
func (b *Balance) Reserve(amountSats int64, seq uint64) {
	if amountSats > b.AvailableSats() {
		panic(fmt.Sprintf("BALANCE_RESERVE_INSUFFICIENT: %s need %d, available %d",
			b.Symbol, amountSats, b.AvailableSats()))
	}
	b.ReservedSats = safe.SafeAdd(b.ReservedSats, amountSats)
	b.LastSeq = seq
}

// Release unlocks reserved funds.
func (b *Balance) Release(amountSats int64, seq uint64) {
	if amountSats > b.ReservedSats {
		panic(fmt.Sprintf("BALANCE_RELEASE_EXCEEDS_RESERVED: %s release %d, reserved %d",
			b.Symbol, amountSats, b.ReservedSats))
	}
	b.ReservedSats = safe.SafeSub(b.ReservedSats, amountSats)
	b.LastSeq = seq
}

// VerifyInvariant checks that balance satisfies invariants.
// Call this after any state change to ensure data integrity.
func (b *Balance) VerifyInvariant() {
	// Invariant 1: Amount must be non-negative
	if b.AmountSats < 0 {
		panic(fmt.Sprintf("BALANCE_INVARIANT_NEGATIVE_AMOUNT: %s = %d",
			b.Symbol, b.AmountSats))
	}

	// Invariant 2: Reserved must be non-negative
	if b.ReservedSats < 0 {
		panic(fmt.Sprintf("BALANCE_INVARIANT_NEGATIVE_RESERVED: %s = %d",
			b.Symbol, b.ReservedSats))
	}

	// Invariant 3: Reserved cannot exceed Amount
	if b.ReservedSats > b.AmountSats {
		panic(fmt.Sprintf("BALANCE_INVARIANT_RESERVED_EXCEEDS_AMOUNT: %s reserved=%d, amount=%d",
			b.Symbol, b.ReservedSats, b.AmountSats))
	}
}

// BalanceBook manages multiple balances with invariant checking.
type BalanceBook struct {
	balances map[string]*Balance
}

// NewBalanceBook creates a new balance book.
func NewBalanceBook() *BalanceBook {
	return &BalanceBook{
		balances: make(map[string]*Balance),
	}
}

// Get returns the balance for a symbol, creating if not exists.
func (bb *BalanceBook) Get(symbol string) *Balance {
	b, ok := bb.balances[symbol]
	if !ok {
		b = &Balance{Symbol: symbol}
		bb.balances[symbol] = b
	}
	return b
}

// VerifyAll checks invariants on all balances.
func (bb *BalanceBook) VerifyAll() {
	for _, b := range bb.balances {
		b.VerifyInvariant()
	}
}

// Snapshot returns a copy of all balances (for state dump).
func (bb *BalanceBook) Snapshot() map[string]Balance {
	result := make(map[string]Balance, len(bb.balances))
	for k, v := range bb.balances {
		result[k] = *v
	}
	return result
}
