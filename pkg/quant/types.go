package quant

import (
	"fmt"
	"math"
	"strconv"
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

// ToPriceMicrosStr converts a numeric string to PriceMicros.
func ToPriceMicrosStr(s string) PriceMicros {
	f, _ := strconv.ParseFloat(s, 64)
	return ToPriceMicros(f)
}

// ToQtySatsStr converts a numeric string to QtySats.
func ToQtySatsStr(s string) QtySats {
	f, _ := strconv.ParseFloat(s, 64)
	return ToQtySats(f)
}
