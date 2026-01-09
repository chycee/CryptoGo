package event

import (
	"testing"
)

func TestEventPool(t *testing.T) {
	// Acquire and use
	ev := AcquireMarketUpdateEvent()
	ev.Symbol = "BTC"
	ev.PriceMicros = 50000000000

	if ev.Symbol != "BTC" {
		t.Error("Symbol not set")
	}

	// Release
	ReleaseMarketUpdateEvent(ev)

	// Acquire again - should be reset
	ev2 := AcquireMarketUpdateEvent()
	if ev2.Symbol != "" {
		t.Error("Event should be reset after release")
	}
	ReleaseMarketUpdateEvent(ev2)
}

// BenchmarkWithoutPool measures allocation without pool
func BenchmarkWithoutPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ev := &MarketUpdateEvent{
			Symbol:      "BTC",
			PriceMicros: 50000000000,
		}
		_ = ev
	}
}

// BenchmarkWithPool measures allocation with pool
func BenchmarkWithPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ev := AcquireMarketUpdateEvent()
		ev.Symbol = "BTC"
		ev.PriceMicros = 50000000000
		ReleaseMarketUpdateEvent(ev)
	}
}
