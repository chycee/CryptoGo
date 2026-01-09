package engine

import (
	"context"
	"crypto_go/internal/event"
	"crypto_go/pkg/quant"
	"testing"
)

// BenchmarkSequencer_ProcessEvent measures Hotpath event processing speed.
// This is the core metric for "Zero-Alloc in Hotpath" principle verification.
func BenchmarkSequencer_ProcessEvent(b *testing.B) {
	seq := NewSequencer(1000, nil, nil, nil)

	// Pre-create event to avoid allocation in loop
	ev := event.AcquireMarketUpdateEvent()
	ev.Seq = 1
	ev.Ts = quant.TimeStamp(1000)
	ev.Symbol = "BTCUSDT"
	ev.PriceMicros = 50000000000
	ev.QtySats = 100000000
	ev.Exchange = "BITGET"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate event processing
		ev.Seq = uint64(i + 1)
		seq.nextSeq = uint64(i + 1) // Align sequence to avoid gap panic

		// Direct call to handleMarketUpdate (Hotpath core)
		seq.handleMarketUpdate(ev)
	}

	event.ReleaseMarketUpdateEvent(ev)
}

// BenchmarkSequencer_FullPipeline measures end-to-end event processing.
// Note: This benchmark includes channel overhead.
func BenchmarkSequencer_FullPipeline(b *testing.B) {
	seq := NewSequencer(b.N+100, nil, nil, nil)
	inbox := seq.Inbox()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start sequencer in background
	go seq.Run(ctx)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ev := event.AcquireMarketUpdateEvent()
		ev.Seq = uint64(i + 1)
		ev.Ts = quant.TimeStamp(int64(i))
		ev.Symbol = "BTCUSDT"
		ev.PriceMicros = 50000000000
		ev.QtySats = 100000000
		ev.Exchange = "BITGET"

		inbox <- ev
	}

	cancel()
}
