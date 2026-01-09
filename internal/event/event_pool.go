package event

import (
	"sync"
)

// EventPool provides sync.Pool for high-frequency event allocation.
// Use this to reduce GC pressure in the hotpath.
//
// Usage:
//
//	ev := AcquireMarketUpdateEvent()
//	ev.Symbol = "BTC"
//	// ... use event ...
//	ReleaseMarketUpdateEvent(ev)  // Return to pool after processing
var marketUpdatePool = sync.Pool{
	New: func() interface{} {
		return &MarketUpdateEvent{}
	},
}

// AcquireMarketUpdateEvent gets a MarketUpdateEvent from the pool.
// The returned event has zero values and must be initialized.
func AcquireMarketUpdateEvent() *MarketUpdateEvent {
	return marketUpdatePool.Get().(*MarketUpdateEvent)
}

// ReleaseMarketUpdateEvent returns a MarketUpdateEvent to the pool.
// The event is reset to zero values before being pooled.
func ReleaseMarketUpdateEvent(ev *MarketUpdateEvent) {
	if ev == nil {
		return
	}
	// Reset all fields to zero values
	ev.Seq = 0
	ev.Ts = 0
	ev.Symbol = ""
	ev.PriceMicros = 0
	ev.QtySats = 0
	ev.Exchange = ""

	marketUpdatePool.Put(ev)
}

// OrderUpdateEvent pool
var orderUpdatePool = sync.Pool{
	New: func() interface{} {
		return &OrderUpdateEvent{}
	},
}

// AcquireOrderUpdateEvent gets an OrderUpdateEvent from the pool.
func AcquireOrderUpdateEvent() *OrderUpdateEvent {
	return orderUpdatePool.Get().(*OrderUpdateEvent)
}

// ReleaseOrderUpdateEvent returns an OrderUpdateEvent to the pool.
func ReleaseOrderUpdateEvent(ev *OrderUpdateEvent) {
	if ev == nil {
		return
	}
	ev.Seq = 0
	ev.Ts = 0
	ev.OrderID = ""
	ev.Status = ""
	ev.PriceMicros = 0
	ev.AccumulatedQtySats = 0

	orderUpdatePool.Put(ev)
}

// Warmup pre-allocates event objects to reduce GC pressure at startup.
// It acquires and releases a batch of events.
func Warmup() {
	const batchSize = 1000
	
	// Warmup MarketUpdate Events
	marketEvs := make([]*MarketUpdateEvent, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		marketEvs = append(marketEvs, AcquireMarketUpdateEvent())
	}
	for _, ev := range marketEvs {
		ReleaseMarketUpdateEvent(ev)
	}

	// Warmup OrderUpdate Events
	orderEvs := make([]*OrderUpdateEvent, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		orderEvs = append(orderEvs, AcquireOrderUpdateEvent())
	}
	for _, ev := range orderEvs {
		ReleaseOrderUpdateEvent(ev)
	}
}
