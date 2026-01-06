package infra

import (
	"time"
)

// =====================================================
// Bitget WebSocket 공통 ?�수 �??�???�의
// =====================================================

const (
	bitgetSpotWSURL    = "wss://ws.bitget.com/v2/ws/public"
	bitgetFuturesWSURL = "wss://ws.bitget.com/v2/ws/public"
	bitgetMaxRetries   = 10
	bitgetBaseDelay    = 1 * time.Second
	bitgetMaxDelay     = 60 * time.Second
	bitgetPingInterval = 25 * time.Second
	bitgetReadTimeout  = 35 * time.Second
)

// bitgetSubscribeRequest represents Bitget WebSocket subscription request
type bitgetSubscribeRequest struct {
	Op   string               `json:"op"`
	Args []bitgetSubscribeArg `json:"args"`
}

type bitgetSubscribeArg struct {
	InstType string `json:"instType"`
	Channel  string `json:"channel"`
	InstId   string `json:"instId"`
}

// bitgetTickerResponse represents Bitget WebSocket ticker response
type bitgetTickerResponse struct {
	Action string `json:"action"` // snapshot, update
	Arg    struct {
		InstType string `json:"instType"` // SPOT, USDT-FUTURES
		Channel  string `json:"channel"`  // ticker
		InstId   string `json:"instId"`   // Trading pair (e.g., BTCUSDT)
	} `json:"arg"`
	Data []bitgetTickerData `json:"data"`
	Ts   string             `json:"ts"` // Response timestamp
}

// bitgetTickerData represents Bitget ticker data (Full API Spec)
// Reference: https://www.bitget.com/api-doc/common/websocket-intro
type bitgetTickerData struct {
	// 기본 ?�보
	InstId string `json:"instId"` // 거래??(e.g., BTCUSDT)
	Symbol string `json:"symbol"` // ?�볼 (?��? ?�답?�서 ?�용)

	// 가�??�보
	LastPr  string `json:"lastPr"`  // 최근 체결가
	AskPr   string `json:"askPr"`   // 매도 ?��?
	BidPr   string `json:"bidPr"`   // 매수 ?��?
	AskSz   string `json:"askSz"`   // 매도 ?�량
	BidSz   string `json:"bidSz"`   // 매수 ?�량
	Open24h string `json:"open24h"` // 24?�간 ?��?
	High24h string `json:"high24h"` // 24?�간 고�?
	Low24h  string `json:"low24h"`  // 24?�간 ?�가

	// 변???�보
	Change24h    string `json:"change24h"`    // 24?�간 변?�률 (0.01 = 1%)
	ChangeUtc24h string `json:"changeUtc24h"` // UTC 기�? 24?�간 변?�률

	// 거래??
	BaseVolume  string `json:"baseVolume"`  // 기초 ?�산 거래??
	QuoteVolume string `json:"quoteVolume"` // 견적 ?�산 거래??
	UsdtVolume  string `json:"usdtVolume"`  // USDT 거래??
	OpenUtc     string `json:"openUtc"`     // UTC ?��?

	// ?�물 ?�용 ?�드
	IndexPrice      string `json:"indexPrice"`      // ?�덱??가�?
	MarkPrice       string `json:"markPrice"`       // 마크 가�?
	FundingRate     string `json:"fundingRate"`     // ?�?�비
	NextFundingTime string `json:"nextFundingTime"` // ?�음 ?�???�간 (ms)
	HoldingAmount   string `json:"holdingAmount"`   // 미결???�정

	// 배송 ?�물 ?�용
	DeliveryStartTime string `json:"deliveryStartTime"` // 배송 ?�작 ?�간
	DeliveryTime      string `json:"deliveryTime"`      // 배송 ?�간
	DeliveryStatus    string `json:"deliveryStatus"`    // 배송 ?�태
	DeliveryPrice     string `json:"deliveryPrice"`     // 배송 가�?

	// 기�?
	Ts         string `json:"ts"`         // ?�이???�?�스?�프 (ms)
	SymbolType string `json:"symbolType"` // ?�볼 ?�??
}

// =====================================================
// Helper functions
// =====================================================

func calculateBitgetBackoff(retryCount int) time.Duration {
	// 2^retryCount calculation without float64
	exp := time.Duration(1 << uint(retryCount))
	delay := bitgetBaseDelay * exp
	if delay > bitgetMaxDelay || exp == 0 { // overflow check
		delay = bitgetMaxDelay
	}
	return delay
}
