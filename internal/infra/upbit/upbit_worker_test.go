package upbit

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

// createMockUpbitServer creates a mock Upbit WebSocket server
func createMockUpbitServer(t *testing.T, responses []interface{}) *httptest.Server {
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

		// Keep connection open briefly
		time.Sleep(100 * time.Millisecond)
	}))

	return server
}

func httpToWS(url string) string {
	return strings.Replace(url, "http://", "ws://", 1)
}

func TestUpbitWorker_TickerParsing(t *testing.T) {
	// Mock ticker response
	mockTicker := map[string]interface{}{
		"type":                 "ticker",
		"code":                 "KRW-BTC",
		"trade_price":          json.Number("50000000"),
		"acc_trade_volume_24h": json.Number("1234.56789"),
		"timestamp":            int64(1704067200000),
	}

	server := createMockUpbitServer(t, []interface{}{mockTicker})
	defer server.Close()

	inbox := make(chan event.Event, 10)
	var seq uint64 = 0

	// Create worker with mock URL
	worker := &Worker{
		symbols: []string{"BTC"},
		inbox:   inbox,
		seq:     &seq,
	}

	// Directly test OnMessage
	data, _ := json.Marshal(mockTicker)
	worker.OnMessage(context.Background(), data)

	// Verify event was emitted
	select {
	case receivedEvent := <-inbox:
		marketEvent, ok := receivedEvent.(*event.MarketUpdateEvent)
		if !ok {
			t.Fatalf("expected MarketUpdateEvent, got %T", receivedEvent)
		}
		if marketEvent.Symbol != "BTC" {
			t.Errorf("expected symbol BTC, got %s", marketEvent.Symbol)
		}
		if marketEvent.Exchange != "UPBIT" {
			t.Errorf("expected exchange UPBIT, got %s", marketEvent.Exchange)
		}
		if marketEvent.PriceMicros == 0 {
			t.Error("price should not be zero")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("no event received")
	}
}

func TestUpbitWorker_IgnoreNonTicker(t *testing.T) {
	inbox := make(chan event.Event, 10)
	var seq uint64 = 0

	worker := &Worker{
		symbols: []string{"BTC"},
		inbox:   inbox,
		seq:     &seq,
	}

	// Non-ticker message should be ignored
	nonTicker := map[string]interface{}{
		"type": "orderbook",
		"code": "KRW-BTC",
	}
	data, _ := json.Marshal(nonTicker)
	worker.OnMessage(context.Background(), data)

	// Should not emit any event
	select {
	case <-inbox:
		t.Error("non-ticker message should be ignored")
	case <-time.After(50 * time.Millisecond):
		// Success - no event emitted
	}
}

func TestUpbitWorker_SymbolExtraction(t *testing.T) {
	inbox := make(chan event.Event, 10)
	var seq uint64 = 0

	worker := &Worker{
		symbols: []string{"ETH", "BTC"},
		inbox:   inbox,
		seq:     &seq,
	}

	// Test ETH ticker
	ethTicker := map[string]interface{}{
		"type":                 "ticker",
		"code":                 "KRW-ETH",
		"trade_price":          json.Number("3000000"),
		"acc_trade_volume_24h": json.Number("100.0"),
		"timestamp":            int64(1704067200000),
	}
	data, _ := json.Marshal(ethTicker)
	worker.OnMessage(context.Background(), data)

	select {
	case receivedEvent := <-inbox:
		marketEvent := receivedEvent.(*event.MarketUpdateEvent)
		if marketEvent.Symbol != "ETH" {
			t.Errorf("expected symbol ETH, got %s", marketEvent.Symbol)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("no event received")
	}
}
