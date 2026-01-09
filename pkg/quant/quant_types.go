package quant

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync/atomic"
)

// PriceMicros represents price multiplied by 1,000,000 (10^6).
// E.g., 1.23 USD = 1,230,000 PriceMicros.
type PriceMicros int64

// QtySats represents quantity multiplied by 100,000,000 (10^8).
// E.g., 1.0 BTC = 100,000,000 QtySats.
type QtySats int64

// TimeStamp represents Unix Microseconds.
type TimeStamp int64

const (
	PriceScale = 1000000
	QtyScale   = 100000000
)

// ToPriceMicros converts a float64 (from external API) to PriceMicros.
// Note: Only used at the boundary. Internal logic uses PriceMicros directly.
func ToPriceMicros(f float64) PriceMicros {
	return PriceMicros(math.Round(f * PriceScale))
}

// ToQtySats converts a float64 to QtySats.
func ToQtySats(f float64) QtySats {
	return QtySats(math.Round(f * QtyScale))
}

func (p PriceMicros) String() string {
	return fmt.Sprintf("%.6f", float64(p)/PriceScale)
}

func (q QtySats) String() string {
	return fmt.Sprintf("%.8f", float64(q)/QtyScale)
}

// NextSeq generates the next sequence number atomically.
func NextSeq(ptr *uint64) uint64 {
	return atomic.AddUint64(ptr, 1)
}

// ParseTimeStamp converts a string (ms) or int64 to TimeStamp (micros).
func ParseTimeStamp(s string) (TimeStamp, error) {
	ms, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return TimeStamp(ms * 1000), nil
}

// ToPriceMicrosStr converts a numeric string to PriceMicros without using float64.
// Rule #1: No Float. Using fixed-point string parsing.
func ToPriceMicrosStr(s string) PriceMicros {
	return PriceMicros(parseFixedPoint(s, 6))
}

// ToQtySatsStr converts a numeric string to QtySats without using float64.
func ToQtySatsStr(s string) QtySats {
	return QtySats(parseFixedPoint(s, 8))
}

// parseFixedPoint parses a numeric string into an int64 with the given precision.
// E.g., parseFixedPoint("1.23", 6) -> 1,230,000.
func parseFixedPoint(s string, precision int) int64 {
	if s == "" || s == "null" {
		return 0
	}

	parts := []string{s}
	if dotIdx := -1; true {
		for i := 0; i < len(s); i++ {
			if s[i] == '.' {
				dotIdx = i
				break
			}
		}
		if dotIdx != -1 {
			parts = []string{s[:dotIdx], s[dotIdx+1:]}
		}
	}

	// 1. Parse Integer Part
	intPart, _ := strconv.ParseInt(parts[0], 10, 64)
	for i := 0; i < precision; i++ {
		intPart *= 10
	}

	if len(parts) < 2 {
		return intPart
	}

	// 2. Parse Fraction Part
	fracStr := parts[1]
	if len(fracStr) > precision {
		fracStr = fracStr[:precision]
	}
	fracPart, _ := strconv.ParseInt(fracStr, 10, 64)

	// Pad fraction part with zeros if shorter than precision
	for i := len(fracStr); i < precision; i++ {
		fracPart *= 10
	}

	// 3. Handle Negative
	if strings.HasPrefix(parts[0], "-") {
		return intPart - fracPart
	}
	return intPart + fracPart
}
