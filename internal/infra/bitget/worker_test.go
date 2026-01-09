package bitget

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"crypto_go/internal/event"

	"github.com/gorilla/websocket"
)

// createMockBitgetServer creates a mock Bitget WebSocket server
func createMockBitgetServer(t *testing.T, responses []interface{}) *httptest.Server {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer conn.Close()

		// Read subscription message
		_, _, _ = conn.ReadMessage()

		// Send responses
		for _, resp := range responses {
			data, _ := json.Marshal(resp)
			conn.WriteMessage(websocket.TextMessage, data)
			time.Sleep(10 * time.Millisecond)
		}

		time.Sleep(100 * time.Millisecond)
	}))

	return server
}

func httpToWS(url string) string {
	return strings.Replace(url, "http://", "ws://", 1)
}

func TestSpotWorker_TickerParsing(t *testing.T) {
	inbox := make(chan event.Event, 10)
	var seq uint64 = 0

	// Map: symbol -> instId (BTC -> BTCUSDT)
	worker := &SpotWorker{
		symbols: map[string]string{"BTC": "BTCUSDT"},
		inbox:   inbox,
		seq:     &seq,
	}

	// Mock Bitget spot ticker response - must match tickerResponse struct
	mockData := map[string]interface{}{
		"action": "snapshot",
		"arg": map[string]interface{}{
			"instType": "SPOT",
			"channel":  "ticker",
			"instId":   "BTCUSDT",
		},
		"data": []interface{}{
			map[string]interface{}{
				"instId":     "BTCUSDT",
				"lastPr":     "92000.50",
				"baseVolume": "1234.5678",
			},
		},
		"ts": int64(1704067200000),
	}

	data, _ := json.Marshal(mockData)
	worker.OnMessage(context.Background(), data)

	select {
	case receivedEvent := <-inbox:
		marketEvent, ok := receivedEvent.(*event.MarketUpdateEvent)
		if !ok {
			t.Fatalf("expected MarketUpdateEvent, got %T", receivedEvent)
		}
		if marketEvent.Symbol != "BTC" {
			t.Errorf("expected symbol BTC, got %s", marketEvent.Symbol)
		}
		if marketEvent.Exchange != "BITGET_SPOT" {
			t.Errorf("expected exchange BITGET_SPOT, got %s", marketEvent.Exchange)
		}
		if marketEvent.PriceMicros == 0 {
			t.Error("price should not be zero")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("no event received")
	}
}

func TestSpotWorker_IgnoreNonTicker(t *testing.T) {
	inbox := make(chan event.Event, 10)
	var seq uint64 = 0

	worker := &SpotWorker{
		symbols: map[string]string{"BTCUSDT": "BTC"},
		inbox:   inbox,
		seq:     &seq,
	}

	// Non-ticker message
	nonTicker := map[string]interface{}{
		"action": "snapshot",
		"arg": map[string]interface{}{
			"channel": "orderbook",
		},
	}
	data, _ := json.Marshal(nonTicker)
	worker.OnMessage(context.Background(), data)

	select {
	case <-inbox:
		t.Error("non-ticker message should be ignored")
	case <-time.After(50 * time.Millisecond):
		// Success
	}
}

func TestFuturesWorker_TickerParsing(t *testing.T) {
	inbox := make(chan event.Event, 10)
	var seq uint64 = 0

	// Map: symbol -> instId (BTC -> BTCUSDT)
	worker := &FuturesWorker{
		symbols: map[string]string{"BTC": "BTCUSDT"},
		inbox:   inbox,
		seq:     &seq,
	}

	// Mock Bitget futures ticker response - must match tickerResponse struct
	mockData := map[string]interface{}{
		"action": "snapshot",
		"arg": map[string]interface{}{
			"instType": "USDT-FUTURES",
			"channel":  "ticker",
			"instId":   "BTCUSDT",
		},
		"data": []interface{}{
			map[string]interface{}{
				"instId":    "BTCUSDT",
				"lastPr":    "92100.25",
				"volume24h": "5678.1234",
			},
		},
		"ts": int64(1704067200000),
	}

	data, _ := json.Marshal(mockData)
	worker.OnMessage(context.Background(), data)

	select {
	case receivedEvent := <-inbox:
		marketEvent, ok := receivedEvent.(*event.MarketUpdateEvent)
		if !ok {
			t.Fatalf("expected MarketUpdateEvent, got %T", receivedEvent)
		}
		if marketEvent.Symbol != "BTC" {
			t.Errorf("expected symbol BTC, got %s", marketEvent.Symbol)
		}
		if marketEvent.Exchange != "BITGET_FUTURES" {
			t.Errorf("expected exchange BITGET_FUTURES, got %s", marketEvent.Exchange)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("no event received")
	}
}

func TestFuturesWorker_IgnoreNonTicker(t *testing.T) {
	inbox := make(chan event.Event, 10)
	var seq uint64 = 0

	worker := &FuturesWorker{
		symbols: map[string]string{"BTCUSDT": "BTC"},
		inbox:   inbox,
		seq:     &seq,
	}

	nonTicker := map[string]interface{}{
		"action": "snapshot",
		"arg": map[string]interface{}{
			"channel": "positions",
		},
	}
	data, _ := json.Marshal(nonTicker)
	worker.OnMessage(context.Background(), data)

	select {
	case <-inbox:
		t.Error("non-ticker message should be ignored")
	case <-time.After(50 * time.Millisecond):
		// Success
	}
}
