package domain

// CoinInfo represents metadata for a cryptocurrency.
// Fields ordered for cache-line friendliness (8-byte fields first).
type CoinInfo struct {
	LastSyncedUnixM int64  `json:"last_synced_unix,string"` // Unix Micro
	CreatedAtUnixM  int64  `json:"created_at_unix,string"`
	UpdatedAtUnixM  int64  `json:"updated_at_unix,string"`
	Symbol          string `json:"symbol"`
	Name            string `json:"name"`
	IconPath        string `json:"icon_path"`
	IsActive        bool   `json:"is_active"`   // Active trading status
	IsFavorite      bool   `json:"is_favorite"` // User favorite status
}
