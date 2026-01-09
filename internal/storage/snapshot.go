package storage

import (
	"crypto_go/internal/domain"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// Snapshot represents a point-in-time capture of system state.
// Used for fast recovery instead of replaying entire WAL.
type Snapshot struct {
	Seq     uint64                         `json:"seq"`     // Last processed sequence number
	TsUnix  int64                          `json:"ts"`      // Snapshot creation timestamp (Unix seconds)
	Markets map[string]*domain.MarketState `json:"markets"` // Market state at snapshot time
}

// SnapshotManager handles saving and loading snapshots.
type SnapshotManager struct {
	dir string
}

// NewSnapshotManager creates a new snapshot manager.
// dir: directory to store snapshot files.
func NewSnapshotManager(dir string) *SnapshotManager {
	return &SnapshotManager{dir: dir}
}

// Save writes a snapshot to disk.
func (sm *SnapshotManager) Save(snap *Snapshot) error {
	// Ensure directory exists
	if err := os.MkdirAll(sm.dir, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot dir: %w", err)
	}

	// Generate filename with timestamp
	filename := fmt.Sprintf("snapshot_%d_%d.json", snap.Seq, snap.TsUnix)
	path := filepath.Join(sm.dir, filename)

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write snapshot: %w", err)
	}

	slog.Info("Snapshot saved",
		slog.Uint64("seq", snap.Seq),
		slog.String("path", path))

	return nil
}

// LoadLatest loads the most recent snapshot from disk.
// Returns nil if no snapshot exists.
func (sm *SnapshotManager) LoadLatest() (*Snapshot, error) {
	entries, err := os.ReadDir(sm.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No snapshots yet
		}
		return nil, fmt.Errorf("failed to read snapshot dir: %w", err)
	}

	var latestPath string
	var latestSeq uint64

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		var seq uint64
		var ts int64
		_, err := fmt.Sscanf(entry.Name(), "snapshot_%d_%d.json", &seq, &ts)
		if err != nil {
			continue // Not a snapshot file
		}

		if seq > latestSeq {
			latestSeq = seq
			latestPath = filepath.Join(sm.dir, entry.Name())
		}
	}

	if latestPath == "" {
		return nil, nil // No snapshots found
	}

	data, err := os.ReadFile(latestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot: %w", err)
	}

	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	slog.Info("Snapshot loaded",
		slog.Uint64("seq", snap.Seq),
		slog.String("path", latestPath))

	return &snap, nil
}

// CreateSnapshot creates a snapshot from current state.
func CreateSnapshot(seq uint64, markets map[string]*domain.MarketState) *Snapshot {
	// Deep copy markets to avoid mutation
	marketsCopy := make(map[string]*domain.MarketState, len(markets))
	for k, v := range markets {
		stateCopy := *v
		marketsCopy[k] = &stateCopy
	}

	return &Snapshot{
		Seq:     seq,
		TsUnix:  time.Now().Unix(),
		Markets: marketsCopy,
	}
}

// Cleanup removes old snapshots, keeping only the latest N.
func (sm *SnapshotManager) Cleanup(keepCount int) error {
	entries, err := os.ReadDir(sm.dir)
	if err != nil {
		return err
	}

	// Collect snapshot files with their sequence numbers
	type snapFile struct {
		path string
		seq  uint64
	}
	var files []snapFile

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		var seq uint64
		var ts int64
		if _, err := fmt.Sscanf(entry.Name(), "snapshot_%d_%d.json", &seq, &ts); err == nil {
			files = append(files, snapFile{
				path: filepath.Join(sm.dir, entry.Name()),
				seq:  seq,
			})
		}
	}

	// Sort by sequence (descending) and remove old ones
	if len(files) <= keepCount {
		return nil
	}

	// Simple bubble sort (small N)
	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			if files[j].seq > files[i].seq {
				files[i], files[j] = files[j], files[i]
			}
		}
	}

	// Remove old snapshots
	for i := keepCount; i < len(files); i++ {
		if err := os.Remove(files[i].path); err != nil {
			slog.Warn("Failed to remove old snapshot", slog.String("path", files[i].path))
		} else {
			slog.Info("Removed old snapshot", slog.String("path", files[i].path))
		}
	}

	return nil
}
