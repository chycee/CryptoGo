package infra

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
	upbitWSURL        = "wss://api.upbit.com/websocket/v1"
	upbitMaxRetries   = 10
	upbitBaseDelay    = 1 * time.Second
	upbitMaxDelay     = 60 * time.Second
	upbitPingInterval = 30 * time.Second
	upbitReadTimeout  = 60 * time.Second
)

// upbitTickerResponse represents Upbit WebSocket ticker response (Full API Spec)
// Reference: https://docs.upbit.com/reference/websocket-ticker
type upbitTickerResponse struct {
	// 기본 정보
	Type string `json:"type"` // 데이터 타입 (ticker)
	Code string `json:"code"` // 마켓 코드 (e.g., "KRW-BTC")

	// 가격 정보
	OpeningPrice     float64 `json:"opening_price"`      // 시가
	HighPrice        float64 `json:"high_price"`         // 고가
	LowPrice         float64 `json:"low_price"`          // 저가
	TradePrice       float64 `json:"trade_price"`        // 현재가 (종가)
	PrevClosingPrice float64 `json:"prev_closing_price"` // 전일 종가

	// 변동 정보
	Change            string  `json:"change"`              // 전일 대비: RISE, EVEN, FALL
	ChangePrice       float64 `json:"change_price"`        // 변동금액 절대값
	SignedChangePrice float64 `json:"signed_change_price"` // 변동금액 (부호 포함)
	ChangeRate        float64 `json:"change_rate"`         // 변동률 절대값
	SignedChangeRate  float64 `json:"signed_change_rate"`  // 변동률 (부호 포함)

	// 거래량/거래대금
	TradeVolume       float64 `json:"trade_volume"`         // 최근 거래량
	AccTradeVolume    float64 `json:"acc_trade_volume"`     // 누적 거래량 (UTC 0시 기준)
	AccTradeVolume24h float64 `json:"acc_trade_volume_24h"` // 24시간 누적 거래량
	AccTradePrice     float64 `json:"acc_trade_price"`      // 누적 거래대금 (UTC 0시 기준)
	AccTradePrice24h  float64 `json:"acc_trade_price_24h"`  // 24시간 누적 거래대금

	// 52주 신고/신저
	Highest52WeekPrice float64 `json:"highest_52_week_price"` // 52주 최고가
	Highest52WeekDate  string  `json:"highest_52_week_date"`  // 52주 최고가 달성일
	Lowest52WeekPrice  float64 `json:"lowest_52_week_price"`  // 52주 최저가
	Lowest52WeekDate   string  `json:"lowest_52_week_date"`   // 52주 최저가 달성일

	// 매수/매도
	AskBid string `json:"ask_bid"` // 최근 체결: ASK(매도), BID(매수)

	// 시간 정보
	TradeDate      string `json:"trade_date"`      // 최근 거래일 (YYYYMMDD)
	TradeTime      string `json:"trade_time"`      // 최근 거래시각 (HHMMSS)
	TradeTimestamp int64  `json:"trade_timestamp"` // 최근 거래 타임스탬프 (ms)
	Timestamp      int64  `json:"timestamp"`       // 데이터 수신 타임스탬프 (ms)

	// 기타
	StreamType   string `json:"stream_type"`   // SNAPSHOT, REALTIME
	SequentialID int64  `json:"sequential_id"` // 체결 고유 ID
	MarketState  string `json:"market_state"`  // 마켓 상태: PREVIEW, ACTIVE, DELISTED
}

// UpbitWorker handles Upbit WebSocket connection as a Gateway
type UpbitWorker struct {
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

// NewUpbitWorker creates a new Upbit gateway worker
func NewUpbitWorker(symbols []string, inbox chan<- event.Event, seq *uint64) *UpbitWorker {
	return &UpbitWorker{
		symbols: symbols,
		inbox:   inbox,
		seq:     seq,
	}
}

// Connect starts the WebSocket connection with automatic reconnection
func (w *UpbitWorker) Connect(ctx context.Context) error {
	ctx, w.cancel = context.WithCancel(ctx)

	w.wg.Add(1)
	go w.connectionLoop(ctx)

	return nil
}

// connectionLoop handles connection and reconnection with exponential backoff
func (w *UpbitWorker) connectionLoop(ctx context.Context) {
	defer w.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Upbit panic recovered", slog.Any("panic", r))
		}
	}()

	retryCount := 0
	for {
		select {
		case <-ctx.Done():
			slog.Info("Upbit connection loop stopped")
			return
		default:
		}

		err := w.connect(ctx)
		if err != nil {
			slog.Warn("Upbit connection failed",
				slog.Any("error", err),
				slog.Int("retry", retryCount),
			)

			// Exponential backoff
			delay := w.calculateBackoff(retryCount)
			retryCount++
			if retryCount > upbitMaxRetries {
				slog.Error("Upbit max retries exceeded, resetting counter")
				retryCount = 0
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
				continue
			}
		}

		// Connection successful, reset retry counter
		retryCount = 0

		// Read messages until error
		w.readLoop(ctx)
	}
}

// calculateBackoff returns the delay for the current retry attempt
func (w *UpbitWorker) calculateBackoff(retryCount int) time.Duration {
	// 2^retryCount calculation without float64
	exp := time.Duration(1 << uint(retryCount))
	delay := upbitBaseDelay * exp
	if delay > upbitMaxDelay || exp == 0 { // overflow check
		delay = upbitMaxDelay
	}
	return delay
}

// connect establishes WebSocket connection and subscribes to tickers
func (w *UpbitWorker) connect(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	header := make(http.Header)
	header.Add("User-Agent", DefaultUserAgent)

	conn, _, err := dialer.DialContext(ctx, upbitWSURL, header)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	w.mu.Lock()
	w.conn = conn
	w.connected = true
	w.mu.Unlock()

	// Subscribe to ticker data
	if err := w.subscribe(); err != nil {
		w.closeConnection()
		return fmt.Errorf("subscribe failed: %w", err)
	}

	slog.Info("Upbit WebSocket connected",
		slog.Int("symbols", len(w.symbols)),
	)

	return nil
}

// subscribe sends subscription message for all symbols
func (w *UpbitWorker) subscribe() error {
	if len(w.symbols) > 50 {
		slog.Warn("Upbit symbol limit exceeded (max 50)", slog.Int("count", len(w.symbols)))
		w.symbols = w.symbols[:50]
	}

	// Build codes list (e.g., ["KRW-BTC", "KRW-ETH"])
	codes := make([]string, len(w.symbols))
	for i, symbol := range w.symbols {
		codes[i] = "KRW-" + symbol
	}

	// Upbit subscription format:
	// [{"ticket":"unique-ticket"},{"type":"ticker","codes":["KRW-BTC","KRW-ETH"]}]
	subscribeMsg := []map[string]interface{}{
		{"ticket": fmt.Sprintf("crypto-go-%d", time.Now().UnixNano())},
		{"type": "ticker", "codes": codes},
	}

	msgBytes, err := json.Marshal(subscribeMsg)
	if err != nil {
		return err
	}

	w.mu.RLock()
	conn := w.conn
	w.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("connection is nil")
	}

	return w.threadSafeWrite(websocket.TextMessage, msgBytes)
}

// threadSafeWrite sends a message to the WebSocket connection in a thread-safe manner
func (w *UpbitWorker) threadSafeWrite(messageType int, data []byte) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()

	w.mu.RLock()
	conn := w.conn
	w.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("connection is nil")
	}

	return conn.WriteMessage(messageType, data)
}

// readLoop reads messages from WebSocket
func (w *UpbitWorker) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		w.mu.RLock()
		conn := w.conn
		w.mu.RUnlock()

		if conn == nil {
			return
		}

		// Set read deadline
		conn.SetReadDeadline(time.Now().Add(upbitReadTimeout))

		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Warn("Upbit WebSocket read error", slog.Any("error", err))
			}
			w.closeConnection()
			return
		}

		w.handleMessage(message)
	}
}

// handleMessage parses and processes ticker message
func (w *UpbitWorker) handleMessage(message []byte) {
	var resp upbitTickerResponse
	if err := json.Unmarshal(message, &resp); err != nil {
		slog.Debug("Upbit message parse error", slog.Any("error", err))
		return
	}

	if resp.Type != "ticker" {
		return
	}

	// Extract symbol from code (e.g., "KRW-BTC" -> "BTC")
	symbol := strings.TrimPrefix(resp.Code, "KRW-")

	// Create MarketUpdateEvent (Boundary: Float64 -> PriceMicros)
	ev := &event.MarketUpdateEvent{
		BaseEvent: event.BaseEvent{
			Seq: quant.NextSeq(w.seq),
			Ts:  quant.TimeStamp(resp.Timestamp * 1000), // ms to micros
		},
		Symbol:      symbol,
		PriceMicros: quant.ToPriceMicros(resp.TradePrice),
		QtySats:     quant.ToQtySats(resp.AccTradeVolume24h),
		Exchange:    "UPBIT",
	}

	if w.inbox != nil {
		select {
		case w.inbox <- ev:
		default:
			slog.Warn("Upbit inbox full, dropping data")
		}
	}
}

// closeConnection safely closes the WebSocket connection
func (w *UpbitWorker) closeConnection() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
	w.connected = false
}

// Disconnect closes the WebSocket connection
func (w *UpbitWorker) Disconnect() {
	if w.cancel != nil {
		w.cancel()
	}
	w.closeConnection()
	w.wg.Wait()
	slog.Info("Upbit WebSocket disconnected")
}

// IsConnected returns connection status
func (w *UpbitWorker) IsConnected() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.connected
}
