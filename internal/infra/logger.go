package infra

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// NewLogger creates a new slog.Logger with log rotation support
func NewLogger(cfg *Config) *slog.Logger {
	// Create logs directory if not exists
	logDir := filepath.Join(GetWorkspaceDir(), "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// Fallback to stderr if directory creation fails
		return slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}

	// Setup lumberjack logger for file rotation
	fileLogger := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "app.log"),
		MaxSize:    10,   // Megabytes
		MaxBackups: 3,    // Number of backups
		MaxAge:     28,   // Days
		Compress:   true, // Disabled by default
	}

	// Multi-writer: Log to both file and stdout
	writer := io.MultiWriter(os.Stdout, fileLogger)

	// Determine log level
	var level slog.Level
	switch cfg.Logging.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	return slog.New(slog.NewJSONHandler(writer, opts))
}
