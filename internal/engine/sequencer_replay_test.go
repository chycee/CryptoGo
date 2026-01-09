package engine

import (
	"context"
	"os"
	"testing"

	"crypto_go/internal/event"
	"crypto_go/internal/storage"
	"crypto_go/pkg/quant"
)

// TestSequencer_Replay_EmptyWAL tests replay with no events.
func TestSequencer_Replay_EmptyWAL(t *testing.T) {
	tempDB := t.TempDir() + "/test_empty.db"
	defer os.Remove(tempDB)

	store, err := storage.NewEventStore(tempDB)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	sequencer := NewSequencer(100, store, nil, nil)

	// Should not error on empty WAL
	if err := sequencer.RecoverFromWAL(ctx); err != nil {
		t.Fatalf("RecoverFromWAL failed on empty WAL: %v", err)
	}

	// nextSeq should be 1 (starting value)
	if sequencer.GetNextSeq() != 1 {
		t.Errorf("expected nextSeq=1, got %d", sequencer.GetNextSeq())
	}
}

// TestSequencer_Replay_SingleEvent tests replay of a single event.
// This verifies "Backtest is Reality" principle for simple case.
func TestSequencer_Replay_SingleEvent(t *testing.T) {
	tempDB := t.TempDir() + "/test_single.db"
	defer os.Remove(tempDB)

	store, err := storage.NewEventStore(tempDB)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create first sequencer
	sequencer1 := NewSequencer(100, store, nil, nil)

	// Create and process event with correct sequence number
	marketEvent := event.AcquireMarketUpdateEvent()
	marketEvent.Seq = 1 // Set correct sequence
	marketEvent.Ts = quant.TimeStamp(1704067200000000)
	marketEvent.Symbol = "BTC"
	marketEvent.Exchange = "UPBIT"
	marketEvent.PriceMicros = 134109000_000000
	marketEvent.QtySats = 12345678

	// Process directly (simulating event from inbox)
	sequencer1.ProcessEventForTest(marketEvent)

	originalPrice := sequencer1.GetMarketPrice("UPBIT", "BTC")
	originalNextSeq := sequencer1.GetNextSeq()
	t.Logf("Original: price=%d, nextSeq=%d", originalPrice, originalNextSeq)

	// Create new sequencer and replay
	sequencer2 := NewSequencer(100, store, nil, nil)
	if err := sequencer2.RecoverFromWAL(ctx); err != nil {
		t.Fatalf("RecoverFromWAL failed: %v", err)
	}

	replayedPrice := sequencer2.GetMarketPrice("UPBIT", "BTC")
	replayedNextSeq := sequencer2.GetNextSeq()
	t.Logf("Replayed: price=%d, nextSeq=%d", replayedPrice, replayedNextSeq)

	// Assertions
	if originalPrice != replayedPrice {
		t.Errorf("price mismatch: original=%d, replayed=%d", originalPrice, replayedPrice)
	}

	if originalNextSeq != replayedNextSeq {
		t.Errorf("nextSeq mismatch: original=%d, replayed=%d", originalNextSeq, replayedNextSeq)
	}
}
