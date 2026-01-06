package domain

import (
	"testing"

	"crypto_go/pkg/quant"
)

func TestNewAlertConfig_Direction(t *testing.T) {
	t.Run("UP direction when target > current", func(t *testing.T) {
		alert := NewAlertConfig("BTC", 50000*quant.PriceScale, 45000*quant.PriceScale, "UPBIT", false)
		if alert.Direction != "UP" {
			t.Errorf("Expected UP, got %s", alert.Direction)
		}
	})

	t.Run("DOWN direction when target < current", func(t *testing.T) {
		alert := NewAlertConfig("BTC", 40000*quant.PriceScale, 45000*quant.PriceScale, "UPBIT", false)
		if alert.Direction != "DOWN" {
			t.Errorf("Expected DOWN, got %s", alert.Direction)
		}
	})

	t.Run("UP direction when target = current", func(t *testing.T) {
		alert := NewAlertConfig("BTC", 45000*quant.PriceScale, 45000*quant.PriceScale, "UPBIT", false)
		if alert.Direction != "UP" {
			t.Errorf("Expected UP for equal prices, got %s", alert.Direction)
		}
	})
}

func TestAlertConfig_CheckCondition(t *testing.T) {
	t.Run("UP alert triggers at target", func(t *testing.T) {
		alert := NewAlertConfig("BTC", 50000*quant.PriceScale, 45000*quant.PriceScale, "UPBIT", false)
		if !alert.CheckCondition(50000 * quant.PriceScale) {
			t.Error("Should trigger at target price")
		}
	})

	t.Run("UP alert triggers above target", func(t *testing.T) {
		alert := NewAlertConfig("BTC", 50000*quant.PriceScale, 45000*quant.PriceScale, "UPBIT", false)
		if !alert.CheckCondition(51000 * quant.PriceScale) {
			t.Error("Should trigger above target price")
		}
	})

	t.Run("UP alert does not trigger below target", func(t *testing.T) {
		alert := NewAlertConfig("BTC", 50000*quant.PriceScale, 45000*quant.PriceScale, "UPBIT", false)
		if alert.CheckCondition(49000 * quant.PriceScale) {
			t.Error("Should not trigger below target price")
		}
	})

	t.Run("DOWN alert triggers at target", func(t *testing.T) {
		alert := NewAlertConfig("BTC", 40000*quant.PriceScale, 45000*quant.PriceScale, "UPBIT", false)
		if !alert.CheckCondition(40000 * quant.PriceScale) {
			t.Error("Should trigger at target price")
		}
	})

	t.Run("Inactive alert does not trigger", func(t *testing.T) {
		alert := NewAlertConfig("BTC", 50000*quant.PriceScale, 45000*quant.PriceScale, "UPBIT", false)
		alert.SetActive(false)
		if alert.CheckCondition(55000 * quant.PriceScale) {
			t.Error("Inactive alert should not trigger")
		}
	})
}
