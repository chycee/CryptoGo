package bitget

import (
	"context"
	"encoding/json"

	"crypto_go/internal/event"
	"crypto_go/internal/infra"
	"crypto_go/pkg/quant"

	"github.com/gorilla/websocket"
)

// FuturesWorker handles Bitget Futures WebSocket using BaseWSWorker.
type FuturesWorker struct {
	base    *infra.BaseWSWorker
	symbols map[string]string
	inbox   chan<- event.Event
	seq     *uint64
}

// NewFuturesWorker factory.
func NewFuturesWorker(symbols map[string]string, inbox chan<- event.Event, seq *uint64) *FuturesWorker {
	w := &FuturesWorker{
		symbols: symbols,
		inbox:   inbox,
		seq:     seq,
	}
	w.base = infra.NewBaseWSWorker(w)
	return w
}

func (w *FuturesWorker) ID() string     { return "BITGET_FUTURES" }
func (w *FuturesWorker) GetURL() string { return futuresWSURL }

func (w *FuturesWorker) Connect(ctx context.Context) error {
	w.base.Start(ctx)
	return nil
}

func (w *FuturesWorker) Disconnect() {
	w.base.Stop()
}

func (w *FuturesWorker) OnConnect(ctx context.Context, conn *websocket.Conn) error {
	args := make([]subscribeArg, 0, len(w.symbols))
	for _, id := range w.symbols {
		// V2 API uses USDT-FUTURES
		args = append(args, subscribeArg{InstType: "USDT-FUTURES", Channel: "ticker", InstId: id})
	}
	req := subscribeRequest{Op: "subscribe", Args: args}
	b, _ := json.Marshal(req)
	return w.base.Write(websocket.TextMessage, b)
}

func (w *FuturesWorker) OnMessage(ctx context.Context, msg []byte) {
	if string(msg) == "pong" {
		return
	}

	var resp tickerResponse
	if err := json.Unmarshal(msg, &resp); err != nil {
		return
	}
	if resp.Arg.Channel != "ticker" || resp.Data == nil {
		return
	}

	ts := quant.TimeStamp(resp.Ts * 1000)

	for _, data := range resp.Data {
		symbol := w.findSymbol(data.InstId)
		if symbol == "" {
			continue
		}

		ev := event.AcquireMarketUpdateEvent()
		ev.Seq = quant.NextSeq(w.seq)
		ev.Ts = ts
		ev.Symbol = symbol
		ev.PriceMicros = quant.ToPriceMicrosStr(data.LastPr)
		ev.QtySats = quant.ToQtySatsStr(data.Volume24h)
		ev.Exchange = "BITGET_FUTURES"

		select {
		case w.inbox <- ev:
		default:
			event.ReleaseMarketUpdateEvent(ev)
		}
	}
}

func (w *FuturesWorker) OnPing(ctx context.Context, conn *websocket.Conn) error {
	return w.base.Write(websocket.TextMessage, []byte("ping"))
}

func (w *FuturesWorker) findSymbol(instId string) string {
	for s, id := range w.symbols {
		if id == instId {
			return s
		}
	}
	return ""
}
