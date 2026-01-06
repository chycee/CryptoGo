package domain

// CoinInfo represents metadata for a cryptocurrency
type CoinInfo struct {
	Symbol          string `json:"symbol"`
	Name            string `json:"name"`
	IconPath        string `json:"icon_path"`
	IsActive        bool   `json:"is_active"`        // Active trading status
	IsFavorite      bool   `json:"is_favorite"`      // User favorite status
	LastSyncedUnixM int64  `json:"last_synced_unix"` // Unix Micro
	CreatedAtUnixM  int64  `json:"created_at_unix"`
	UpdatedAtUnixM  int64  `json:"updated_at_unix"`
}

// AppConfig represents user-specific configuration (Key-Value)
type AppConfig struct {
	Key            string `json:"key"`
	Value          string `json:"value"`
	UpdatedAtUnixM int64  `json:"updated_at_unix"`
}
