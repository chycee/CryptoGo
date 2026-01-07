package bitget

import (
	"bytes"
	"context"
	"crypto_go/internal/domain"
	"crypto_go/internal/infra"
	"io"
	"net/http"
	"testing"
)

// MockRoundTripper allows us to mock HTTP responses
type MockRoundTripper struct {
	Func func(req *http.Request) (*http.Response, error)
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.Func(req)
}

func TestClient_PlaceOrder(t *testing.T) {
	// 1. Setup Mock Client
	cfg := &infra.Config{}
	cfg.API.Bitget.AccessKey = "test_access"
	cfg.API.Bitget.SecretKey = "test_secret"
	cfg.API.Bitget.Passphrase = "test_pass"

	client := NewClient(cfg, true)

	// Inject Mock Transport (White-box testing: accessing private field in same package)
	client.httpClient.Transport = &MockRoundTripper{
		Func: func(req *http.Request) (*http.Response, error) {
			// Validate URL and Method (FUTURES ENDPOINT)
			if req.URL.Path != "/api/v2/mix/order/place-order" {
				t.Errorf("Unexpected path: %s", req.URL.Path)
			}
			if req.Method != "POST" {
				t.Errorf("Unexpected method: %s", req.Method)
			}

			// Return Success JSON
			jsonResp := `{"code":"00000","msg":"success","data":{"clientOid":"test_oid"}}`
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
				Header:     make(http.Header),
			}, nil
		},
	}

	// 2. Execute
	order := domain.Order{
		ID:          "test_oid",
		Symbol:      "BTCUSDT",
		Side:        domain.SideBuy,
		Type:        domain.OrderTypeLimit,
		PriceMicros: 50_000_000_000, // $50,000
		QtySats:     100_000,        // 0.001 BTC
	}

	err := client.PlaceOrder(context.Background(), order)

	// 3. Verify
	if err != nil {
		t.Fatalf("PlaceOrder failed: %v", err)
	}
}

func TestClient_GetBalance_USDT(t *testing.T) {
	cfg := &infra.Config{}
	client := NewClient(cfg, true)

	client.httpClient.Transport = &MockRoundTripper{
		Func: func(req *http.Request) (*http.Response, error) {
			// Validate URL: /api/v2/mix/account/accounts
			if req.URL.Path != "/api/v2/mix/account/accounts" {
				t.Errorf("Unexpected path: %s", req.URL.Path)
			}

			// Return USDT Balance (Futures Structure)
			// [{"marginCoin":"USDT", "available":"100.500000"}]
			jsonResp := `{"code":"00000","msg":"success","data":[{"marginCoin":"USDT","available":"100.500000"}]}`
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
				Header:     make(http.Header),
			}, nil
		},
	}

	// Execute
	balance, err := client.GetBalance(context.Background(), "USDT")
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}

	// Verify (Int64 Only!)
	expected := int64(100_500_000)
	if balance != expected {
		t.Errorf("GetBalance mismatch. Got %d, Want %d", balance, expected)
	}
}

func TestClient_GetBalance_BTC(t *testing.T) {
	cfg := &infra.Config{}
	client := NewClient(cfg, true)

	client.httpClient.Transport = &MockRoundTripper{
		Func: func(req *http.Request) (*http.Response, error) {
			// Validate URL: /api/v2/mix/account/accounts
			if req.URL.Path != "/api/v2/mix/account/accounts" {
				t.Errorf("Unexpected path: %s", req.URL.Path)
			}

			// Return BTC Balance (Futures)
			// 1.23456789 BTC -> 123,456,789 Sats
			jsonResp := `{"code":"00000","msg":"success","data":[{"marginCoin":"BTC","available":"1.23456789"}]}`
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
				Header:     make(http.Header),
			}, nil
		},
	}

	// Execute
	balance, err := client.GetBalance(context.Background(), "BTC")
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}

	// Verify (Int64 Only: Sats for non-USDT)
	expected := int64(123_456_789)
	if balance != expected {
		t.Errorf("GetBalance mismatch. Got %d, Want %d", balance, expected)
	}
}
