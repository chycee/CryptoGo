package execution

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"crypto_go/internal/domain"
	"crypto_go/internal/infra"
	"crypto_go/internal/infra/bitget"
	"crypto_go/pkg/quant"
)

// Mode represents the trading execution mode
type Mode string

const (
	ModePaper Mode = "PAPER"
	ModeDemo  Mode = "DEMO"
	ModeReal  Mode = "REAL"
)

// ExecutionFactory creates execution instances based on mode
type ExecutionFactory struct {
	config *infra.Config
}

// NewExecutionFactory creates a new factory
func NewExecutionFactory(cfg *infra.Config) *ExecutionFactory {
	return &ExecutionFactory{config: cfg}
}

// CreateExecution returns the appropriate Execution implementation
func (f *ExecutionFactory) CreateExecution() (domain.Execution, error) {
	mode := Mode(f.config.Trading.Mode)

	slog.Info("Initializing Execution System", "mode", mode)

	switch mode {
	case ModePaper:
		// Paper Trading: Start with 100M KRW virtual balance
		initialBalance := quant.ToPriceMicros(100_000_000.0)
		return NewPaperExecution(initialBalance), nil

	case ModeDemo:
		// Demo Trading: Connect to Bitget Testnet
		slog.Info("🔒 Connecting to Bitget DEMO (Testnet)")
		secretCfg, err := infra.LoadSecretConfig("secrets/demo.yaml")
		if err != nil {
			return nil, fmt.Errorf("failed to load demo secrets: %w", err)
		}

		// Apply secrets to main config (in-memory only)
		// Note: Ideally, we should pass separate config to client, but for now we mix it carefully
		// Creating a copy of config or modifying it is tricky in shared pointer.
		// Better approach: Pass keys directly to NewClient or update config struct.
		// For STES, let's update the config object's API section specifically for this run.
		f.config.API.Bitget.AccessKey = secretCfg.API.Bitget.AccessKey
		f.config.API.Bitget.SecretKey = secretCfg.API.Bitget.SecretKey
		f.config.API.Bitget.Passphrase = secretCfg.API.Bitget.Passphrase

		client := bitget.NewClient(f.config, true) // true = Testnet
		return NewRealExecution(client), nil

	case ModeReal:
		// Real Trading: SAFETY LATCH CHECK
		if os.Getenv("CONFIRM_REAL_MONEY") != "true" {
			err := fmt.Errorf("SAFETY_GUARD: Real trading requires 'CONFIRM_REAL_MONEY=true' environment variable")
			slog.Error(err.Error())
			panic(err) // Fail Fast
		}

		slog.Info("🚨🚨🚨 Connecting to Bitget REAL (Mainnet) 🚨🚨🚨")
		secretCfg, err := infra.LoadSecretConfig("secrets/real.yaml")
		if err != nil {
			return nil, fmt.Errorf("failed to load real secrets: %w", err)
		}

		f.config.API.Bitget.AccessKey = secretCfg.API.Bitget.AccessKey
		f.config.API.Bitget.SecretKey = secretCfg.API.Bitget.SecretKey
		f.config.API.Bitget.Passphrase = secretCfg.API.Bitget.Passphrase

		client := bitget.NewClient(f.config, false) // false = Mainnet
		return NewRealExecution(client), nil

	default:
		return nil, fmt.Errorf("unknown execution mode: %s", mode)
	}
}

// RealExecution adapts a real exchange client to the Execution interface
// This is a placeholder for now, ensuring the skeleton exists.
type RealExecution struct {
	client *bitget.Client
}

func NewRealExecution(client *bitget.Client) *RealExecution {
	return &RealExecution{client: client}
}

// Implement ExecuteOrder interface (Skeleton)
func (e *RealExecution) ExecuteOrder(ctx context.Context, order domain.Order) error {
	slog.Info("Sending Order to Exchange", "symbol", order.Symbol, "qty", order.QtySats)
	// TODO: e.client.PlaceOrder(order)
	return nil
}

// Implement CancelOrder interface (Skeleton)
func (e *RealExecution) CancelOrder(ctx context.Context, orderID string) error {
	// TODO: e.client.CancelOrder(orderID)
	return nil
}
