package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"crypto_go/internal/event"
	"crypto_go/pkg/quant"
)

// rateAPIResponse represents the exchange rate API response.
// Provider can be swapped by changing the API URL and response parsing.
type rateAPIResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency           string      `json:"currency"`
				Symbol             string      `json:"symbol"`
				RegularMarketPrice json.Number `json:"regularMarketPrice"`
				PreviousClose      json.Number `json:"previousClose"`
			} `json:"meta"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// ExchangeRateClient fetches USD/KRW exchange rate from configured API.
type ExchangeRateClient struct {
	inbox        chan<- event.Event
	nextSeq      *uint64
	pollInterval time.Duration
	apiURL       string
	httpClient   *http.Client
	cancel       context.CancelFunc
}

// NewExchangeRateClient creates a new exchange rate client.
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

// NewExchangeRateClientWithConfig creates a client with custom configuration.
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

// Start begins polling for exchange rate updates.
func (c *ExchangeRateClient) Start(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)
	if err := c.fetchRate(ctx); err != nil {
		fmt.Printf("Initial exchange rate fetch failed: %v\n", err)
	}

	go func() {
		ticker := time.NewTicker(c.pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.fetchRate(ctx)
			}
		}
	}()
	return nil
}

// Stop cancels the polling.
func (c *ExchangeRateClient) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *ExchangeRateClient) fetchRate(ctx context.Context) error {
	for i := 0; i < 3; i++ {
		if i > 0 {
			time.Sleep(CalculateBackoff(i))
		}
		if err := c.doFetch(ctx); err == nil {
			return nil
		}
	}
	return fmt.Errorf("all fetch attempts failed")
}

func (c *ExchangeRateClient) doFetch(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", GetUserAgent())

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

	var data rateAPIResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	if data.Chart.Error != nil {
		return fmt.Errorf("rate API error: %s - %s", data.Chart.Error.Code, data.Chart.Error.Description)
	}

	if len(data.Chart.Result) == 0 {
		return fmt.Errorf("empty response from exchange rate API")
	}

	// Rule #1: No Float. Use string conversion via json.Number
	priceStr := data.Chart.Result[0].Meta.RegularMarketPrice.String()

	// Emit event using Pool (Rule #3: Zero-Alloc)
	ev := event.AcquireMarketUpdateEvent()
	ev.Seq = quant.NextSeq(c.nextSeq)
	ev.Ts = quant.TimeStamp(time.Now().UnixMicro())
	ev.Symbol = "USD/KRW"
	ev.PriceMicros = quant.ToPriceMicrosStr(priceStr)
	ev.QtySats = quant.QtyScale // 1.0 fixed as baseline for rate
	ev.Exchange = "FX"

	select {
	case c.inbox <- ev:
	default:
		event.ReleaseMarketUpdateEvent(ev)
	}

	return nil
}

// GetRate is no longer needed in the Gateway as it doesn't own the state.
