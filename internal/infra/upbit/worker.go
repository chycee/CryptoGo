package upbit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"crypto_go/internal/event"
	"crypto_go/pkg/quant"

	"github.com/gorilla/websocket"
)

const (
	wsURL            = "wss://api.upbit.com/websocket/v1"
	maxRetries       = 10
	baseDelay        = 1 * time.Second
	maxDelay         = 60 * time.Second
	pingInterval     = 30 * time.Second
	readTimeout      = 60 * time.Second
	DefaultUserAgent = "Mozilla/5.0"
)

// tickerResponse represents Upbit WebSocket ticker response
type tickerResponse struct {
	Type string `json:"type"` // ticker
	Code string `json:"code"` // KRW-BTC

	TradePrice        float64 `json:"trade_price"`
	AccTradeVolume24h float64 `json:"acc_trade_volume_24h"`
	Timestamp         int64   `json:"timestamp"`
}

// Worker handles Upbit WebSocket connection
type Worker struct {
	symbols   []string
	inbox     chan<- event.Event
	seq       *uint64
	conn      *websocket.Conn
	mu        sync.RWMutex
	writeMu   sync.Mutex
	connected bool
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewWorker creates a new Upbit gateway worker
func NewWorker(symbols []string, inbox chan<- event.Event, seq *uint64) *Worker {
	return &Worker{
		symbols: symbols,
		inbox:   inbox,
		seq:     seq,
	}
}

// Connect starts the WebSocket connection
func (w *Worker) Connect(ctx context.Context) error {
	ctx, w.cancel = context.WithCancel(ctx)
	w.wg.Add(1)
	go w.connectionLoop(ctx)
	return nil
}

func (w *Worker) connectionLoop(ctx context.Context) {
	defer w.wg.Done()
	retryCount := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := w.connect(ctx); err != nil {
			slog.Warn("Upbit connection failed", slog.Any("error", err), slog.Int("retry", retryCount))
			delay := w.calculateBackoff(retryCount)
			retryCount++
			if retryCount > maxRetries {
				retryCount = 0
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
				continue
			}
		} else {
			retryCount = 0
			w.readLoop(ctx)
		}
	}
}

func (w *Worker) calculateBackoff(retryCount int) time.Duration {
	exp := time.Duration(1 << uint(retryCount))
	delay := baseDelay * exp
	if delay > maxDelay || exp == 0 {
		delay = maxDelay
	}
	return delay
}

func (w *Worker) connect(ctx context.Context) error {
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	header := make(http.Header)
	// header.Add("User-Agent", DefaultUserAgent)

	conn, _, err := dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	w.mu.Lock()
	w.conn = conn
	w.connected = true
	w.mu.Unlock()

	if err := w.subscribe(); err != nil {
		w.closeConnection()
		return err
	}

	slog.Info("Upbit Connected", slog.Int("subs", len(w.symbols)))
	return nil
}

func (w *Worker) subscribe() error {
	if len(w.symbols) > 50 {
		w.symbols = w.symbols[:50]
	}
	codes := make([]string, len(w.symbols))
	for i, s := range w.symbols {
		codes[i] = "KRW-" + s
	}

	msg := []map[string]interface{}{
		{"ticket": fmt.Sprintf("go-%d", time.Now().UnixNano())},
		{"type": "ticker", "codes": codes},
	}
	b, _ := json.Marshal(msg)
	return w.threadSafeWrite(websocket.TextMessage, b)
}

func (w *Worker) threadSafeWrite(msgType int, data []byte) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.conn == nil {
		return fmt.Errorf("no conn")
	}
	return w.conn.WriteMessage(msgType, data)
}

func (w *Worker) readLoop(ctx context.Context) {
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
		w.handleMessage(msg)
	}
}

func (w *Worker) handleMessage(msg []byte) {
	var resp tickerResponse
	if json.Unmarshal(msg, &resp) != nil || resp.Type != "ticker" {
		return
	}

	symbol := strings.TrimPrefix(resp.Code, "KRW-")
	ev := &event.MarketUpdateEvent{
		BaseEvent: event.BaseEvent{
			Seq: quant.NextSeq(w.seq),
			Ts:  quant.TimeStamp(resp.Timestamp * 1000),
		},
		Symbol:      symbol,
		PriceMicros: quant.ToPriceMicros(resp.TradePrice),
		QtySats:     quant.ToQtySats(resp.AccTradeVolume24h),
		Exchange:    "UPBIT",
	}

	select {
	case w.inbox <- ev:
	default: // DROP
	}
}

func (w *Worker) closeConnection() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
	w.connected = false
}

func (w *Worker) Disconnect() {
	if w.cancel != nil {
		w.cancel()
	}
	w.closeConnection()
	w.wg.Wait()
}
