package infra

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// mockHandler implements WebSocketHandler for testing
type mockHandler struct {
	url            string
	onConnectCalls int32
	onMessageCalls int32
	messages       [][]byte
}

func (m *mockHandler) GetURL() string { return m.url }
func (m *mockHandler) ID() string     { return "MOCK" }
func (m *mockHandler) OnConnect(ctx context.Context, conn *websocket.Conn) error {
	atomic.AddInt32(&m.onConnectCalls, 1)
	return nil
}
func (m *mockHandler) OnMessage(ctx context.Context, msg []byte) {
	atomic.AddInt32(&m.onMessageCalls, 1)
	m.messages = append(m.messages, msg)
}
func (m *mockHandler) OnPing(ctx context.Context, conn *websocket.Conn) error {
	return nil
}

// createMockWSServer creates a test WebSocket server
func createMockWSServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
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
		handler(conn)
	}))

	return server
}

// httpToWS converts http:// URL to ws://
func httpToWS(url string) string {
	return strings.Replace(url, "http://", "ws://", 1)
}

func TestBaseWSWorker_Connect(t *testing.T) {
	// Create mock server that sends one message
	server := createMockWSServer(t, func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"test"}`))
		time.Sleep(100 * time.Millisecond)
	})
	defer server.Close()

	handler := &mockHandler{url: httpToWS(server.URL)}
	worker := NewBaseWSWorker(handler)
	worker.ReadTimeout = 500 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	worker.Start(ctx)
	time.Sleep(200 * time.Millisecond) // Give time for connection and message

	worker.Stop()

	if atomic.LoadInt32(&handler.onConnectCalls) == 0 {
		t.Error("OnConnect was not called")
	}
	if atomic.LoadInt32(&handler.onMessageCalls) == 0 {
		t.Error("OnMessage was not called")
	}
}

func TestBaseWSWorker_GracefulShutdown(t *testing.T) {
	// Create mock server that stays open
	serverClosed := make(chan struct{})
	server := createMockWSServer(t, func(conn *websocket.Conn) {
		<-serverClosed
	})
	defer server.Close()
	defer close(serverClosed)

	handler := &mockHandler{url: httpToWS(server.URL)}
	worker := NewBaseWSWorker(handler)

	ctx := context.Background()
	worker.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Stop should not hang
	done := make(chan struct{})
	go func() {
		worker.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success - Stop returned
	case <-time.After(2 * time.Second):
		t.Error("Stop did not return within timeout")
	}
}

func TestBaseWSWorker_Write(t *testing.T) {
	receivedMsg := make(chan []byte, 1)

	server := createMockWSServer(t, func(conn *websocket.Conn) {
		_, msg, err := conn.ReadMessage()
		if err == nil {
			receivedMsg <- msg
		}
		time.Sleep(100 * time.Millisecond)
	})
	defer server.Close()

	handler := &mockHandler{url: httpToWS(server.URL)}
	worker := NewBaseWSWorker(handler)

	ctx := context.Background()
	worker.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Write a message
	testMsg := []byte(`{"action":"subscribe"}`)
	err := worker.Write(websocket.TextMessage, testMsg)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}

	// Verify server received it
	select {
	case msg := <-receivedMsg:
		if string(msg) != string(testMsg) {
			t.Errorf("expected %s, got %s", testMsg, msg)
		}
	case <-time.After(1 * time.Second):
		t.Error("server did not receive message")
	}

	worker.Stop()
}
