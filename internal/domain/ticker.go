package domain

import (
	"crypto_go/pkg/quant"
	"crypto_go/pkg/safe"
)

// Ticker represents price data from a single exchange
type Ticker struct {
	Symbol           string            `json:"symbol"`      // Unified symbol (e.g., "BTC")
	PriceMicros      quant.PriceMicros `json:"price"`       // Current price
	VolumeSats       quant.QtySats     `json:"volume"`      // 24h volume
	ChangeRateMicros int64             `json:"change_rate"` // 24h change (Micros: 0.01 = 10,000)
	Exchange         string            `json:"exchange"`    // "UPBIT", "BITGET_S", "BITGET_F"
	Precision        int               `json:"precision"`   // Decimal places from exchange

	// Bitget Futures specific
	FundingRateMicros int64             `json:"funding_rate,omitempty"`      // Micros
	NextFundingUnix   *int64            `json:"next_funding_time,omitempty"` // Unix Micro
	HighPriceMicros   quant.PriceMicros `json:"historical_high,omitempty"`
	LowPriceMicros    quant.PriceMicros `json:"historical_low,omitempty"`
}

// MarketData aggregates data for a single symbol from all exchanges
type MarketData struct {
	Symbol        string  `json:"symbol"`
	Upbit         *Ticker `json:"upbit,omitempty"`
	BitgetS       *Ticker `json:"bitget_s,omitempty"`
	BitgetF       *Ticker `json:"bitget_f,omitempty"`
	PremiumMicros int64   `json:"premium,omitempty"` // Kimchi Premium (Micros)
	StatusMsg     string  `json:"status_msg"`
	IsFavorite    bool    `json:"is_favorite"`
}

// GapPct calculates Futures vs Spot gap percentage (Micros)
func (m *MarketData) GapPct() int64 {
	if m.BitgetS == nil || m.BitgetF == nil || m.BitgetS.PriceMicros == 0 {
		return 0
	}

	// GapPct calculates Futures vs Spot gap. Result is in Micros (1% = 10,000).
	// gap_micros = (Future - Spot) * 1,000,000 / Spot
	diff := safe.SafeSub(int64(m.BitgetF.PriceMicros), int64(m.BitgetS.PriceMicros))
	num := safe.SafeMul(diff, quant.PriceScale)
	return safe.SafeDiv(num, int64(m.BitgetS.PriceMicros))
}

// ChangeDirection returns "positive", "negative", or "neutral"
func (m *MarketData) ChangeDirection() string {
	if m.Upbit == nil {
		return "neutral"
	}
	if m.Upbit.ChangeRateMicros > 0 {
		return "positive"
	}
	if m.Upbit.ChangeRateMicros < 0 {
		return "negative"
	}
	return "neutral"
}
