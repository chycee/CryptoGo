package bitget

import (
	"crypto_go/pkg/quant"
)

const (
	spotWSURL    = "wss://ws.bitget.com/v2/ws/public"
	futuresWSURL = "wss://ws.bitget.com/v2/ws/public"
)

// subscribeRequest Structure
type subscribeRequest struct {
	Op   string         `json:"op"`
	Args []subscribeArg `json:"args"`
}

type subscribeArg struct {
	InstType string `json:"instType"`
	Channel  string `json:"channel"`
	InstId   string `json:"instId"`
}

// tickerResponse Structure
type tickerResponse struct {
	Action string       `json:"action"`
	Arg    subscribeArg `json:"arg"`
	Data   []tickerData `json:"data"`
	Ts     int64        `json:"ts"`
}

type tickerData struct {
	InstId     string `json:"instId"`
	LastPr     string `json:"lastPr"`     // Spot & Futures
	BaseVolume string `json:"baseVolume"` // Spot
	Volume24h  string `json:"volume24h"`  // Futures
}

func NextSeq(seq *uint64) uint64 {
	return quant.NextSeq(seq)
}
