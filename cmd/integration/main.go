package main

import (
	"context"
	"crypto_go/internal/domain"
	"crypto_go/internal/execution"
	"crypto_go/internal/infra"
	"fmt"
	"log/slog"
	"os"
	"time"
)

func main() {
	// 0. Global Panic Recovery (Debug Exception Handling)
	defer infra.Recover()

	// 1. Setup Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("🚀 Starting Bitget Integration Test...")

	// 2. Load Secret Config First (for explicit DEMO keys)
	secretPath := "secrets/demo.yaml"
	slog.Info("🔑 Loading Secrets", "path", secretPath)

	secretCfg, err := infra.LoadSecretConfig(secretPath)
	if err != nil {
		slog.Error("❌ Failed to load secrets", "error", err)
		os.Exit(1)
	}
	// Note: Strings in SecretConfig will be GC'd.
	// Signer converts them to []byte and handles wiping.

	// 3. Manually Construct Config for DEMO Mode
	// Note: We bypass LoadConfig to force specific testing state.
	// We populate the nested anonymous structs directly.
	cfg := &infra.Config{}

	// Set Mode
	cfg.Trading.Mode = "DEMO"

	// Revert to Mainnet URL (Only one that responds)
	// If keys are valid Demo keys, they might work here for Futures or specific Sandbox logic?
	cfg.API.Bitget.RestURL = "https://api.bitget.com"

	// Set API Keys from Secret
	cfg.API.Bitget.AccessKey = secretCfg.API.Bitget.AccessKey
	cfg.API.Bitget.SecretKey = secretCfg.API.Bitget.SecretKey
	cfg.API.Bitget.Passphrase = secretCfg.API.Bitget.Passphrase

	// 4. Create Execution Factory and Engine
	factory := execution.NewExecutionFactory(cfg)

	// CreateExecution reads mode from cfg.Trading.Mode we just set
	execEngine, err := factory.CreateExecution()
	if err != nil {
		slog.Error("❌ Failed to create execution engine", "error", err)
		os.Exit(1)
	}
	// Ensure Client wipes its internal keys on exit
	defer execEngine.Close()

	// 4. Test Scenario: Place & Cancel Order
	ctx := context.Background()

	// 4.1 Place Limit Buy Order (Safe Price)
	// BTCUSDT at $10,000 (Far below current price ~100k)
	// Qty: 0.001 BTC
	order := domain.Order{
		ID:           "TEST-" + fmt.Sprintf("%d", time.Now().Unix()),
		Symbol:       "BTCUSDT",
		Side:         domain.SideBuy,
		Type:         domain.OrderTypeLimit,
		PriceMicros:  10_000_000_000, // $10,000.00
		QtySats:      100_000,        // 0.001 BTC
		CreatedUnixM: time.Now().UnixMicro(),
		Status:       domain.OrderStatusNew,
	}

	slog.Info("STEP 1: Placing Order...", "oid", order.ID, "price", "$10,000")
	if err := execEngine.ExecuteOrder(ctx, order); err != nil {
		slog.Error("❌ ExecuteOrder Failed", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ Order Placed Successfully")

	// 4.2 Wait
	time.Sleep(2 * time.Second)

	// 4.3 Cancel Order
	slog.Info("STEP 2: Canceling Order...", "oid", order.ID)
	if err := execEngine.CancelOrder(ctx, order.ID, order.Symbol); err != nil {
		slog.Error("❌ CancelOrder Failed", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ Order Canceled Successfully")
	slog.Info("🎉 Integration Test Passed!")
}

// Helper for format (imports need "fmt")
