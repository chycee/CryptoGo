package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"crypto_go/internal/app"
	"crypto_go/internal/domain"
	"crypto_go/internal/engine"
	"crypto_go/internal/infra"
	"crypto_go/internal/infra/bitget"
	"crypto_go/internal/infra/upbit"
	"crypto_go/internal/strategy"

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

	// 2.1 Display Safety UX (Banner)
	infra.PrintBanner(bootstrap.Config)

	// 3. Graceful Shutdown Context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. Background Asset Sync (Simulating Loading Screen logic)
	go bootstrap.SyncAssets(ctx)

	// 5. Initialize Strategy & Sequencer
	evStore := bootstrap.EventStore

	// Example Strategy: SMA Cross (3, 5) for BTC-USDT
	strat := strategy.NewSMACrossStrategy("BTC-USDT", 3, 5)

	seq := engine.NewSequencer(1024, evStore, strat, func(state *domain.MarketState) {
		// slog.Info("State changed", slog.String("symbol", state.Symbol), slog.String("price", state.PriceMicros.String()))
	})

	// Start Sequencer in its own goroutine (The Hotpath Loop)
	go seq.Run(ctx)
	slog.InfoContext(ctx, "✅ Sequencer (Hotpath) started")

	cfg := bootstrap.Config
	nextSeq := uint64(1)

	// Exchange Rate Client (Gateway) - Still in infra root for common utility
	exchangeRateClient := infra.NewExchangeRateClient(seq.Inbox(), &nextSeq)
	if err := exchangeRateClient.Start(ctx); err != nil {
		slog.Error("Failed to start exchange rate client", slog.Any("error", err))
	}
	defer exchangeRateClient.Stop()

	// 6. Upbit/Bitget Workers (Modular Gateways)
	if len(cfg.API.Upbit.Symbols) > 0 {
		upbitWorker := upbit.NewWorker(cfg.API.Upbit.Symbols, seq.Inbox(), &nextSeq)
		if err := upbitWorker.Connect(ctx); err != nil {
			slog.Error("Failed to connect Upbit", slog.Any("error", err))
		}
		defer upbitWorker.Disconnect()
		slog.InfoContext(ctx, "✅ UpbitWorker started", slog.Int("symbols", len(cfg.API.Upbit.Symbols)))
	}

	if len(cfg.API.Bitget.Symbols) > 0 {
		// Spot
		bitgetSpotWorker := bitget.NewSpotWorker(cfg.API.Bitget.Symbols, seq.Inbox(), &nextSeq)
		if err := bitgetSpotWorker.Connect(ctx); err != nil {
			slog.Error("Failed to connect Bitget Spot", slog.Any("error", err))
		}
		defer bitgetSpotWorker.Disconnect()
		slog.InfoContext(ctx, "✅ BitgetSpotWorker started")

		// Futures
		bitgetFuturesWorker := bitget.NewFuturesWorker(cfg.API.Bitget.Symbols, seq.Inbox(), &nextSeq)
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
