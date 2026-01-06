package domain

import "crypto_go/pkg/quant"

// AlertConfig represents a price alert configuration
type AlertConfig struct {
	Symbol            string            `json:"symbol"`
	TargetPriceMicros quant.PriceMicros `json:"target"`
	Direction         string            `json:"direction"` // "UP" or "DOWN"
	Exchange          string            `json:"exchange"`  // "UPBIT", "BITGET_F"
	IsPersistent      bool              `json:"is_persistent"`
	active            bool
}

// NewAlertConfig creates a new alert configuration.
func NewAlertConfig(symbol string, targetPriceMicros, currentPriceMicros quant.PriceMicros, exchange string, isPersistent bool) *AlertConfig {
	direction := "UP"
	if targetPriceMicros < currentPriceMicros {
		direction = "DOWN"
	}
	return &AlertConfig{
		Symbol:            symbol,
		TargetPriceMicros: targetPriceMicros,
		Direction:         direction,
		Exchange:          exchange,
		IsPersistent:      isPersistent,
		active:            true,
	}
}

// IsActive returns whether the alert is active
func (a *AlertConfig) IsActive() bool {
	return a.active
}

// SetActive sets the alert's active state
func (a *AlertConfig) SetActive(active bool) {
	a.active = active
}

// CheckCondition checks if alert condition is met.
func (a *AlertConfig) CheckCondition(currentPriceMicros quant.PriceMicros) bool {
	if !a.active {
		return false
	}
	switch a.Direction {
	case "UP":
		return currentPriceMicros >= a.TargetPriceMicros
	case "DOWN":
		return currentPriceMicros <= a.TargetPriceMicros
	default:
		return false
	}
}
