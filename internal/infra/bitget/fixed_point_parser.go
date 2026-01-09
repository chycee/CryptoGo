package bitget

import (
	"errors"
	"strconv"
	"strings"
)

// ParseValueToMicros parses a string decimal (e.g. "123.45") to int64 micros (10^-6).
// It avoids float64 entirely for safety.
func ParseValueToMicros(s string) (int64, error) {
	return parseFixedPoint(s, 6)
}

// ParseValueToSats parses a string decimal (e.g. "0.00123") to int64 sats (10^-8).
// It avoids float64 entirely for safety.
func ParseValueToSats(s string) (int64, error) {
	return parseFixedPoint(s, 8)
}

// parseFixedPoint parses a string representation of a decimal into an integer
// scaled by 10^decimals.
// Example: "1.23", decimals=6 -> 1230000
func parseFixedPoint(s string, decimals int) (int64, error) {
	if s == "" {
		return 0, nil
	}

	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return 0, errors.New("invalid decimal format: multiple dots")
	}

	integerPart := parts[0]
	fractionalPart := ""
	if len(parts) == 2 {
		fractionalPart = parts[1]
	}

	// Sign handling
	sign := int64(1)
	if strings.HasPrefix(integerPart, "-") {
		sign = -1
		integerPart = integerPart[1:]
	}

	// Parse Integer Part
	intVal, err := strconv.ParseInt(integerPart, 10, 64)
	if err != nil {
		if integerPart == "" {
			intVal = 0 // ".5" case
		} else {
			return 0, err
		}
	}

	// Pad or Truncate Fractional Part
	// needed length = decimals
	if len(fractionalPart) > decimals {
		// Truncate extra precision (floor)
		fractionalPart = fractionalPart[:decimals]
	} else {
		// Pad with zeros
		padding := decimals - len(fractionalPart)
		fractionalPart = fractionalPart + strings.Repeat("0", padding)
	}

	fracVal, err := strconv.ParseInt(fractionalPart, 10, 64)
	if err != nil {
		return 0, err
	}

	// Combine: intVal * 10^decimals + fracVal
	// We construct the multiplier manually or using loop to avoid float math (math.Pow)
	multiplier := int64(1)
	for i := 0; i < decimals; i++ {
		multiplier *= 10
	}

	total := intVal*multiplier + fracVal
	return total * sign, nil
}
