package domain

import (
	"errors"
	"testing"
)

func TestNetworkError(t *testing.T) {
	baseErr := errors.New("connection refused")

	t.Run("retriable error", func(t *testing.T) {
		err := NewNetworkError("connect", baseErr)

		if !err.IsRetriable() {
			t.Error("Expected error to be retriable")
		}

		if err.Error() != "connect: connection refused" {
			t.Errorf("Error message = %q, want %q", err.Error(), "connect: connection refused")
		}

		if !errors.Is(err, baseErr) {
			t.Error("Expected error to wrap baseErr")
		}
	})

	t.Run("fatal error", func(t *testing.T) {
		err := NewFatalNetworkError("auth", baseErr)

		if err.IsRetriable() {
			t.Error("Expected error to not be retriable")
		}
	})

	t.Run("IsRetriable helper", func(t *testing.T) {
		retriable := NewNetworkError("dial", baseErr)
		fatal := NewFatalNetworkError("auth", baseErr)
		plain := errors.New("plain error")

		if !IsRetriable(retriable) {
			t.Error("IsRetriable should return true for retriable error")
		}

		if IsRetriable(fatal) {
			t.Error("IsRetriable should return false for fatal error")
		}

		if IsRetriable(plain) {
			t.Error("IsRetriable should return false for plain error")
		}
	})
}

func TestConfigError(t *testing.T) {
	baseErr := errors.New("missing value")
	err := &ConfigError{Field: "api_key", Err: baseErr}

	if err.IsRetriable() {
		t.Error("ConfigError should never be retriable")
	}

	expected := "config error [api_key]: missing value"
	if err.Error() != expected {
		t.Errorf("Error message = %q, want %q", err.Error(), expected)
	}
}
