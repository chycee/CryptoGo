package domain

import (
	"testing"

	"crypto_go/pkg/quant"
)

func TestMarketData_GapPct(t *testing.T) {
	t.Run("Normal Calculation", func(t *testing.T) {
		spot := Ticker{PriceMicros: 100 * quant.PriceScale}   // 100 USD
		future := Ticker{PriceMicros: 105 * quant.PriceScale} // 105 USD

		data := MarketData{
			BitgetS: &spot,
			BitgetF: &future,
		}

		gap := data.GapPct()
		// (5 * 1,000,000) / 100 = 50,000 Micros (5%)
		if gap != 50000 {
			t.Errorf("Expected 50000 Micros (5%%), got %v", gap)
		}
	})

	t.Run("Safety: Nil Pointers", func(t *testing.T) {
		data := MarketData{}
		if data.GapPct() != 0 {
			t.Error("Should return 0 when tickers are missing")
		}
	})

	t.Run("Safety: Zero Price", func(t *testing.T) {
		spot := Ticker{PriceMicros: 0}
		future := Ticker{PriceMicros: 105 * quant.PriceScale}
		data := MarketData{BitgetS: &spot, BitgetF: &future}
		if data.GapPct() != 0 {
			t.Error("Should return 0 when spot price is zero to avoid crash")
		}
	})
}
