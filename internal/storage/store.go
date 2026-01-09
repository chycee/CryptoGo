package storage

import (
	"context"
	"crypto_go/internal/event"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/glebarez/go-sqlite"
)

// EventStore handles persistent storage of events in SQLite.
type EventStore struct {
	db *sql.DB
}

// NewEventStore creates a new SQLite event store with WAL mode enabled.
func NewEventStore(dbPath string) (*EventStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// Configure SQLite for high-performance deterministic logging
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA cache_size=-2000;", // 2MB cache
		"PRAGMA foreign_keys=ON;",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("failed to set pragma %s: %w", pragma, err)
		}
	}

	// Create metadata table for KV storage (GORM replacement)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at INTEGER NOT NULL
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata table: %w", err)
	}

	// Create events table for WAL-first event logging
	// version column is for future Optimistic Lock support (multi-writer)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY,
			type INTEGER NOT NULL,
			ts INTEGER NOT NULL,
			payload BLOB NOT NULL,
			version INTEGER NOT NULL DEFAULT 1
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create events table: %w", err)
	}

	return &EventStore{db: db}, nil
}

// SaveEvent stores an event in the database.
func (s *EventStore) SaveEvent(ctx context.Context, ev event.Event) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		"INSERT INTO events (id, type, ts, payload) VALUES (?, ?, ?, ?)",
		ev.GetSeq(), ev.GetType(), ev.GetTs(), payload,
	)
	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	return nil
}

// UpsertMetadata saves a key-value pair to the metadata table.
func (s *EventStore) UpsertMetadata(ctx context.Context, key, value string, ts int64) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO metadata (key, value, updated_at) VALUES (?, ?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at",
		key, value, ts,
	)
	return err
}

// GetMetadata retrieves a value from the metadata table.
func (s *EventStore) GetMetadata(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM metadata WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// GetLastSeq returns the highest event sequence number stored in WAL.
// Returns 0 if no events exist.
func (s *EventStore) GetLastSeq(ctx context.Context) (uint64, error) {
	var lastSeq sql.NullInt64
	err := s.db.QueryRowContext(ctx, "SELECT MAX(id) FROM events").Scan(&lastSeq)
	if err != nil {
		return 0, fmt.Errorf("failed to get last seq: %w", err)
	}
	if !lastSeq.Valid {
		return 0, nil // No events yet
	}
	return uint64(lastSeq.Int64), nil
}

// LoadEvents loads all events from WAL starting from fromSeq (inclusive).
// Used for Replay Engine to reconstruct state.
func (s *EventStore) LoadEvents(ctx context.Context, fromSeq uint64) ([]*event.MarketUpdateEvent, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, type, ts, payload FROM events WHERE id >= ? ORDER BY id ASC",
		fromSeq,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*event.MarketUpdateEvent
	for rows.Next() {
		var id int64
		var evType int
		var ts int64
		var payload []byte

		if err := rows.Scan(&id, &evType, &ts, &payload); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		// Currently only supporting MarketUpdateEvent for replay
		if event.Type(evType) == event.EvMarketUpdate {
			var ev event.MarketUpdateEvent
			if err := json.Unmarshal(payload, &ev); err != nil {
				return nil, fmt.Errorf("failed to unmarshal event %d: %w", id, err)
			}
			events = append(events, &ev)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return events, nil
}

// Close closes the database connection.
func (s *EventStore) Close() error {
	return s.db.Close()
}
