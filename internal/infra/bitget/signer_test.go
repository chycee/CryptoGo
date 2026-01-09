package bitget

import (
	"testing"
)

func TestSigner_GenerateSignature(t *testing.T) {
	// Standard HMAC Validation requires direct access to logic or predictable output.
	// Since GenerateHeaders relies on time.Now(), we verify the logic indirectly
	// or trusting the unit test of computeHmacSha256 below.

	// Test actual Signer struct
	signer := NewSigner("key", "secret", "pass")

	headers := signer.GenerateHeaders("POST", "/api/v2/order", "", "{\"symbol\":\"BTCUSDT\"}")

	if headers["ACCESS-KEY"] != "key" {
		t.Errorf("Expected ACCESS-KEY to be 'key', got %s", headers["ACCESS-KEY"])
	}
	if headers["ACCESS-PASSPHRASE"] != "pass" {
		t.Errorf("Expected ACCESS-PASSPHRASE to be 'pass', got %s", headers["ACCESS-PASSPHRASE"])
	}
	if headers["ACCESS-SIGN"] == "" {
		t.Error("ACCESS-SIGN should not be empty")
	}
	if len(headers["ACCESS-TIMESTAMP"]) != 13 { // Milliseconds
		t.Errorf("Expected timestamp len 13, got %s", headers["ACCESS-TIMESTAMP"])
	}
}

func TestComputeHmacSha256(t *testing.T) {
	// Standard HMAC-SHA256 Test Vector
	key := "key"
	data := "The quick brown fox jumps over the lazy dog"
	// Expected Base64: 97yD9DBThCSxMpjmqm+xQ+9NWaFJRhdZl0edvC0aPNg=

	expected := "97yD9DBThCSxMpjmqm+xQ+9NWaFJRhdZl0edvC0aPNg="

	// Create a signer initialized with the test key
	signer := NewSigner("dummy_access", key, "dummy_pass")

	// Call the private method (allowed since we are in package bitget)
	result := signer.computeHmacSha256(data)

	if result != expected {
		t.Errorf("HMAC Mismatch. Expected %s, got %s", expected, result)
	}
}
