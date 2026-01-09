package storage

import (
	"crypto_go/internal/domain"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshot_SaveAndLoad(t *testing.T) {
	// Create temp directory
	dir := filepath.Join(os.TempDir(), "snapshot_test")
	defer os.RemoveAll(dir)

	sm := NewSnapshotManager(dir)

	// Create test snapshot
	markets := map[string]*domain.MarketState{
		"BTCUSDT": {
			Symbol:       "BTCUSDT",
			PriceMicros:  50000000000,
			TotalQtySats: 100000000,
		},
	}
	snap := CreateSnapshot(100, markets)

	// Save
	if err := sm.Save(snap); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load
	loaded, err := sm.LoadLatest()
	if err != nil {
		t.Fatalf("LoadLatest failed: %v", err)
	}

	if loaded == nil {
		t.Fatal("Expected snapshot, got nil")
	}

	if loaded.Seq != 100 {
		t.Errorf("Expected seq 100, got %d", loaded.Seq)
	}

	if loaded.Markets["BTCUSDT"].PriceMicros != 50000000000 {
		t.Errorf("Market price mismatch")
	}
}

func TestSnapshot_LoadLatest_MultipleSnapshots(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "snapshot_test2")
	defer os.RemoveAll(dir)

	sm := NewSnapshotManager(dir)

	// Save multiple snapshots
	for _, seq := range []uint64{10, 50, 30} {
		snap := &Snapshot{
			Seq:     seq,
			TsUnix:  int64(seq),
			Markets: make(map[string]*domain.MarketState),
		}
		if err := sm.Save(snap); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// Should load seq=50 (highest)
	loaded, err := sm.LoadLatest()
	if err != nil {
		t.Fatalf("LoadLatest failed: %v", err)
	}

	if loaded.Seq != 50 {
		t.Errorf("Expected latest seq 50, got %d", loaded.Seq)
	}
}

func TestSnapshot_LoadLatest_NoSnapshots(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "snapshot_empty")
	defer os.RemoveAll(dir)

	sm := NewSnapshotManager(dir)

	loaded, err := sm.LoadLatest()
	if err != nil {
		t.Fatalf("LoadLatest failed: %v", err)
	}

	if loaded != nil {
		t.Errorf("Expected nil for empty dir, got %v", loaded)
	}
}

func TestSnapshot_Cleanup(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "snapshot_cleanup")
	defer os.RemoveAll(dir)

	sm := NewSnapshotManager(dir)

	// Create 5 snapshots
	for seq := uint64(1); seq <= 5; seq++ {
		snap := &Snapshot{Seq: seq, TsUnix: int64(seq), Markets: make(map[string]*domain.MarketState)}
		if err := sm.Save(snap); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// Cleanup, keep only 2
	if err := sm.Cleanup(2); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Count remaining files
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Errorf("Expected 2 snapshots after cleanup, got %d", len(entries))
	}

	// Should keep seq 4 and 5
	loaded, _ := sm.LoadLatest()
	if loaded.Seq != 5 {
		t.Errorf("Expected seq 5 to remain, got %d", loaded.Seq)
	}
}
