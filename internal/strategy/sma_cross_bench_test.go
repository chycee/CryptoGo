package strategy_test

import (
	"crypto_go/internal/domain"
	"crypto_go/internal/strategy"
	"crypto_go/pkg/quant"
	"testing"
)

// BenchmarkSMACrossStrategy_OnMarketUpdate measures strategy computation speed.
// Verifies "Zero-Alloc Ring Buffer" principle.
func BenchmarkSMACrossStrategy_OnMarketUpdate(b *testing.B) {
	strat := strategy.NewSMACrossStrategy("BTC", 20, 50)

	// Pre-fill buffer to reach steady state
	out := make([]domain.Order, 1)
	for i := 0; i < 50; i++ {
		state := domain.MarketState{
			Symbol:      "BTC",
			PriceMicros: quant.PriceMicros(50000000000 + int64(i*1000)),
		}
		strat.OnMarketUpdate(state, out)
	}

	state := domain.MarketState{
		Symbol:      "BTC",
		PriceMicros: quant.PriceMicros(51000000000),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		state.PriceMicros = quant.PriceMicros(50000000000 + int64(i%10000)*1000)
		strat.OnMarketUpdate(state, out)
	}
}

// BenchmarkSMACrossStrategy_ColdStart measures strategy initialization overhead.
func BenchmarkSMACrossStrategy_ColdStart(b *testing.B) {
	out := make([]domain.Order, 1)
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		strat := strategy.NewSMACrossStrategy("BTC", 20, 50)
		state := domain.MarketState{
			Symbol:      "BTC",
			PriceMicros: quant.PriceMicros(50000000000),
		}
		strat.OnMarketUpdate(state, out)
	}
}
