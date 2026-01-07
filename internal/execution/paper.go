package execution

import (
	"context"
	"crypto_go/internal/domain"
	"crypto_go/pkg/quant"
	"crypto_go/pkg/safe"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Fill represents a simulated order fill.
type Fill struct {
	OrderID      string
	Symbol       string
	Side         string // "BUY" or "SELL"
	PriceMicros  quant.PriceMicros
	QtySats      quant.QtySats
	TsUnixMicros int64
}

// PaperExecution simulates order execution with virtual balances.
// This is used for strategy backtesting and pre-production validation.
type PaperExecution struct {
	balances *domain.BalanceBook
	orders   map[string]*domain.Order
	fills    []Fill
	mu       sync.Mutex

	// Current market prices for PnL calculation
	prices map[string]quant.PriceMicros
}

// NewPaperExecution creates a new paper trading executor.
func NewPaperExecution(initialBalance quant.PriceMicros) *PaperExecution {
	// Default to USDT initial balance
	balances := domain.NewBalanceBook()
	balances.Get("USDT").Credit(int64(initialBalance), 0)

	return &PaperExecution{
		balances: balances,
		orders:   make(map[string]*domain.Order),
		fills:    make([]Fill, 0),
		prices:   make(map[string]quant.PriceMicros),
	}
}

// Deposit adds funds to the virtual account.
func (p *PaperExecution) Deposit(symbol string, amountSats int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	balance := p.balances.Get(symbol)
	balance.Credit(amountSats, 0)
}

// UpdatePrice updates current market price for a symbol.
func (p *PaperExecution) UpdatePrice(symbol string, priceMicros quant.PriceMicros) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.prices[symbol] = priceMicros
}

// ExecuteOrder executes a market order immediately against virtual balance.
// For MARKET orders, uses current price. For LIMIT orders, uses order price.
func (p *PaperExecution) ExecuteOrder(ctx context.Context, order domain.Order) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Determine execution price
	var execPrice quant.PriceMicros
	if order.Type == "MARKET" {
		price, ok := p.prices[order.Symbol]
		if !ok {
			return fmt.Errorf("no price available for %s", order.Symbol)
		}
		execPrice = price
	} else {
		execPrice = quant.PriceMicros(order.PriceMicros)
	}

	// Calculate required amount
	// BUY: need quote currency (e.g., USDT)
	// SELL: need base currency (e.g., BTC)
	baseSymbol := order.Symbol[:3]  // e.g., "BTC" from "BTCUSDT"
	quoteSymbol := order.Symbol[3:] // e.g., "USDT" from "BTCUSDT"

	if order.Side == "BUY" {
		// Need quote currency: price * qty
		requiredQuote := safe.SafeMul(int64(execPrice), order.QtySats)
		// Scale down (price is in Micros, qty is in Sats)
		requiredQuote = safe.SafeDiv(requiredQuote, quant.QtyScale)

		quoteBalance := p.balances.Get(quoteSymbol)
		if quoteBalance.AvailableSats() < requiredQuote {
			return fmt.Errorf("insufficient %s balance: need %d, have %d",
				quoteSymbol, requiredQuote, quoteBalance.AvailableSats())
		}

		// Execute: debit quote, credit base
		quoteBalance.Debit(requiredQuote, 0)
		baseBalance := p.balances.Get(baseSymbol)
		baseBalance.Credit(order.QtySats, 0)

	} else { // SELL
		baseBalance := p.balances.Get(baseSymbol)
		if baseBalance.AvailableSats() < order.QtySats {
			return fmt.Errorf("insufficient %s balance: need %d, have %d",
				baseSymbol, order.QtySats, baseBalance.AvailableSats())
		}

		// Execute: debit base, credit quote
		baseBalance.Debit(order.QtySats, 0)
		creditQuote := safe.SafeMul(int64(execPrice), order.QtySats)
		creditQuote = safe.SafeDiv(creditQuote, quant.QtyScale)
		quoteBalance := p.balances.Get(quoteSymbol)
		quoteBalance.Credit(creditQuote, 0)
	}

	// Record fill
	fill := Fill{
		OrderID:      order.ID,
		Symbol:       order.Symbol,
		Side:         order.Side,
		PriceMicros:  execPrice,
		QtySats:      quant.QtySats(order.QtySats),
		TsUnixMicros: time.Now().UnixMicro(),
	}
	p.fills = append(p.fills, fill)

	// Update order status
	order.Status = "FILLED"
	p.orders[order.ID] = &order

	slog.Info("PAPER EXECUTION: Order Filled",
		slog.String("id", order.ID),
		slog.String("symbol", order.Symbol),
		slog.String("side", order.Side),
		slog.Int64("price", int64(execPrice)),
		slog.Int64("qty", order.QtySats))

	return nil
}

// CancelOrder cancels an unfilled order.
func (p *PaperExecution) CancelOrder(ctx context.Context, orderID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	order, ok := p.orders[orderID]
	if !ok {
		return fmt.Errorf("order not found: %s", orderID)
	}

	if order.Status == "FILLED" {
		return fmt.Errorf("cannot cancel filled order: %s", orderID)
	}

	order.Status = "CANCELED"
	slog.Info("PAPER EXECUTION: Order Canceled", slog.String("id", orderID))
	return nil
}

// GetFills returns all executed fills.
func (p *PaperExecution) GetFills() []Fill {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]Fill, len(p.fills))
	copy(result, p.fills)
	return result
}

// GetBalance returns balance for a symbol.
func (p *PaperExecution) GetBalance(symbol string) domain.Balance {
	p.mu.Lock()
	defer p.mu.Unlock()
	return *p.balances.Get(symbol)
}

// GetTotalEquityMicros calculates total portfolio value in quote currency.
func (p *PaperExecution) GetTotalEquityMicros() int64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	priceMap := make(map[string]int64)
	for k, v := range p.prices {
		priceMap[k] = int64(v)
	}

	return p.balances.CalculateTotalEquity(priceMap)
}
