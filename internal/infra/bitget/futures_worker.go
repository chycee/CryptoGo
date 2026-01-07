package bitget

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"crypto_go/internal/event"
	"crypto_go/internal/infra"
	"crypto_go/pkg/quant"

	"github.com/gorilla/websocket"
)

// FuturesWorker handles Bitget Futures WebSocket
type FuturesWorker struct {
	symbols   map[string]string
	inbox     chan<- event.Event
	seq       *uint64
	conn      *websocket.Conn
	mu        sync.RWMutex
	writeMu   sync.Mutex
	connected bool
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewFuturesWorker factory
func NewFuturesWorker(symbols map[string]string, inbox chan<- event.Event, seq *uint64) *FuturesWorker {
	return &FuturesWorker{
		symbols: symbols,
		inbox:   inbox,
		seq:     seq,
	}
}

func (w *FuturesWorker) Connect(ctx context.Context) error {
	ctx, w.cancel = context.WithCancel(ctx)
	w.wg.Add(1)
	go w.connectionLoop(ctx)
	return nil
}

func (w *FuturesWorker) connectionLoop(ctx context.Context) {
	defer w.wg.Done()
	retryCount := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := w.connect(ctx); err != nil {
			slog.Warn("Bitget Futures connection failed", slog.Any("error", err), slog.Int("retry", retryCount))
			retryCount++
			if retryCount > maxRetries {
				retryCount = 0
			}
			delay := infra.CalculateBackoff(retryCount)
			time.Sleep(delay)
		} else {
			retryCount = 0
			w.readLoop(ctx)
		}
	}
}

func (w *FuturesWorker) connect(ctx context.Context) error {
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.DialContext(ctx, futuresWSURL, nil)
	if err != nil {
		return err
	}

	w.mu.Lock()
	w.conn = conn
	w.connected = true
	w.mu.Unlock()

	if err := w.subscribe(); err != nil {
		w.closeConnection()
		return err
	}

	go w.pingLoop(ctx)
	slog.Info("Bitget Futures Connected")
	return nil
}

func (w *FuturesWorker) subscribe() error {
	args := make([]subscribeArg, 0, len(w.symbols))
	for _, id := range w.symbols {
		// V2 API uses USDT-FUTURES
		args = append(args, subscribeArg{InstType: "USDT-FUTURES", Channel: "ticker", InstId: id})
	}
	req := subscribeRequest{Op: "subscribe", Args: args}
	b, _ := json.Marshal(req)
	return w.threadSafeWrite(websocket.TextMessage, b)
}

func (w *FuturesWorker) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.threadSafeWrite(websocket.TextMessage, []byte("ping"))
		}
	}
}

func (w *FuturesWorker) threadSafeWrite(msgType int, data []byte) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.conn == nil {
		return fmt.Errorf("no conn")
	}
	return w.conn.WriteMessage(msgType, data)
}

func (w *FuturesWorker) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		w.mu.RLock()
		if w.conn == nil {
			w.mu.RUnlock()
			return
		}
		w.conn.SetReadDeadline(time.Now().Add(readTimeout))
		w.mu.RUnlock()

		_, msg, err := w.conn.ReadMessage()
		if err != nil {
			w.closeConnection()
			return
		}
		if string(msg) == "pong" {
			continue
		}
		w.handleMessage(msg)
	}
}

func (w *FuturesWorker) handleMessage(msg []byte) {
	var resp tickerResponse
	json.Unmarshal(msg, &resp)
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
		ev.Exchange = "BITGET_F"

		select {
		case w.inbox <- ev:
		default:
			event.ReleaseMarketUpdateEvent(ev)
		}
	}
}

func (w *FuturesWorker) findSymbol(instId string) string {
	for s, id := range w.symbols {
		if id == instId {
			return s
		}
	}
	return ""
}

func (w *FuturesWorker) closeConnection() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
	w.connected = false
}

func (w *FuturesWorker) Disconnect() {
	if w.cancel != nil {
		w.cancel()
	}
	w.closeConnection()
	w.wg.Wait()
}
