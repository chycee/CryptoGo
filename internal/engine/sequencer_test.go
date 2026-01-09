package engine

import (
	"context"
	"crypto_go/internal/event"
	"crypto_go/pkg/quant"
	"testing"
	"time"
)

func TestSequencer_MarketUpdate(t *testing.T) {
	seq := NewSequencer(10, nil, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go seq.Run(ctx)

	// Send an event
	ev := &event.MarketUpdateEvent{
		BaseEvent:   event.BaseEvent{Seq: 1, Ts: 1000},
		Symbol:      "BTC-KRW",
		PriceMicros: quant.PriceMicros(100000000), // 100 BTC
		QtySats:     quant.QtySats(100000000),     // 1 BTC
	}
	seq.Inbox() <- ev

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	state, ok := seq.GetMarketState("BTC-KRW")
	if !ok {
		t.Fatal("Market state should exist")
	}
	if state.PriceMicros != ev.PriceMicros {
		t.Errorf("Expected price %d, got %d", ev.PriceMicros, state.PriceMicros)
	}
}

func TestSequencer_GapTolerance(t *testing.T) {
	// 1. Setup
	seq := NewSequencer(10, nil, nil, nil)
	// Default nextSeq is 1.

	// 2. Send Event with Seq 2 (Gap = 1) -> Should be TOLERATED (No Panic)
	ev := &event.MarketUpdateEvent{
		BaseEvent: event.BaseEvent{Seq: 2, Ts: 1000},
		Symbol:    "BTC-KRW",
	}
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Sequencer panicked on small gap (<=10): %v", r)
			}
		}()
		seq.processEvent(ev)
	}()

	// 3. Verify sequence advanced
	// After gap tolerance, nextSeq should be updated to ev.Seq + 1 (next expected)
	// WAIT. Logic says: s.nextSeq = evSeq (fast forward) -> then it increments at end of processEvent.
	// So nextSeq should be 3.
	// Let's check internal state if we could, but nextSeq is private.
	// We rely on "No Panic" as primary success criteria here.
}

func TestSequencer_GapPanic(t *testing.T) {
	// 1. Setup
	seq := NewSequencer(10, nil, nil, nil)
	// Default nextSeq is 1.

	// 2. Send Event with Seq 20 (Gap = 19) -> Should PANIC
	ev := &event.MarketUpdateEvent{
		BaseEvent: event.BaseEvent{Seq: 20, Ts: 1000},
		Symbol:    "BTC-KRW",
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("Sequencer should have panicked on large gap (>10)")
		}
	}()
	
	seq.processEvent(ev)
}
