package event

import (
	"crypto_go/pkg/quant"
)

// Type defines the type of event.
type Type uint16

const (
	EvMarketUpdate Type = iota + 1
	EvOrderUpdate
	EvBalanceUpdate
	EvSystemHalt
)

// Event is the interface for all sequencer events.
type Event interface {
	GetSeq() uint64
	GetTs() quant.TimeStamp
	GetType() Type
}

// BaseEvent contains common fields for all events.
type BaseEvent struct {
	Seq uint64          `json:"seq"`
	Ts  quant.TimeStamp `json:"ts"`
}

func (e BaseEvent) GetSeq() uint64         { return e.Seq }
func (e BaseEvent) GetTs() quant.TimeStamp { return e.Ts }

// MarketUpdateEvent represents a price change in the market.
type MarketUpdateEvent struct {
	BaseEvent
	Symbol      string            `json:"symbol"`
	PriceMicros quant.PriceMicros `json:"price"`
	QtySats     quant.QtySats     `json:"qty"`
	Exchange    string            `json:"exchange"`
}

func (e MarketUpdateEvent) GetType() Type { return EvMarketUpdate }

// OrderUpdateEvent represents an order status change.
type OrderUpdateEvent struct {
	BaseEvent
	OrderID            string            `json:"order_id"`
	Status             string            `json:"status"`
	PriceMicros        quant.PriceMicros `json:"price"`
	AccumulatedQtySats quant.QtySats     `json:"qty"`
}

func (e OrderUpdateEvent) GetType() Type { return EvOrderUpdate }
