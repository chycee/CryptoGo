package upbit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"crypto_go/internal/event"
	"crypto_go/internal/infra"
	"crypto_go/pkg/quant"

	"github.com/gorilla/websocket"
)

const (
	wsURL = "wss://api.upbit.com/websocket/v1"
)

// tickerResponse represents Upbit WebSocket ticker response.
// Uses json.Number to avoid float64 precision issues (Rule #1: No Float in Hotpath).
type tickerResponse struct {
	Type string `json:"type"` // ticker
	Code string `json:"code"` // KRW-BTC

	TradePrice        json.Number `json:"trade_price"`
	AccTradeVolume24h json.Number `json:"acc_trade_volume_24h"`
	Timestamp         int64       `json:"timestamp"`
}

// Worker handles Upbit WebSocket connection using BaseWSWorker.
type Worker struct {
	base    *infra.BaseWSWorker
	symbols []string
	inbox   chan<- event.Event
	seq     *uint64
}

// NewWorker creates a new Upbit gateway worker.
func NewWorker(symbols []string, inbox chan<- event.Event, seq *uint64) *Worker {
	w := &Worker{
		symbols: symbols,
		inbox:   inbox,
		seq:     seq,
	}
	w.base = infra.NewBaseWSWorker(w)
	return w
}

// ID returns the worker identifier.
func (w *Worker) ID() string { return "UPBIT" }

// GetURL returns the Upbit WebSocket endpoint.
func (w *Worker) GetURL() string { return wsURL }

// Connect starts the WebSocket connection.
func (w *Worker) Connect(ctx context.Context) error {
	w.base.Start(ctx)
	return nil
}

// Disconnect terminates the connection.
func (w *Worker) Disconnect() {
	w.base.Stop()
}

// OnConnect handles the subscription logic after connection is established.
func (w *Worker) OnConnect(ctx context.Context, conn *websocket.Conn) error {
	codes := make([]string, 0, len(w.symbols))
	for _, s := range w.symbols {
		codes = append(codes, "KRW-"+s)
	}

	msg := []map[string]interface{}{
		{"ticket": fmt.Sprintf("go-%d", time.Now().UnixNano())},
		{"type": "ticker", "codes": codes},
	}
	b, _ := json.Marshal(msg)
	return w.base.Write(websocket.TextMessage, b)
}

// OnMessage handles incoming ticker updates.
func (w *Worker) OnMessage(ctx context.Context, msg []byte) {
	var resp tickerResponse
	if err := json.Unmarshal(msg, &resp); err != nil || resp.Type != "ticker" {
		return
	}

	symbol := strings.TrimPrefix(resp.Code, "KRW-")

	// Optimization: Use Pool and int64 conversion (Rule #1, #3)
	ev := event.AcquireMarketUpdateEvent()
	ev.Seq = quant.NextSeq(w.seq)
	ev.Ts = quant.TimeStamp(resp.Timestamp * 1000)
	ev.Symbol = symbol
	ev.PriceMicros = quant.ToPriceMicrosStr(resp.TradePrice.String())
	ev.QtySats = quant.ToQtySatsStr(resp.AccTradeVolume24h.String())
	ev.Exchange = "UPBIT"

	select {
	case w.inbox <- ev:
	default:
		// Drop if inbox is full, but release to pool to prevent leak.
		event.ReleaseMarketUpdateEvent(ev)
	}
}

// OnPing is called by BaseWSWorker. Upbit doesn't require explicit ping,
// as it uses Pong frames, but we leave it as a no-op or default.
func (w *Worker) OnPing(ctx context.Context, conn *websocket.Conn) error {
	// Upbit doesn't need explicit application-level ping for ticker.
	return nil
}
