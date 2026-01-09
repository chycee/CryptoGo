package strategy

import (
	"crypto_go/internal/domain"
	"crypto_go/pkg/safe"
)

// SMACrossStrategy implements a simple SMA Crossover strategy.
// It is stateful and deterministic.
// OPTIMIZED: Uses a Ring Buffer and field alignment for Zero-Alloc Hotpath & Cache Efficiency.
type SMACrossStrategy struct {
	// 64-bit fields grouped for alignment (Rule #3: Cache-Line Friendly)
	sum          int64
	prevShortSMA int64
	prevLongSMA  int64
	prices       []int64

	// Metadata and pointers
	symbol      string
	shortPeriod int
	longPeriod  int
	head        int
	count       int
}

// NewSMACrossStrategy creates a new instance.
func NewSMACrossStrategy(symbol string, shortPeriod, longPeriod int) *SMACrossStrategy {
	if shortPeriod >= longPeriod {
		panic("SMACrossStrategy: shortPeriod must be less than longPeriod")
	}
	return &SMACrossStrategy{
		symbol:      symbol,
		shortPeriod: shortPeriod,
		longPeriod:  longPeriod,
		prices:      make([]int64, longPeriod), // Fixed size allocation during init
	}
}

// OnMarketUpdate processes market updates and generates signals.
// Zero-Alloc: Populates the 'out' buffer instead of returning a new slice.
func (s *SMACrossStrategy) OnMarketUpdate(state domain.MarketState, out []domain.Order) int {
	// 1. Filter by symbol
	if state.Symbol != s.symbol {
		return 0
	}

	currentPrice := int64(state.PriceMicros)

	// 2. Update Price History (Ring Buffer)
	// If full, subtract the oldest value from sum before overwriting
	if s.count == s.longPeriod {
		oldestPrice := s.prices[s.head] // s.head points to the oldest value when full
		s.sum = safe.SafeSub(s.sum, oldestPrice)
	}

	// Add new price
	s.prices[s.head] = currentPrice
	s.sum = safe.SafeAdd(s.sum, currentPrice)

	// Move head
	s.head = (s.head + 1) % s.longPeriod

	// Increment count if not yet full
	if s.count < s.longPeriod {
		s.count++
	}

	// 3. Check if we have enough data
	if s.count < s.longPeriod {
		return 0
	}

	// 4. Calculate SMAs
	// Long SMA is easy: s.sum / s.longPeriod
	currLongSMA := safe.SafeDiv(s.sum, int64(s.longPeriod))

	// Short SMA requires manual calculation over the ring buffer
	currShortSMA := s.calculateShortSMA()

	signalCount := 0

	// 5. Check for Cross
	if s.prevShortSMA != 0 && s.prevLongSMA != 0 {
		// Golden Cross: Short goes above Long -> BUY
		if s.prevShortSMA <= s.prevLongSMA && currShortSMA > currLongSMA {
			if signalCount < len(out) {
				out[signalCount] = domain.Order{
					Symbol:      s.symbol,
					Side:        "BUY",
					Type:        "MARKET",
					PriceMicros: currentPrice, // Market order doesn't strictly need price, but good for reference
					QtySats:     10000,        // Hardcoded for MVP
					Status:      "NEW",
				}
				signalCount++
			}
		}

		// Dead Cross: Short goes below Long -> SELL
		if s.prevShortSMA >= s.prevLongSMA && currShortSMA < currLongSMA {
			if signalCount < len(out) {
				out[signalCount] = domain.Order{
					Symbol:      s.symbol,
					Side:        "SELL",
					Type:        "MARKET",
					PriceMicros: currentPrice,
					QtySats:     10000,
					Status:      "NEW",
				}
				signalCount++
			}
		}
	}

	// 6. Update State
	s.prevShortSMA = currShortSMA
	s.prevLongSMA = currLongSMA

	return signalCount
}

// OnOrderUpdate handles order updates (Empty for now)
func (s *SMACrossStrategy) OnOrderUpdate(order domain.Order) {
	// TODO: Update internal state based on fills if needed
}

// calculateShortSMA calculates the SMA for the short period using the ring buffer.
func (s *SMACrossStrategy) calculateShortSMA() int64 {
	var sum int64 = 0
	// Walk backwards from current head (which points to next write slot, so head-1 is latest)
	idx := s.head
	for i := 0; i < s.shortPeriod; i++ {
		idx--
		if idx < 0 {
			idx = s.longPeriod - 1
		}
		sum = safe.SafeAdd(sum, s.prices[idx])
	}
	return safe.SafeDiv(sum, int64(s.shortPeriod))
}
