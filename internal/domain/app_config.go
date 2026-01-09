package domain

// AppConfig represents user-specific configuration (Key-Value).
type AppConfig struct {
	Key            string `json:"key"`
	Value          string `json:"value"`
	UpdatedAtUnixM int64  `json:"updated_at_unix,string"` // Rule #1: JSON string for int64
}
