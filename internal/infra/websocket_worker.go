package infra

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketHandler defines exchange-specific logic for the BaseWSWorker.
type WebSocketHandler interface {
	GetURL() string
	OnConnect(ctx context.Context, conn *websocket.Conn) error
	OnMessage(ctx context.Context, msg []byte)
	OnPing(ctx context.Context, conn *websocket.Conn) error
	ID() string
}

// BaseWSWorker manages the lifecycle of a WebSocket connection.
// It handles reconnection with backoff, read timeouts, and thread-safe writes.
type BaseWSWorker struct {
	handler WebSocketHandler
	mu      sync.RWMutex
	conn    *websocket.Conn
	writeMu sync.Mutex
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	ReadTimeout  time.Duration
	PingInterval time.Duration
}

// NewBaseWSWorker creates a new generic WebSocket worker.
func NewBaseWSWorker(handler WebSocketHandler) *BaseWSWorker {
	return &BaseWSWorker{
		handler:      handler,
		ReadTimeout:  60 * time.Second,
		PingInterval: 30 * time.Second,
	}
}

// Start initiates the connection loop.
func (w *BaseWSWorker) Start(ctx context.Context) {
	ctx, w.cancel = context.WithCancel(ctx)
	w.wg.Add(1)
	go w.runLoop(ctx)
}

// Stop terminates the worker.
func (w *BaseWSWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	w.close()
	w.wg.Wait()
}

func (w *BaseWSWorker) runLoop(ctx context.Context) {
	defer w.wg.Done()
	retry := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := w.connect(ctx); err != nil {
			slog.Warn("WS Connection failed", "id", w.handler.ID(), "err", err, "retry", retry)
			delay := CalculateBackoff(retry)
			retry++

			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
				continue
			}
		}

		retry = 0 // Reset on successful connect
		w.process(ctx)
	}
}

func (w *BaseWSWorker) connect(ctx context.Context) error {
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	header := make(http.Header)
	header.Set("User-Agent", GetUserAgent())

	conn, _, err := dialer.DialContext(ctx, w.handler.GetURL(), header)
	if err != nil {
		return err
	}

	w.mu.Lock()
	w.conn = conn
	w.mu.Unlock()

	if err := w.handler.OnConnect(ctx, conn); err != nil {
		w.close()
		return fmt.Errorf("OnConnect failed: %w", err)
	}

	if w.PingInterval > 0 {
		go w.pingLoop(ctx)
	}

	slog.Info("WS Connected", "id", w.handler.ID())
	return nil
}

func (w *BaseWSWorker) process(ctx context.Context) {
	for {
		w.mu.RLock()
		c := w.conn
		w.mu.RUnlock()
		if c == nil {
			return
		}

		c.SetReadDeadline(time.Now().Add(w.ReadTimeout))
		_, msg, err := c.ReadMessage()
		if err != nil {
			slog.Warn("WS Read error", "id", w.handler.ID(), "err", err)
			w.close()
			return
		}

		w.handler.OnMessage(ctx, msg)
	}
}

func (w *BaseWSWorker) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(w.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.mu.RLock()
			c := w.conn
			w.mu.RUnlock()
			if c == nil {
				return
			}
			if err := w.handler.OnPing(ctx, c); err != nil {
				slog.Warn("WS Ping error", "id", w.handler.ID(), "err", err)
				w.close()
				return
			}
		}
	}
}

func (w *BaseWSWorker) Write(msgType int, data []byte) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()

	w.mu.RLock()
	c := w.conn
	w.mu.RUnlock()

	if c == nil {
		return fmt.Errorf("ws not connected")
	}

	return c.WriteMessage(msgType, data)
}

func (w *BaseWSWorker) close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
}
