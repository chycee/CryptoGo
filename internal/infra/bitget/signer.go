package bitget

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

// Signer handles Bitget V2 API Authentication.
// It stores keys as []byte to allow memory wiping (Security Rule #5).
type Signer struct {
	accessKey  []byte
	secretKey  []byte
	passphrase []byte
}

// NewSigner creates a new signer.
// It converts string inputs to []byte for internal safety.
func NewSigner(accessKey, secretKey, passphrase string) *Signer {
	return &Signer{
		accessKey:  []byte(accessKey),
		secretKey:  []byte(secretKey),
		passphrase: []byte(passphrase),
	}
}

// Wipe clears the keys from memory.
func (s *Signer) Wipe() {
	if s == nil {
		return
	}
	s.wipeSlice(s.accessKey)
	s.wipeSlice(s.secretKey)
	s.wipeSlice(s.passphrase)
}

func (s *Signer) wipeSlice(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// GenerateHeaders creates the required headers for Bitget V2 API.
func (s *Signer) GenerateHeaders(method, path, query, body string) map[string]string {
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())

	// Pre-signature string: timestamp + method + path + query + body
	// Note: query should be appended to path if not empty, typically caller handles full path?
	// Bitget: "timestamp + method + requestPath + ? + queryString + body"
	// Here assume path includes query if necessary.

	payload := timestamp + method + path + query + body
	signature := s.computeHmacSha256(payload)

	return map[string]string{
		"ACCESS-KEY":        string(s.accessKey),
		"ACCESS-SIGN":       signature,
		"ACCESS-TIMESTAMP":  timestamp,
		"ACCESS-PASSPHRASE": string(s.passphrase),
		"Content-Type":      "application/json",
		"locale":            "en-US",
	}
}

func (s *Signer) computeHmacSha256(payload string) string {
	// SecretKey is already []byte, perfect for HMAC
	mac := hmac.New(sha256.New, s.secretKey)
	mac.Write([]byte(payload))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
