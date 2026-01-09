package infra

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	AppName = "crypto-go"
)

// GetWorkspaceDir returns the root directory for all runtime data.
// It prioritizes a local "_workspace" directory if it exists (Portable/Dev mode).
// Otherwise, it returns the OS-standard data directory.
func GetWorkspaceDir() string {
	// 1. Check for local workspace (Priority 1: Portable/Dev)
	localDir := "_workspace"
	if _, err := os.Stat(localDir); err == nil {
		return localDir
	}

	// 2. Identify OS Standard Data Dir (Priority 2: Production)
	var baseDir string
	switch runtime.GOOS {
	case "windows":
		// Windows: %AppData%\crypto-go
		baseDir = os.Getenv("APPDATA")
		if baseDir == "" {
			baseDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
	case "darwin":
		// macOS: ~/Library/Application Support/crypto-go
		home, _ := os.UserHomeDir()
		baseDir = filepath.Join(home, "Library", "Application Support")
	case "linux":
		// Linux: ~/.local/share/crypto-go (XDG_DATA_HOME)
		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome != "" {
			baseDir = dataHome
		} else {
			home, _ := os.UserHomeDir()
			baseDir = filepath.Join(home, ".local", "share")
		}
	default:
		// Fallback to local
		return localDir
	}

	res := filepath.Join(baseDir, AppName)
	return res
}

// EnsureDir creates the directory if it doesn't exist with safe permissions (0755).
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// CreateLockFile attempts to create and lock a file to prevent multiple instances.
// It returns a closer function and an error if another instance is already running.
func CreateLockFile(workDir string) (func(), error) {
	lockPath := filepath.Join(workDir, "instance.lock")

	// On Windows, os.OpenFile with specific flags can act as a lock.
	// On Linux, we just check if it exists or use more complex flock (omitted for pure Go simplicity).
	// A simpler "Fail Fast" approach: Try to create, if exists, fail.

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("another instance is already running (lock file exists: %s)", lockPath)
		}
		return nil, err
	}

	// Write current PID for debugging
	f.WriteString(fmt.Sprintf("%d", os.Getpid()))
	f.Close()

	closer := func() {
		os.Remove(lockPath)
	}

	return closer, nil
}

// ResolveConfigPath attempts to find the config.yaml.
// Priority: 1. Current Dir, 2. OS Config Dir
func ResolveConfigPath() string {
	defaultPath := filepath.Join("configs", "config.yaml")

	// 1. Current working directory (standard)
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath
	}

	// 2. OS Standard Config Dir
	configRoot, err := os.UserConfigDir()
	if err == nil {
		osPath := filepath.Join(configRoot, AppName, "config.yaml")
		if _, err := os.Stat(osPath); err == nil {
			return osPath
		}
	}

	// Return default and let LoadConfig handle the "file not found" error if it's really missing
	return defaultPath
}
