package storage

import (
	"context"
	"crypto_go/internal/event"
	"crypto_go/pkg/quant"
	"os"
	"testing"
)

func TestEventStore_SaveAndLoad(t *testing.T) {
	// Use temp file for test DB
	dbPath := "test_events.db"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + "-wal")
	defer os.Remove(dbPath + "-shm")

	store, err := NewEventStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create test events using BaseEvent embedding
	ev1 := &event.MarketUpdateEvent{
		BaseEvent:   event.BaseEvent{Seq: 1, Ts: quant.TimeStamp(1000)},
		Symbol:      "BTCUSDT",
		PriceMicros: 50000000000,
		QtySats:     100000000,
		Exchange:    "BITGET",
	}
	ev2 := &event.MarketUpdateEvent{
		BaseEvent:   event.BaseEvent{Seq: 2, Ts: quant.TimeStamp(2000)},
		Symbol:      "BTCUSDT",
		PriceMicros: 51000000000,
		QtySats:     200000000,
		Exchange:    "BITGET",
	}

	// Save events
	if err := store.SaveEvent(ctx, ev1); err != nil {
		t.Fatalf("Failed to save ev1: %v", err)
	}
	if err := store.SaveEvent(ctx, ev2); err != nil {
		t.Fatalf("Failed to save ev2: %v", err)
	}

	// Load events
	loaded, err := store.LoadEvents(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to load events: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(loaded))
	}

	// Verify first event
	if loaded[0].GetSeq() != 1 {
		t.Errorf("Event 1 seq mismatch: got %d", loaded[0].GetSeq())
	}
	if loaded[0].PriceMicros != 50000000000 {
		t.Errorf("Event 1 price mismatch: got %d", loaded[0].PriceMicros)
	}

	// Verify second event
	if loaded[1].GetSeq() != 2 {
		t.Errorf("Event 2 seq mismatch: got %d", loaded[1].GetSeq())
	}
}

func TestEventStore_GetLastSeq(t *testing.T) {
	dbPath := "test_lastseq.db"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + "-wal")
	defer os.Remove(dbPath + "-shm")

	store, err := NewEventStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Empty DB should return 0
	lastSeq, err := store.GetLastSeq(ctx)
	if err != nil {
		t.Fatalf("GetLastSeq failed: %v", err)
	}
	if lastSeq != 0 {
		t.Errorf("Expected 0 for empty DB, got %d", lastSeq)
	}

	// Add events
	ev := &event.MarketUpdateEvent{
		BaseEvent: event.BaseEvent{Seq: 5, Ts: quant.TimeStamp(1000)},
		Symbol:    "TEST",
	}
	if err := store.SaveEvent(ctx, ev); err != nil {
		t.Fatalf("Failed to save event: %v", err)
	}

	ev2 := &event.MarketUpdateEvent{
		BaseEvent: event.BaseEvent{Seq: 10, Ts: quant.TimeStamp(2000)},
		Symbol:    "TEST",
	}
	if err := store.SaveEvent(ctx, ev2); err != nil {
		t.Fatalf("Failed to save event: %v", err)
	}

	// Should return highest seq
	lastSeq, err = store.GetLastSeq(ctx)
	if err != nil {
		t.Fatalf("GetLastSeq failed: %v", err)
	}
	if lastSeq != 10 {
		t.Errorf("Expected 10, got %d", lastSeq)
	}
}
