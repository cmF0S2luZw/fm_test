package logger

import (
	"log/slog"
	"os"
	"strings"
)

const (
	DebugLevel = "debug"
	InfoLevel  = "info"
	ErrorLevel = "error"
)

func NewLogger(level string) *slog.Logger {
	var logLevel slog.Level

	switch strings.ToLower(level) {
	case DebugLevel:
		logLevel = slog.LevelDebug
	case InfoLevel:
		logLevel = slog.LevelInfo
	case ErrorLevel:
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: logLevel == slog.LevelDebug,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)

	return slog.New(handler)
}

func SetupGlobalLogger(level string) {
	slog.SetDefault(NewLogger(level))
}
