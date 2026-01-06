package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"crypto_go/internal/event"
	"crypto_go/pkg/quant"
)

// yahooChartResponse represents the Yahoo Finance Chart API response
type yahooChartResponse struct {
	Chart struct {
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
	} `json:"chart"`
}

// ExchangeRateClient fetches USD/KRW exchange rate from Yahoo Finance API
type ExchangeRateClient struct {
	inbox        chan<- event.Event
	nextSeq      *uint64 // Pointer to a shared/global atomic or managed sequence
	pollInterval time.Duration
	apiURL       string
	httpClient   *http.Client
	cancel       context.CancelFunc
}

// NewExchangeRateClient creates a new exchange rate client
func NewExchangeRateClient(inbox chan<- event.Event, seq *uint64) *ExchangeRateClient {
	return &ExchangeRateClient{
		inbox:        inbox,
		nextSeq:      seq,
		pollInterval: 60 * time.Second,
		apiURL:       "https://query1.finance.yahoo.com/v8/finance/chart/KRW=X",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NewExchangeRateClientWithConfig creates a client with custom configuration
func NewExchangeRateClientWithConfig(inbox chan<- event.Event, seq *uint64, apiURL string, pollIntervalSec int) *ExchangeRateClient {
	client := NewExchangeRateClient(inbox, seq)
	if apiURL != "" {
		client.apiURL = apiURL
	}
	if pollIntervalSec > 0 {
		client.pollInterval = time.Duration(pollIntervalSec) * time.Second
	}
	return client
}

// Start begins polling for exchange rate updates
func (c *ExchangeRateClient) Start(ctx context.Context) error {
	// Create a cancellable context
	ctx, c.cancel = context.WithCancel(ctx)

	// Fetch immediately on start
	if err := c.fetchRate(ctx); err != nil {
		slog.Warn("Initial exchange rate fetch failed", slog.Any("error", err))
		// Continue anyway - will retry on next tick
	}

	// Start polling goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Exchange rate polling panic recovered", slog.Any("panic", r))
			}
		}()

		ticker := time.NewTicker(c.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("Exchange rate polling stopped")
				return
			case <-ticker.C:
				if err := c.fetchRate(ctx); err != nil {
					slog.Warn("Exchange rate fetch failed", slog.Any("error", err))
				}
			}
		}
	}()

	return nil
}

// Stop cancels the polling context.
func (c *ExchangeRateClient) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

// fetchRate fetches the current exchange rate from Yahoo Finance API with retry logic
func (c *ExchangeRateClient) fetchRate(ctx context.Context) error {
	var lastErr error
	for i := 0; i < 3; i++ {
		if i > 0 {
			// Exponential backoff: 1s, 2s, 4s
			delay := time.Duration(1<<uint(i-1)) * time.Second
			slog.Info("Retrying exchange rate fetch", slog.Int("attempt", i), slog.Duration("delay", delay))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := c.doFetch(ctx)
		if err == nil {
			return nil
		}
		lastErr = err
		slog.Warn("Exchange rate fetch attempt failed", slog.Int("attempt", i+1), slog.Any("error", err))
	}
	return lastErr
}

func (c *ExchangeRateClient) doFetch(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL, nil)
	if err != nil {
		return err
	}

	// Add browser-like User-Agent to avoid bot detection
	req.Header.Set("User-Agent", DefaultUserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var data yahooChartResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	// Check for API error
	if data.Chart.Error != nil {
		return fmt.Errorf("Yahoo API error: %s - %s", data.Chart.Error.Code, data.Chart.Error.Description)
	}

	if len(data.Chart.Result) == 0 {
		return fmt.Errorf("empty response from Yahoo Finance API")
	}

	// Use regularMarketPrice as the exchange rate (USD/KRW)
	price := quant.ToPriceMicros(data.Chart.Result[0].Meta.RegularMarketPrice)

	// Emit event to sequencer
	c.inbox <- &event.MarketUpdateEvent{
		BaseEvent: event.BaseEvent{
			Seq: quant.NextSeq(c.nextSeq),
			Ts:  quant.TimeStamp(time.Now().UnixMicro()),
		},
		Symbol:      "USD/KRW",
		PriceMicros: price,
		QtySats:     quant.QtyScale, // 1.0 fixed for rate
		Exchange:    "YAHOO",
	}

	return nil
}

// GetRate is no longer needed in the Gateway as it doesn't own the state.
