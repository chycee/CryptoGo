package bitget

import (
	"context"
	"encoding/json"

	"crypto_go/internal/event"
	"crypto_go/internal/infra"
	"crypto_go/pkg/quant"

	"github.com/gorilla/websocket"
)

// SpotWorker handles Bitget Spot WebSocket using BaseWSWorker.
type SpotWorker struct {
	base    *infra.BaseWSWorker
	symbols map[string]string
	inbox   chan<- event.Event
	seq     *uint64
}

// NewSpotWorker factory.
func NewSpotWorker(symbols map[string]string, inbox chan<- event.Event, seq *uint64) *SpotWorker {
	w := &SpotWorker{
		symbols: symbols,
		inbox:   inbox,
		seq:     seq,
	}
	w.base = infra.NewBaseWSWorker(w)
	return w
}

func (w *SpotWorker) ID() string     { return "BITGET_SPOT" }
func (w *SpotWorker) GetURL() string { return spotWSURL }

func (w *SpotWorker) Connect(ctx context.Context) error {
	w.base.Start(ctx)
	return nil
}

func (w *SpotWorker) Disconnect() {
	w.base.Stop()
}

func (w *SpotWorker) OnConnect(ctx context.Context, conn *websocket.Conn) error {
	args := make([]subscribeArg, 0, len(w.symbols))
	for _, id := range w.symbols {
		args = append(args, subscribeArg{InstType: "SPOT", Channel: "ticker", InstId: id})
	}
	req := subscribeRequest{Op: "subscribe", Args: args}
	b, _ := json.Marshal(req)
	return w.base.Write(websocket.TextMessage, b)
}

func (w *SpotWorker) OnMessage(ctx context.Context, msg []byte) {
	if string(msg) == "pong" {
		return
	}

	var resp tickerResponse
	if err := json.Unmarshal(msg, &resp); err != nil {
		return
	}
	if resp.Arg.Channel != "ticker" || len(resp.Data) == 0 {
		return
	}

	// Bitget sends Timestamp in Milliseconds (int64)
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
		ev.QtySats = quant.ToQtySatsStr(data.BaseVolume)
		ev.Exchange = "BITGET_SPOT"

		select {
		case w.inbox <- ev:
		default:
			event.ReleaseMarketUpdateEvent(ev)
		}
	}
}

func (w *SpotWorker) OnPing(ctx context.Context, conn *websocket.Conn) error {
	return w.base.Write(websocket.TextMessage, []byte("ping"))
}

func (w *SpotWorker) findSymbol(instId string) string {
	for s, id := range w.symbols {
		if id == instId {
			return s
		}
	}
	return ""
}
