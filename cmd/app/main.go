package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"crypto_go/internal/app"
	"crypto_go/internal/engine"
	"crypto_go/internal/infra"

	_ "net/http/pprof" // For pprof profiling
)

func main() {
	// 1. Pprof Server (for performance profiling)
	go func() {
		// Localhost only for security
		slog.Info("🕵️ Pprof server started on localhost:6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			slog.Error("Pprof server failed", slog.Any("error", err))
		}
	}()

	// 2. System Bootstrapping
	bootstrap := app.NewBootstrap()
	if err := bootstrap.Initialize(); err != nil {
		slog.Error("❌ Bootstrapping failed", slog.Any("error", err))
		os.Exit(1)
	}

	// 3. Graceful Shutdown Context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. Background Asset Sync (Simulating Loading Screen logic)
	go bootstrap.SyncAssets(ctx)

	// 5. Initialize Sequencer & Store
	evStore := bootstrap.EventStore
	seq := engine.NewSequencer(1024, evStore, func(state *engine.MarketState) {
		// slog.Info("State changed", slog.String("symbol", state.Symbol), slog.String("price", state.Price.String()))
	})

	// Start Sequencer in its own goroutine (The Hotpath Loop)
	go seq.Run(ctx)
	slog.InfoContext(ctx, "✅ Sequencer (Hotpath) started")

	cfg := bootstrap.Config
	nextSeq := uint64(1)

	// Exchange Rate Client (Gateway)
	exchangeRateClient := infra.NewExchangeRateClient(seq.Inbox(), &nextSeq)
	if err := exchangeRateClient.Start(ctx); err != nil {
		slog.Error("Failed to start exchange rate client", slog.Any("error", err))
	}
	defer exchangeRateClient.Stop()

	// 6. Upbit/Bitget Workers (Gateways)
	if len(cfg.API.Upbit.Symbols) > 0 {
		upbitWorker := infra.NewUpbitWorker(cfg.API.Upbit.Symbols, seq.Inbox(), &nextSeq)
		if err := upbitWorker.Connect(ctx); err != nil {
			slog.Error("Failed to connect Upbit", slog.Any("error", err))
		}
		defer upbitWorker.Disconnect()
		slog.InfoContext(ctx, "✅ UpbitWorker started", slog.Int("symbols", len(cfg.API.Upbit.Symbols)))
	}

	if len(cfg.API.Bitget.Symbols) > 0 {
		// Spot
		bitgetSpotWorker := infra.NewBitgetSpotWorker(cfg.API.Bitget.Symbols, seq.Inbox(), &nextSeq)
		if err := bitgetSpotWorker.Connect(ctx); err != nil {
			slog.Error("Failed to connect Bitget Spot", slog.Any("error", err))
		}
		defer bitgetSpotWorker.Disconnect()
		slog.InfoContext(ctx, "✅ BitgetSpotWorker started")

		// Futures
		bitgetFuturesWorker := infra.NewBitgetFuturesWorker(cfg.API.Bitget.Symbols, seq.Inbox(), &nextSeq)
		if err := bitgetFuturesWorker.Connect(ctx); err != nil {
			slog.Error("Failed to connect Bitget Futures", slog.Any("error", err))
		}
		defer bitgetFuturesWorker.Disconnect()
		slog.InfoContext(ctx, "✅ BitgetFuturesWorker started")
	}

	slog.InfoContext(ctx, "✨ Indie Quant System fully operational. Press Ctrl+C to exit.")

	// Wait for shutdown signal
	<-ctx.Done()

	slog.InfoContext(ctx, "👋 Shutting down gracefully...")
}
