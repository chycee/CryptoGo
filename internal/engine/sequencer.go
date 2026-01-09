package engine

import (
	"context"
	"crypto_go/internal/domain"
	"crypto_go/internal/event"
	"crypto_go/internal/storage"
	"crypto_go/internal/strategy"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

// Sequencer is the core single-threaded event processor.
type Sequencer struct {
	inbox   chan event.Event
	markets map[string]*domain.MarketState
	nextSeq uint64
	store   *storage.EventStore

	strategy    strategy.Strategy
	orderBuf    [16]domain.Order    // Pre-allocated buffer for strategy results (Rule #3: Zero-Alloc)
	balanceBook *domain.BalanceBook // Rule #8: Balance invariant enforcement

	// Boundary: used to notify UI or other systems of state changes
	onStateUpdate func(*domain.MarketState)

	mu sync.RWMutex // Used only for external reads (e.g. UI)
}

// NewSequencer creates a new sequencer instance.
func NewSequencer(inboxSize int, store *storage.EventStore, strat strategy.Strategy, onUpdate func(*domain.MarketState)) *Sequencer {
	seq := &Sequencer{
		inbox:         make(chan event.Event, inboxSize),
		markets:       make(map[string]*domain.MarketState),
		nextSeq:       1,
		store:         store,
		strategy:      strat,
		onStateUpdate: onUpdate,
		balanceBook:   domain.NewBalanceBook(), // Rule #8: Invariant enforcement
	}
	return seq
}

// RecoverFromWAL restores state by replaying all events from WAL.
// This is the core of "Backtest is Reality" - same code path for live and replay.
func (s *Sequencer) RecoverFromWAL(ctx context.Context) error {
	if s.store == nil {
		slog.Info("No store configured, starting fresh")
		return nil
	}

	// Get last sequence number from WAL
	lastSeq, err := s.store.GetLastSeq(ctx)
	if err != nil {
		return fmt.Errorf("failed to get last seq: %w", err)
	}

	if lastSeq == 0 {
		slog.Info("WAL is empty, starting fresh")
		return nil
	}

	// Load all events from WAL
	events, err := s.store.LoadEvents(ctx, 1)
	if err != nil {
		return fmt.Errorf("failed to load events: %w", err)
	}

	slog.Info("Replaying events from WAL", slog.Int("count", len(events)))

	// Replay each event using the same code path as live
	for _, ev := range events {
		s.ReplayEvent(ev)
	}

	// Rule #8: Verify balance invariants after replay
	s.balanceBook.VerifyAll()

	slog.Info("State recovered from WAL", slog.Uint64("next_seq", s.nextSeq))
	return nil
}

// ValidateSequence checks for gaps based on strictness policy.
func (s *Sequencer) ValidateSequence(evSeq uint64) {
	expected := s.nextSeq
	if evSeq == expected {
		return
	}

	diff := int64(evSeq) - int64(expected)

	// Case 1: Replay/Duplicate (Old event)
	if diff < 0 {
		slog.Warn("SEQUENCE_DUPLICATE_IGNORED", slog.Uint64("expected", expected), slog.Uint64("got", evSeq))
		return
	}

	// Case 2: Future Gap
	if diff > 0 {
		// User Request: Allow small gaps <= 10 for Availability
		if diff <= 10 {
			slog.Warn("SEQUENCE_GAP_TOLERATED",
				slog.Uint64("expected", expected),
				slog.Uint64("got", evSeq),
				slog.Int64("gap", diff))

			// Fast-forward sequence to match event
			// TODO: In Execution Phase, this MUST accept a state-resync triggers
			s.nextSeq = evSeq
			return
		}

		// Hard Panic for large gaps
		panic(fmt.Sprintf("SEQUENCE_GAP_FATAL: expected %d, got %d", expected, evSeq))
	}
}

// Inbox returns the event channel. External workers send events here.
func (s *Sequencer) Inbox() chan<- event.Event {
	return s.inbox
}

// Run starts the main event loop. This MUST be run in a single goroutine.
func (s *Sequencer) Run(ctx context.Context) {
	slog.Info("Sequencer started (Single-Thread Hotpath)")

	defer func() {
		if r := recover(); r != nil {
			slog.Error("CRITICAL_PANIC_DETECTED", slog.Any("panic", r))
			s.DumpState("panic_dump.json")
			// In Quant, we halt after dump.
			panic(fmt.Sprintf("HALTED: %v", r))
		}
	}()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Sequencer stopping...")
			return
		case ev := <-s.inbox:
			s.processEvent(ev)
		}
	}
}

// ReplayEvent processes an event synchronously without WAL logging.
// This is used exclusively by the Replayer.
func (s *Sequencer) ReplayEvent(ev event.Event) {
	if ev.GetSeq() != s.nextSeq {
		panic(fmt.Sprintf("REPLAY_GAP_DETECTED: expected %d, got %d", s.nextSeq, ev.GetSeq()))
	}

	switch e := ev.(type) {
	case *event.MarketUpdateEvent:
		s.handleMarketUpdate(e)
	case *event.OrderUpdateEvent:
		// Pending
	}

	s.nextSeq++
}

func (s *Sequencer) processEvent(ev event.Event) {
	// 1. Sequence Gap Check (Halt Policy)
	s.ValidateSequence(ev.GetSeq())

	// 2. WAL-first: Persistence
	if s.store != nil {
		if err := s.store.SaveEvent(context.Background(), ev); err != nil {
			panic(fmt.Sprintf("PERSISTENCE_FAILURE: %v", err))
		}
	}

	// 3. Logic Dispatch
	switch e := ev.(type) {
	case *event.MarketUpdateEvent:
		s.handleMarketUpdate(e)
	case *event.OrderUpdateEvent:
		// Pending
	}

	// 4. Increment Sequence
	s.nextSeq++
}

func (s *Sequencer) handleMarketUpdate(e *event.MarketUpdateEvent) {
	state, ok := s.markets[e.Symbol]
	if !ok {
		// Cold path: New symbol allocation
		state = &domain.MarketState{Symbol: e.Symbol}
		s.markets[e.Symbol] = state
	}

	// Hot path: No mutex (Single-threaded owner)
	state.PriceMicros = e.PriceMicros
	state.TotalQtySats = e.QtySats
	state.LastUpdateUnixM = e.Ts

	// Trace logging should be disabled or sampled in production (Rule #6: Lean Metrics)
	// slog.Debug("HOT_INGEST", "symbol", e.Symbol, "price", e.PriceMicros)

	// Invoke Strategy
	if s.strategy != nil {
		count := s.strategy.OnMarketUpdate(*state, s.orderBuf[:])
		for i := 0; i < count; i++ {
			s.handleStrategyAction(&s.orderBuf[i])
		}
	}

	if s.onStateUpdate != nil {
		// Rule #2: Pass copy to external callback, not pointer (state ownership protection)
		stateCopy := *state
		s.onStateUpdate(&stateCopy)
	}
}

func (s *Sequencer) handleStrategyAction(order *domain.Order) {
	// Root of Rule #1: Deterministic order generation
	// Rule #6: Hotpath logging removed. Use metrics or sampling if needed.

	// TODO: Create OrderRequestEvent and dispatch to execution gateway
}

// GetMarketState returns a snapshot of the market state (external read).
func (s *Sequencer) GetMarketState(symbol string) (domain.MarketState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.markets[symbol]
	if !ok {
		return domain.MarketState{}, false
	}
	return *state, true // Return copy
}

// DumpState writes the entire internal state to a file (for post-mortem).
func (s *Sequencer) DumpState(filename string) {
	slog.Info("Dumping internal state...", slog.String("file", filename))

	// Rule #8: Verify balance invariants before dump (may panic if corrupted)
	s.balanceBook.VerifyAll()

	data := struct {
		NextSeq  uint64                         `json:"next_seq"`
		Markets  map[string]*domain.MarketState `json:"markets"`
		Balances map[string]domain.Balance      `json:"balances"`
	}{
		NextSeq:  s.nextSeq,
		Markets:  s.markets,
		Balances: s.balanceBook.Snapshot(),
	}

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal state", slog.Any("error", err))
		return
	}

	err = os.WriteFile(filename, b, 0644)
	if err != nil {
		slog.Error("Failed to write state dump", slog.Any("error", err))
	}
}

// BalanceBook returns the balance book for external access (e.g., UI, testing).
func (s *Sequencer) BalanceBook() *domain.BalanceBook {
	return s.balanceBook
}

// GetNextSeq returns the next expected sequence number (for testing).
func (s *Sequencer) GetNextSeq() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextSeq
}

// GetMarketPrice returns the current price for an exchange+symbol (for testing).
func (s *Sequencer) GetMarketPrice(exchange, symbol string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := exchange + ":" + symbol
	if state, ok := s.markets[key]; ok {
		return int64(state.PriceMicros)
	}
	return 0
}

// ProcessEventForTest processes an event synchronously for testing.
// This bypasses the inbox channel for easier test control.
func (s *Sequencer) ProcessEventForTest(ev event.Event) {
	s.processEvent(ev)
}
