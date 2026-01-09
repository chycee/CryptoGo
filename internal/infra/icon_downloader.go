package infra

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
)

// IconDownloader handles downloading and caching coin icons
type IconDownloader struct {
	basePath string
	client   *http.Client
}

// NewIconDownloader creates a new IconDownloader
func NewIconDownloader() (*IconDownloader, error) {
	path, err := getAssetsPath()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve assets path: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create assets directory: %w", err)
	}

	// Optimize HTTP Transport to prevent connection leaks
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100
	transport.MaxConnsPerHost = 10
	transport.IdleConnTimeout = 30 * time.Second

	return &IconDownloader{
		basePath: path,
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
	}, nil
}

// DownloadIcon downloads the icon for a symbol if it doesn't exist
// Returns the local file path on success
// Images are resized to 24x24 pixels for consistent UI display
func (d *IconDownloader) DownloadIcon(symbol string) (string, error) {
	// Security: Sanitize symbol to prevent path traversal
	safeSymbol := sanitizeSymbol(symbol)
	if safeSymbol == "" {
		return "", fmt.Errorf("invalid symbol: %s", symbol)
	}

	fileName := strings.ToLower(safeSymbol) + ".png"
	filePath := filepath.Join(d.basePath, fileName)

	// Check if exists
	if _, err := os.Stat(filePath); err == nil {
		return filePath, nil // Already exists (Cache Hit)
	}

	// Construct URL (Using Upbit CDN - best coverage for Korean exchanges)
	url := fmt.Sprintf("https://static.upbit.com/logos/%s.png", strings.ToUpper(symbol))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", GetUserAgent())

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	// Decode the image
	srcImg, err := imaging.Decode(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize to 24x24 with high-quality Lanczos filter
	resizedImg := imaging.Resize(srcImg, 24, 24, imaging.Lanczos)

	// Save the resized image
	if err := imaging.Save(resizedImg, filePath); err != nil {
		return "", fmt.Errorf("failed to save resized image: %w", err)
	}

	return filePath, nil
}

// GetIconPath returns the local path for a symbol's icon
func (d *IconDownloader) GetIconPath(symbol string) string {
	return filepath.Join(d.basePath, strings.ToLower(symbol)+".png")
}

func getAssetsPath() (string, error) {
	// Dynamically resolve base directory (Portable or OS-Standard)
	base := GetWorkspaceDir()
	return filepath.Join(base, "data", "icons"), nil
}

func sanitizeSymbol(symbol string) string {
	res := make([]rune, 0, len(symbol))
	for _, r := range symbol {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			res = append(res, r)
		}
	}
	return string(res)
}
