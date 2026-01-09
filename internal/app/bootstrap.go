package app

import (
	"context"
	"log/slog"

	"crypto_go/internal/domain"
	"crypto_go/internal/event"
	"crypto_go/internal/infra"
	"crypto_go/internal/storage"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Bootstrap orchestrates the application startup sequence
type Bootstrap struct {
	Config     *infra.Config
	EventStore *storage.EventStore
	Downloader *infra.IconDownloader
}

// NewBootstrap creates a new Bootstrap instance
func NewBootstrap() *Bootstrap {
	return &Bootstrap{}
}

// Initialize performs core system initialization (DB, Dir, etc.)
func (b *Bootstrap) Initialize() error {
	slog.Info("🚀 Bootstrapping Crypto Go...")

	// 0. Runtime Warmup (GC Optimization)
	event.Warmup()
	slog.Info("🔥 Event Pool Warmed up")

	// 1. Load Config (Dynamic Path Resolution)
	cfg, err := infra.LoadConfig(infra.ResolveConfigPath())
	if err != nil {
		return err // Let main handle the error
	}
	b.Config = cfg

	// 2. Setup Logger
	logger := infra.NewLogger(cfg)
	slog.SetDefault(logger)

	// 3. Initialize EventStore (Single-Writer WAL DB)
	// STES: Data Isolation - _workspace/data/{mode}/events.db
	mode := strings.ToLower(cfg.Trading.Mode)
	if mode == "" {
		mode = "paper" // Default to paper if not set
	}

	workDir := infra.GetWorkspaceDir()
	dataDir := filepath.Join(workDir, "data", mode)
	logDir := filepath.Join(workDir, "logs", mode)

	// Ensure directories exist (0755)
	if err := infra.EnsureDir(dataDir); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}
	if err := infra.EnsureDir(logDir); err != nil {
		return fmt.Errorf("failed to create log dir: %w", err)
	}

	// 3.1 Singleton Instance Lock (OS Security)
	// Prevent DB corruption on Desktop environments by blocking multi-process access to same data.
	unlock, err := infra.CreateLockFile(workDir)
	if err != nil {
		return err
	}
	// Note: In a real app, you might want to store 'unlock' in the Bootstrap struct to call on Exit.
	// For now, we rely on os.Exit cleaning up or manual cleanup if crash occurs.
	_ = unlock

	dbPath := filepath.Join(dataDir, "events.db")
	evStore, err := storage.NewEventStore(dbPath)
	if err != nil {
		return err
	}
	b.EventStore = evStore
	slog.Info("✅ EventStore initialized (WAL-mode)", "path", dbPath, "mode", mode)

	// 4. Initialize Icon Downloader
	downloader, err := infra.NewIconDownloader()
	if err != nil {
		return err
	}
	b.Downloader = downloader
	slog.Info("✅ Icon downloader ready")

	return nil
}

// SyncAssets synchronizes symbols and icons in the background
func (b *Bootstrap) SyncAssets(ctx context.Context) {
	slog.Info("🔄 Starting asset synchronization...")

	uniqueSymbols := make(map[string]bool)
	for _, s := range b.Config.API.Upbit.Symbols {
		uniqueSymbols[s] = true
	}
	for s := range b.Config.API.Bitget.Symbols {
		uniqueSymbols[s] = true
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)

	for symbol := range uniqueSymbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case semaphore <- struct{}{}:
			}
			defer func() { <-semaphore }()

			nowUnixM := time.Now().UnixMicro()
			coin := &domain.CoinInfo{
				Symbol:         sym,
				Name:           sym,
				IsActive:       true,
				UpdatedAtUnixM: nowUnixM,
			}

			// Try to load existing
			key := "coin:" + sym
			if val, _ := b.EventStore.GetMetadata(ctx, key); val != "" {
				var existing domain.CoinInfo
				if err := json.Unmarshal([]byte(val), &existing); err == nil {
					coin.IsFavorite = existing.IsFavorite
					coin.IconPath = existing.IconPath
					coin.LastSyncedUnixM = existing.LastSyncedUnixM
				}
			}

			// Download Icon if needed
			if path, err := b.Downloader.DownloadIcon(sym); err == nil && path != "" {
				coin.IconPath = path
				coin.LastSyncedUnixM = nowUnixM
			}

			// Save back to metadata
			data, _ := json.Marshal(coin)
			b.EventStore.UpsertMetadata(ctx, key, string(data), nowUnixM)
		}(symbol)
	}

	wg.Wait()
	slog.Info("✨ Asset synchronization completed")
}
