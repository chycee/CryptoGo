package backtest

import (
	"context"
	"crypto_go/internal/engine"
	"crypto_go/internal/event"
	"crypto_go/internal/storage"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
)

// Replayer reads event logs from SQLite and feeds them into the Sequencer.
type Replayer struct {
	store *storage.EventStore
	db    *sql.DB // Direct DB access for reading logs
}

// NewReplayer creates a new replayer instance.
func NewReplayer(dbPath string) (*Replayer, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	store, err := storage.NewEventStore(dbPath)
	if err != nil {
		return nil, err
	}

	return &Replayer{
		store: store,
		db:    db,
	}, nil
}

// RunReplay replays all events into the provided sequencer.
func (r *Replayer) RunReplay(ctx context.Context, seq *engine.Sequencer) error {
	rows, err := r.db.QueryContext(ctx, "SELECT id, type, payload FROM events ORDER BY id ASC")
	if err != nil {
		return fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uint64
		var typ event.Type
		var payload []byte

		if err := rows.Scan(&id, &typ, &payload); err != nil {
			return err
		}

		var ev event.Event
		switch typ {
		case event.EvMarketUpdate:
			var m event.MarketUpdateEvent
			if err := json.Unmarshal(payload, &m); err != nil {
				return err
			}
			ev = &m
		case event.EvOrderUpdate:
			var o event.OrderUpdateEvent
			if err := json.Unmarshal(payload, &o); err != nil {
				return err
			}
			ev = &o
		default:
			slog.Warn("Unknown event type in log", slog.Any("type", typ))
			continue
		}

		// Feed into sequencer synchronously for deterministic replay.
		seq.ReplayEvent(ev)
	}

	return nil
}
