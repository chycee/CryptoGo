package engine

import (
	"context"
	"crypto_go/internal/event"
	"crypto_go/pkg/quant"
	"testing"
	"time"
)

func TestSequencer_MarketUpdate(t *testing.T) {
	seq := NewSequencer(10, nil, nil)
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

func TestSequencer_GapDetection(t *testing.T) {
	seq := NewSequencer(10, nil, nil)

	// Should panic when receiving out-of-order event
	defer func() {
		if r := recover(); r == nil {
			t.Error("Sequencer should have panicked on sequence gap")
		}
	}()

	ev := &event.MarketUpdateEvent{
		BaseEvent: event.BaseEvent{Seq: 2, Ts: 1000}, // Start with 2 instead of 1
		Symbol:    "BTC-KRW",
	}
	seq.processEvent(ev)
}
