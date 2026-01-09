package infra

import (
	"sync/atomic"
	"time"
)

// Metrics provides lightweight observability without external dependencies.
// Uses atomic operations for thread-safety.
type Metrics struct {
	// Counters
	eventsProcessed atomic.Uint64
	ordersFilled    atomic.Uint64
	errorsTotal     atomic.Uint64

	// Latency tracking
	latencySumNs atomic.Int64
	latencyCount atomic.Uint64

	// Gauges
	activeConnections atomic.Int32
	circuitOpen       atomic.Int32 // 1 = open, 0 = closed
}

// GlobalMetrics is the singleton metrics instance.
var GlobalMetrics = &Metrics{}

// RecordEvent records an event processing with latency.
func (m *Metrics) RecordEvent(latencyNs int64) {
	m.eventsProcessed.Add(1)
	m.latencySumNs.Add(latencyNs)
	m.latencyCount.Add(1)
}

// RecordError records an error occurrence.
func (m *Metrics) RecordError() {
	m.errorsTotal.Add(1)
}

// RecordOrderFilled records a filled order.
func (m *Metrics) RecordOrderFilled() {
	m.ordersFilled.Add(1)
}

// SetActiveConnections sets the current active connection count.
func (m *Metrics) SetActiveConnections(count int32) {
	m.activeConnections.Store(count)
}

// IncrementConnections increments active connections by 1.
func (m *Metrics) IncrementConnections() {
	m.activeConnections.Add(1)
}

// DecrementConnections decrements active connections by 1.
func (m *Metrics) DecrementConnections() {
	m.activeConnections.Add(-1)
}

// SetCircuitState sets the circuit breaker state (true = open).
func (m *Metrics) SetCircuitState(open bool) {
	if open {
		m.circuitOpen.Store(1)
	} else {
		m.circuitOpen.Store(0)
	}
}

// MetricsSnapshot is a point-in-time view of all metrics.
type MetricsSnapshot struct {
	EventsProcessed   uint64
	OrdersFilled      uint64
	ErrorsTotal       uint64
	AvgLatencyNs      int64
	ActiveConnections int32
	CircuitOpen       bool
	Timestamp         time.Time
}

// Snapshot returns current metrics as a snapshot.
func (m *Metrics) Snapshot() MetricsSnapshot {
	var avgLatency int64
	count := m.latencyCount.Load()
	if count > 0 {
		avgLatency = m.latencySumNs.Load() / int64(count)
	}

	return MetricsSnapshot{
		EventsProcessed:   m.eventsProcessed.Load(),
		OrdersFilled:      m.ordersFilled.Load(),
		ErrorsTotal:       m.errorsTotal.Load(),
		AvgLatencyNs:      avgLatency,
		ActiveConnections: m.activeConnections.Load(),
		CircuitOpen:       m.circuitOpen.Load() == 1,
		Timestamp:         time.Now(),
	}
}

// Reset clears all metrics (for testing).
func (m *Metrics) Reset() {
	m.eventsProcessed.Store(0)
	m.ordersFilled.Store(0)
	m.errorsTotal.Store(0)
	m.latencySumNs.Store(0)
	m.latencyCount.Store(0)
	m.activeConnections.Store(0)
	m.circuitOpen.Store(0)
}
