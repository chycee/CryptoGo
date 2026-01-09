package infra

import (
	"fmt"
	"os"
	"runtime"
	"sync"

	"gopkg.in/yaml.v3"
)

var (
	// currentUserAgent is protected by a mutex to allow dynamic synchronization from UI/WebView
	uaMu             sync.RWMutex
	currentUserAgent = GetPlatformUserAgent() // Initialize with OS-appropriate string
)

// GetUserAgent returns the current active User-Agent string. (Thread-safe)
func GetUserAgent() string {
	uaMu.RLock()
	defer uaMu.RUnlock()
	return currentUserAgent
}

// SetUserAgent updates the global User-Agent string. (Thread-safe)
// Used by GUI/Wails to sync the actual WebView User-Agent.
func SetUserAgent(ua string) {
	uaMu.Lock()
	defer uaMu.Unlock()
	currentUserAgent = ua
}

// GetPlatformUserAgent generates a browser-like User-Agent string based on current OS.
func GetPlatformUserAgent() string {
	chromeVer := "120.0.0.0" // Standard stable version
	os := runtime.GOOS
	arch := runtime.GOARCH

	switch os {
	case "windows":
		return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", chromeVer)
	case "linux":
		// Map arch to common Linux UA strings
		linuxArch := "x86_64"
		if arch == "arm64" {
			linuxArch = "aarch64"
		}
		return fmt.Sprintf("Mozilla/5.0 (X11; Linux %s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", linuxArch, chromeVer)
	case "darwin":
		return fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", chromeVer)
	default:
		// Fallback
		return "Mozilla/5.0 (compatible; Quant/1.0; +https://github.com/user/cryptoGo)"
	}
}

// Config는 애플리케이션의 모든 설정을 담습니다.
// LoadConfig로 로드된 후에 환경 변수를 통해 민감 내용을 덮어씁니다.
type Config struct {
	App struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"app"`

	Trading struct {
		Mode string `yaml:"mode"`
	} `yaml:"trading"`

	API struct {
		Upbit struct {
			WSURL     string   `yaml:"ws_url"`
			RestURL   string   `yaml:"rest_url"`
			AccessKey string   `yaml:"access_key"`
			SecretKey string   `yaml:"secret_key"`
			Symbols   []string `yaml:"symbols"`
		} `yaml:"upbit"`
		Bitget struct {
			WSURL      string            `yaml:"ws_url"`
			RestURL    string            `yaml:"rest_url"`
			AccessKey  string            `yaml:"access_key"`
			SecretKey  string            `yaml:"secret_key"`
			Passphrase string            `yaml:"passphrase"`
			Symbols    map[string]string `yaml:"symbols"`
		} `yaml:"bitget"`
		ExchangeRate struct {
			URL             string `yaml:"url"`
			PollIntervalSec int    `yaml:"poll_interval_sec"`
		} `yaml:"exchange_rate"`
	} `yaml:"api"`

	UI struct {
		UpdateIntervalMS int    `yaml:"update_interval_ms"`
		HistoryDays      int    `yaml:"history_days"`
		GapThreshold     int64  `yaml:"gap_threshold"` // Micros
		Theme            string `yaml:"theme"`
	} `yaml:"ui"`

	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`
}

// LoadConfig는 설정 파일을 읽고 파싱합니다.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// 4원칙: 보안 우선 - 환경 변수 오버라이드 지원
	overrideWithEnv(&cfg)

	// 5원칙: 설정 유효성 검사
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate checks configuration validity
func (c *Config) Validate() error {
	// Upbit
	if c.API.Upbit.WSURL == "" || (!hasPrefix(c.API.Upbit.WSURL, "ws://") && !hasPrefix(c.API.Upbit.WSURL, "wss://")) {
		return fmt.Errorf("invalid Upbit WS URL: %s", c.API.Upbit.WSURL)
	}
	if len(c.API.Upbit.Symbols) == 0 {
		return fmt.Errorf("at least one Upbit symbol is required")
	}

	// Bitget
	if c.API.Bitget.WSURL == "" || (!hasPrefix(c.API.Bitget.WSURL, "ws://") && !hasPrefix(c.API.Bitget.WSURL, "wss://")) {
		return fmt.Errorf("invalid Bitget WS URL: %s", c.API.Bitget.WSURL)
	}

	// UI
	if c.UI.UpdateIntervalMS <= 0 {
		return fmt.Errorf("update interval must be positive")
	}

	return nil
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

// overrideWithEnv는 환경 변수가 존재할 경우 설정 값을 덮어씁니다.
// Rule #5: 환경 변수는 설정 파일보다 우선합니다 (보안 강화).
func overrideWithEnv(cfg *Config) {
	// Security Warning: Log if secrets found in config file
	if cfg.API.Bitget.SecretKey != "" || cfg.API.Upbit.SecretKey != "" {
		// Using fmt instead of slog to avoid import cycle
		fmt.Println("⚠️  SECURITY WARNING: API secrets found in config file.")
		fmt.Println("   Recommendation: Use environment variables instead:")
		fmt.Println("   - CRYPTO_BITGET_KEY, CRYPTO_BITGET_SECRET, CRYPTO_BITGET_PASSPHRASE")
		fmt.Println("   - CRYPTO_UPBIT_KEY, CRYPTO_UPBIT_SECRET")
	}

	if key := os.Getenv("CRYPTO_UPBIT_KEY"); key != "" {
		cfg.API.Upbit.AccessKey = key
	}
	if secret := os.Getenv("CRYPTO_UPBIT_SECRET"); secret != "" {
		cfg.API.Upbit.SecretKey = secret
	}
	if key := os.Getenv("CRYPTO_BITGET_KEY"); key != "" {
		cfg.API.Bitget.AccessKey = key
	}
	if secret := os.Getenv("CRYPTO_BITGET_SECRET"); secret != "" {
		cfg.API.Bitget.SecretKey = secret
	}
	if pass := os.Getenv("CRYPTO_BITGET_PASSPHRASE"); pass != "" {
		cfg.API.Bitget.Passphrase = pass
	}
}
