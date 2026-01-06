package engine

import (
	"context"
	"crypto_go/internal/event"
	"crypto_go/internal/storage"
	"crypto_go/pkg/quant"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

// MarketState holds the current state of a single market.
// Fields are ordered for cache-line efficiency: hot fields (price/qty) first.
type MarketState struct {
	// Hot fields (frequently accessed together in the hotpath)
	PriceMicros     quant.PriceMicros `json:"price"`
	TotalQtySats    quant.QtySats     `json:"qty"`
	LastUpdateUnixM quant.TimeStamp   `json:"last_update"`
	// Cold fields (less frequent access)
	Symbol string `json:"symbol"`
}

// Sequencer is the core single-threaded event processor.
type Sequencer struct {
	inbox   chan event.Event
	markets map[string]*MarketState
	nextSeq uint64
	store   *storage.EventStore

	// Boundary: used to notify UI or other systems of state changes
	onStateUpdate func(*MarketState)

	mu sync.RWMutex // Used only for external reads (e.g. UI)
}

// NewSequencer creates a new sequencer instance.
func NewSequencer(inboxSize int, store *storage.EventStore, onUpdate func(*MarketState)) *Sequencer {
	return &Sequencer{
		inbox:         make(chan event.Event, inboxSize),
		markets:       make(map[string]*MarketState),
		nextSeq:       1,
		store:         store,
		onStateUpdate: onUpdate,
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
	if ev.GetSeq() != s.nextSeq {
		panic(fmt.Sprintf("SEQUENCE_GAP_DETECTED: expected %d, got %d", s.nextSeq, ev.GetSeq()))
	}

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
		state = &MarketState{Symbol: e.Symbol}
		s.markets[e.Symbol] = state
	}

	// Update state (No Mutex needed here as we are in the Hotpath)
	state.PriceMicros = e.PriceMicros
	state.TotalQtySats = e.QtySats
	state.LastUpdateUnixM = e.Ts

	if s.onStateUpdate != nil {
		s.onStateUpdate(state)
	}
}

// GetMarketState returns a snapshot of the market state (external read).
func (s *Sequencer) GetMarketState(symbol string) (MarketState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.markets[symbol]
	if !ok {
		return MarketState{}, false
	}
	return *state, true // Return copy
}

// DumpState writes the entire internal state to a file (for post-mortem).
func (s *Sequencer) DumpState(filename string) {
	slog.Info("Dumping internal state...", slog.String("file", filename))

	data := struct {
		NextSeq uint64                  `json:"next_seq"`
		Markets map[string]*MarketState `json:"markets"`
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
