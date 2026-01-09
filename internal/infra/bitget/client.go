package bitget

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"crypto_go/internal/domain"
	"crypto_go/internal/infra"
)

// Bitget API Const// Base URLs
const (
	BaseURLMainnet = "https://api.bitget.com"
	// BaseURLTestnet is removed because Bitget V2 uses Mainnet URL + Header for Demo Trading
)

// Client for Bitget API (Spot & Futures/Mix)
type Client struct {
	httpClient     *http.Client
	baseURL        string
	signer         *Signer
	logger         *slog.Logger
	circuitBreaker *infra.CircuitBreaker // Rule #5: Fault isolation
	isTestnet      bool                  // Quant: Flag to enable "paptrading" header
}

// NewClient creates a new Bitget API client.
// isTestnet=true enables "paptrading: 1" header for Simulated Trading on Mainnet URL.
func NewClient(cfg *infra.Config, isTestnet bool) *Client {
	baseURL := BaseURLMainnet
	// If explicit URL provided in config, use it.
	if cfg.API.Bitget.RestURL != "" {
		baseURL = cfg.API.Bitget.RestURL
	}

	// Signer is required for V2
	signer := NewSigner(
		cfg.API.Bitget.AccessKey,
		cfg.API.Bitget.SecretKey,
		cfg.API.Bitget.Passphrase,
	)

	return &Client{
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		baseURL:        baseURL,
		signer:         signer,
		logger:         slog.With("module", "bitget_client"),
		circuitBreaker: infra.NewCircuitBreaker(infra.DefaultCircuitBreakerConfig("bitget-api")),
		isTestnet:      isTestnet,
	}
}

// Close wipes secrets from memory.
func (c *Client) Close() error {
	c.signer.Wipe()
	return nil
}

// PlaceOrderRequest - Internal Struct for JSON Marshaling
type placeOrderRequest struct {
	Symbol        string `json:"symbol"`
	ProductType   string `json:"productType"` // USDT-FUTURES
	MarginMode    string `json:"marginMode"`  // isolated, crossed
	MarginCoin    string `json:"marginCoin"`  // USDT
	Side          string `json:"side"`        // buy, sell
	TradeSide     string `json:"tradeSide"`   // open, close
	OrderType     string `json:"orderType"`
	Force         string `json:"force,omitempty"`
	Price         string `json:"price,omitempty"`
	Size          string `json:"size"`
	ClientOrderId string `json:"clientOid"`
}

// PlaceOrder sends an order to the exchange (FUTURES V2).
// Quant: Inputs are strictly int64 types.
func (c *Client) PlaceOrder(ctx context.Context, order domain.Order) error {
	// Rate Limiting: Prevent IP ban (보안 강화)
	infra.GetBitgetOrderLimiter().Wait()

	// 1. Boundary Conversion
	priceStr := fmt.Sprintf("%d.%06d", order.PriceMicros/1_000_000, order.PriceMicros%1_000_000)
	// V2 Futures Size: Usually in "Base Coin" or "Contracts"?
	// Bitget V2 Mix: "size" is amount in BASE coin for USDT-FUTURES (e.g. 0.1 BTC).
	// QtySats (8 decimals) -> String
	sizeStr := fmt.Sprintf("%d.%08d", order.QtySats/100_000_000, order.QtySats%100_000_000)

	side := "buy"
	if order.Side == domain.SideSell {
		side = "sell"
	}

	reqBody := placeOrderRequest{
		Symbol:      order.Symbol,
		ProductType: "USDT-FUTURES", // Hardcoded for now
		MarginMode:  "crossed",      // Default to Crossed
		MarginCoin:  "USDT",
		Side:        side,   // buy / sell
		TradeSide:   "open", // open / close
		OrderType:   "limit",
		// Force:         "normal",    // Removing entirely to rely on default
		Price:         priceStr,
		Size:          sizeStr,
		ClientOrderId: order.ID, // Restore mandatory field
	}

	if order.Type == domain.OrderTypeMarket {
		reqBody.OrderType = "market"
		reqBody.Price = ""
	}

	// 2. Send Request to MIX (Futures) Endpoint
	resp, err := c.doRequest(ctx, "POST", "/api/v2/mix/order/place-order", reqBody)
	if err != nil {
		return fmt.Errorf("bitget place order failed: %w", err)
	}
	defer resp.Body.Close()

	if _, err := c.parseResponse(resp); err != nil {
		return fmt.Errorf("place order error: %w", err)
	}

	c.logger.Info("Order Placed Successfully", "oid", order.ID, "symbol", order.Symbol)
	return nil
}

// CancelOrder sends a cancel request (FUTURES V2).
func (c *Client) CancelOrder(ctx context.Context, orderID string, symbol string) error {
	// Rate Limiting: Prevent IP ban (보안 강화)
	infra.GetBitgetOrderLimiter().Wait()

	reqBody := map[string]string{
		"symbol":      symbol,
		"productType": "USDT-FUTURES",
		"clientOid":   orderID,
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v2/mix/order/cancel-order", reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if _, err := c.parseResponse(resp); err != nil {
		return fmt.Errorf("cancel order error: %w", err)
	}

	c.logger.Info("Order Canceled Successfully", "oid", orderID, "symbol", symbol)
	return nil
}

// GetBalance fetches the available balance (FUTURES V2).
func (c *Client) GetBalance(ctx context.Context, coin string) (int64, error) {
	// Rate Limiting: Prevent IP ban (보안 강화)
	infra.GetBitgetAccountLimiter().Wait()

	// Path: /api/v2/mix/account/accounts?productType=USDT-FUTURES
	path := "/api/v2/mix/account/accounts?productType=USDT-FUTURES"

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	data, err := c.parseResponse(resp)
	if err != nil {
		return 0, fmt.Errorf("get balance error: %w", err)
	}

	// Bitget V2 Mix Account Structure
	var accounts []struct {
		MarginCoin string `json:"marginCoin"`
		Available  string `json:"available"`
	}

	if err := json.Unmarshal(data, &accounts); err != nil {
		return 0, fmt.Errorf("failed to parse accounts json: %w", err)
	}

	// Find the requested coin (marginCoin)
	for _, acc := range accounts {
		if acc.MarginCoin == coin {
			if coin == "USDT" {
				return ParseValueToMicros(acc.Available)
			}
			return ParseValueToSats(acc.Available)
		}
	}

	return 0, nil // Not found
}

// parseResponse handles standard Bitget API response validation and returns Raw Data
func (c *Client) parseResponse(resp *http.Response) (json.RawMessage, error) {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp struct {
		Code string          `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response json: %w", err)
	}

	if apiResp.Code != "00000" {
		return nil, fmt.Errorf("business error: code=%s msg=%s", apiResp.Code, apiResp.Msg)
	}

	return apiResp.Data, nil
}

// doRequest performs the HTTP request with circuit breaker protection.
func (c *Client) doRequest(ctx context.Context, method, path string, payload interface{}) (*http.Response, error) {
	// Circuit Breaker: Check if request is allowed (Rule #5: Fault isolation)
	if !c.circuitBreaker.Allow() {
		return nil, fmt.Errorf("circuit breaker open: bitget-api")
	}

	url := c.baseURL + path

	var body io.Reader
	var bodyStr string

	if payload != nil {
		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshaling payload: %w", err)
		}
		body = bytes.NewBuffer(jsonBytes)
		bodyStr = string(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// 1. Generate Auth Headers
	headers := c.signer.GenerateHeaders(method, path, "", bodyStr)
	for k, v := range headers {
		req.Header[k] = []string{v}
	}

	// 2. Add Simulation Header (Critical for Demo Keys)
	if c.isTestnet {
		req.Header["paptrading"] = []string{"1"}
	}

	// 3. Add Browser-like User-Agent
	req.Header.Set("User-Agent", infra.GetUserAgent())

	// 4. Execute with Circuit Breaker recording
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.circuitBreaker.RecordFailure()
		return nil, err
	}

	// Record success for successful HTTP response (even 4xx is "server responded")
	c.circuitBreaker.RecordSuccess()
	return resp, nil
}
