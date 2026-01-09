package infra

import (
	"testing"
)

func TestMetrics_RecordEvent(t *testing.T) {
	m := &Metrics{}

	m.RecordEvent(1000)
	m.RecordEvent(2000)
	m.RecordEvent(3000)

	snap := m.Snapshot()

	if snap.EventsProcessed != 3 {
		t.Errorf("Expected 3 events, got %d", snap.EventsProcessed)
	}

	// Average latency: (1000 + 2000 + 3000) / 3 = 2000
	if snap.AvgLatencyNs != 2000 {
		t.Errorf("Expected avg latency 2000, got %d", snap.AvgLatencyNs)
	}
}

func TestMetrics_Connections(t *testing.T) {
	m := &Metrics{}

	m.IncrementConnections()
	m.IncrementConnections()
	m.IncrementConnections()

	snap := m.Snapshot()
	if snap.ActiveConnections != 3 {
		t.Errorf("Expected 3 connections, got %d", snap.ActiveConnections)
	}

	m.DecrementConnections()
	snap = m.Snapshot()
	if snap.ActiveConnections != 2 {
		t.Errorf("Expected 2 connections, got %d", snap.ActiveConnections)
	}
}

func TestMetrics_CircuitState(t *testing.T) {
	m := &Metrics{}

	snap := m.Snapshot()
	if snap.CircuitOpen {
		t.Error("Expected circuit closed initially")
	}

	m.SetCircuitState(true)
	snap = m.Snapshot()
	if !snap.CircuitOpen {
		t.Error("Expected circuit open")
	}

	m.SetCircuitState(false)
	snap = m.Snapshot()
	if snap.CircuitOpen {
		t.Error("Expected circuit closed")
	}
}

func TestMetrics_Reset(t *testing.T) {
	m := &Metrics{}

	m.RecordEvent(1000)
	m.RecordError()
	m.IncrementConnections()

	m.Reset()
	snap := m.Snapshot()

	if snap.EventsProcessed != 0 {
		t.Error("Expected 0 events after reset")
	}
	if snap.ErrorsTotal != 0 {
		t.Error("Expected 0 errors after reset")
	}
	if snap.ActiveConnections != 0 {
		t.Error("Expected 0 connections after reset")
	}
}
