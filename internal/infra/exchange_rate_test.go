package infra

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"crypto_go/internal/event"
	"crypto_go/pkg/quant"
)

// Helper function to create Yahoo Finance mock response
func createYahooMockResponse(price float64) yahooChartResponse {
	return yahooChartResponse{
		Chart: struct {
			Result []struct {
				Meta struct {
					Currency           string  `json:"currency"`
					Symbol             string  `json:"symbol"`
					RegularMarketPrice float64 `json:"regularMarketPrice"`
					PreviousClose      float64 `json:"previousClose"`
				} `json:"meta"`
			} `json:"result"`
			Error *struct {
				Code        string `json:"code"`
				Description string `json:"description"`
			} `json:"error"`
		}{
			Result: []struct {
				Meta struct {
					Currency           string  `json:"currency"`
					Symbol             string  `json:"symbol"`
					RegularMarketPrice float64 `json:"regularMarketPrice"`
					PreviousClose      float64 `json:"previousClose"`
				} `json:"meta"`
			}{
				{
					Meta: struct {
						Currency           string  `json:"currency"`
						Symbol             string  `json:"symbol"`
						RegularMarketPrice float64 `json:"regularMarketPrice"`
						PreviousClose      float64 `json:"previousClose"`
					}{
						Currency:           "KRW",
						Symbol:             "KRW=X",
						RegularMarketPrice: price,
						PreviousClose:      price - 1.0,
					},
				},
			},
			Error: nil,
		},
	}
}

func TestExchangeRateClient_FetchRate(t *testing.T) {
	// Create mock server with Yahoo Finance response
	mockResp := createYahooMockResponse(1380.50)
	mockBody, _ := json.Marshal(mockResp)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(mockBody)
	}))
	defer server.Close()

	// Create client with mock server
	inbox := make(chan event.Event, 1)
	nextSeq := uint64(1)
	client := NewExchangeRateClientWithConfig(inbox, &nextSeq, server.URL, 1)

	// Fetch rate
	ctx := context.Background()
	err := client.fetchRate(ctx)
	if err != nil {
		t.Fatalf("fetchRate failed: %v", err)
	}

	// Verify event in inbox
	select {
	case ev := <-inbox:
		m, ok := ev.(*event.MarketUpdateEvent)
		if !ok {
			t.Fatalf("Expected MarketUpdateEvent, got %T", ev)
		}
		expectedPrice := quant.ToPriceMicros(1380.50)
		if m.PriceMicros != expectedPrice {
			t.Errorf("Expected price %d, got %d", expectedPrice, m.PriceMicros)
		}
		if m.Symbol != "USD/KRW" {
			t.Errorf("Expected symbol USD/KRW, got %s", m.Symbol)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for event in inbox")
	}
}

func TestExchangeRateClient_StartStop(t *testing.T) {
	// Create mock server
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		mockResp := createYahooMockResponse(1380.50)
		body, _ := json.Marshal(mockResp)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}))
	defer server.Close()

	inbox := make(chan event.Event, 10)
	nextSeq := uint64(1)
	client := NewExchangeRateClientWithConfig(inbox, &nextSeq, server.URL, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := client.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for initial fetch
	time.Sleep(100 * time.Millisecond)

	if callCount < 1 {
		t.Error("Expected at least one API call")
	}

	// Stop should complete without hanging
	client.Stop()
}

func TestExchangeRateClient_EmptyResponse(t *testing.T) {
	// Yahoo Finance returns empty result array
	emptyResp := yahooChartResponse{}
	emptyResp.Chart.Result = nil
	mockBody, _ := json.Marshal(emptyResp)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(mockBody)
	}))
	defer server.Close()

	inbox := make(chan event.Event, 1)
	nextSeq := uint64(1)
	client := NewExchangeRateClientWithConfig(inbox, &nextSeq, server.URL, 1)

	err := client.fetchRate(context.Background())
	if err == nil {
		t.Error("Empty response should return error")
	}
}

func TestExchangeRateClient_RetryOnFailure(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		mockResp := createYahooMockResponse(1380.50)
		body, _ := json.Marshal(mockResp)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}))
	defer server.Close()

	inbox := make(chan event.Event, 5)
	nextSeq := uint64(1)
	client := NewExchangeRateClientWithConfig(inbox, &nextSeq, server.URL, 1)

	// Fetch rate (should retry 2 times and succeed on 3rd)
	err := client.fetchRate(context.Background())
	if err != nil {
		t.Fatalf("fetchRate should succeed after retries: %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}
