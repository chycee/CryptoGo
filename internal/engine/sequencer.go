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

	strategy strategy.Strategy

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
			// In Indie Quant, we halt after dump.
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

func (s *Sequencer) processEvent(ev event.Event) {
	// 1. Sequence Gap Check (Halt Policy)
	// 1. Sequence Gap Check (with Tolerance Policy)
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
		// Pending: trading logic not yet implemented
	default:
		slog.Warn("Unknown event type", slog.Any("type", ev.GetType()))
	}

	// 4. Increment Sequence
	s.nextSeq++
}

// ReplayEvent processes an event synchronously without WAL logging.
// This is used exclusively by the Replayer.
func (s *Sequencer) ReplayEvent(ev event.Event) {
	// Replay must still respect sequence order
	if ev.GetSeq() != s.nextSeq {
		panic(fmt.Sprintf("REPLAY_GAP_DETECTED: expected %d, got %d", s.nextSeq, ev.GetSeq()))
	}

	// Dispatch without WAL
	switch e := ev.(type) {
	case *event.MarketUpdateEvent:
		s.handleMarketUpdate(e)
	case *event.OrderUpdateEvent:
		// Pending: trading logic not yet implemented
	default:
		slog.Warn("Unknown event type in replay", slog.Any("type", ev.GetType()))
	}

	s.nextSeq++
}

func (s *Sequencer) handleMarketUpdate(e *event.MarketUpdateEvent) {
	state, ok := s.markets[e.Symbol]
	if !ok {
		state = &domain.MarketState{Symbol: e.Symbol}
		s.markets[e.Symbol] = state
	}

	// Update state (No Mutex needed here as we are in the Hotpath)
	state.PriceMicros = e.PriceMicros
	state.TotalQtySats = e.QtySats
	state.LastUpdateUnixM = e.Ts

	// Invoke Strategy
	if s.strategy != nil {
		actions := s.strategy.OnMarketUpdate(*state)
		for _, action := range actions {
			slog.Info("STRATEGY_ACTION", slog.Any("action", action))
			// TODO: Convert Action to OrderRequestEvent and process effectively
		}
	}

	if s.onStateUpdate != nil {
		s.onStateUpdate(state)
	}
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

	data := struct {
		NextSeq uint64                         `json:"next_seq"`
		Markets map[string]*domain.MarketState `json:"markets"`
	}{
		NextSeq: s.nextSeq,
		Markets: s.markets,
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
