package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"crypto_go/pkg/quant"
)

func main() {
	fmt.Println("=== CryptoGo Fixed-Point Price Fetcher ===")
	fmt.Println()

	// 1. Upbit KRW-BTC
	upbitPrice := fetchUpbitPrice()
	fmt.Printf("📊 업비트 KRW-BTC\n")
	fmt.Printf("   원본 문자열: %s\n", upbitPrice.Raw)
	fmt.Printf("   PriceMicros: %d\n", upbitPrice.Micros)
	fmt.Printf("   표시 가격:   ₩%s\n", formatKRW(upbitPrice.Micros))
	fmt.Println()

	// 2. Bitget Spot BTCUSDT
	spotPrice := fetchBitgetSpotPrice()
	fmt.Printf("📊 비트겟 현물 BTCUSDT\n")
	fmt.Printf("   원본 문자열: %s\n", spotPrice.Raw)
	fmt.Printf("   PriceMicros: %d\n", spotPrice.Micros)
	fmt.Printf("   표시 가격:   $%s\n", formatUSD(spotPrice.Micros))
	fmt.Println()

	// 3. Bitget Futures BTCUSDT
	futuresPrice := fetchBitgetFuturesPrice()
	fmt.Printf("📊 비트겟 선물 BTCUSDT\n")
	fmt.Printf("   원본 문자열: %s\n", futuresPrice.Raw)
	fmt.Printf("   PriceMicros: %d\n", futuresPrice.Micros)
	fmt.Printf("   표시 가격:   $%s\n", formatUSD(futuresPrice.Micros))
	fmt.Println()

	// 4. Spot vs Futures 차이
	diff := spotPrice.Micros - futuresPrice.Micros
	fmt.Printf("💹 스팟-선물 차이: %d micros ($%.2f)\n", diff, float64(diff)/1_000_000)
	fmt.Println()
	fmt.Println("✅ 모든 가격이 float64 없이 int64로 처리됨!")
}

type PriceResult struct {
	Raw    string
	Micros quant.PriceMicros
}

func fetchUpbitPrice() PriceResult {
	resp, err := http.Get("https://api.upbit.com/v1/ticker?markets=KRW-BTC")
	if err != nil {
		return PriceResult{Raw: "ERROR", Micros: 0}
	}
	defer resp.Body.Close()

	var data []struct {
		TradePrice json.Number `json:"trade_price"`
	}
	json.NewDecoder(resp.Body).Decode(&data)

	if len(data) == 0 {
		return PriceResult{Raw: "NO_DATA", Micros: 0}
	}

	raw := data[0].TradePrice.String()
	// Remove trailing zeros after decimal for cleaner display
	raw = strings.TrimRight(strings.TrimRight(raw, "0"), ".")

	return PriceResult{
		Raw:    raw,
		Micros: quant.ToPriceMicrosStr(data[0].TradePrice.String()),
	}
}

func fetchBitgetSpotPrice() PriceResult {
	resp, err := http.Get("https://api.bitget.com/api/v2/spot/market/tickers?symbol=BTCUSDT")
	if err != nil {
		return PriceResult{Raw: "ERROR", Micros: 0}
	}
	defer resp.Body.Close()

	var data struct {
		Data []struct {
			LastPr string `json:"lastPr"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&data)

	if len(data.Data) == 0 {
		return PriceResult{Raw: "NO_DATA", Micros: 0}
	}

	return PriceResult{
		Raw:    data.Data[0].LastPr,
		Micros: quant.ToPriceMicrosStr(data.Data[0].LastPr),
	}
}

func fetchBitgetFuturesPrice() PriceResult {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.bitget.com/api/v2/mix/market/ticker?symbol=BTCUSDT&productType=USDT-FUTURES")
	if err != nil {
		return PriceResult{Raw: "ERROR", Micros: 0}
	}
	defer resp.Body.Close()

	var data struct {
		Data []struct {
			LastPr string `json:"lastPr"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&data)

	if len(data.Data) == 0 {
		return PriceResult{Raw: "NO_DATA", Micros: 0}
	}

	return PriceResult{
		Raw:    data.Data[0].LastPr,
		Micros: quant.ToPriceMicrosStr(data.Data[0].LastPr),
	}
}

func formatKRW(micros quant.PriceMicros) string {
	krw := int64(micros) / 1_000_000
	return fmt.Sprintf("%d", krw)
}

func formatUSD(micros quant.PriceMicros) string {
	dollars := float64(micros) / 1_000_000
	return fmt.Sprintf("%.2f", dollars)
}
