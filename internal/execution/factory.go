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
		// TODO: Load secrets/demo.yaml
		// cfg := infra.LoadSecretConfig("secrets/demo.yaml")
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
		// TODO: Load secrets/real.yaml
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
