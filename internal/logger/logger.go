package logger

import (
	"fmt"
	"log"
	"strings"
)

type LoggerInterface interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
)

type Logger struct {
	level string
}

var _ LoggerInterface = (*Logger)(nil)

func NewBaseLogger() *Logger {
	return &Logger{level: LevelError}
}

func NewLogger(level string) *Logger {
	normalizedLevel := strings.ToLower(level)

	supportedLevels := []string{LevelDebug, LevelInfo, LevelWarn, LevelError}
	isSupported := false
	for _, l := range supportedLevels {
		if normalizedLevel == l {
			isSupported = true
			break
		}
	}

	if !isSupported {
		log.Printf("[WARN] Уровень логирования %q не поддерживается. Используется 'info'.", level)
		normalizedLevel = LevelInfo
	}

	return &Logger{level: normalizedLevel}
}

func (l *Logger) shouldLog(level string) bool {
	switch l.level {
	case LevelDebug:
		return true
	case LevelInfo:
		return level != LevelDebug
	case LevelWarn:
		return level == LevelWarn || level == LevelError
	case LevelError:
		return level == LevelError
	default:
		return false
	}
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.shouldLog(LevelDebug) {
		fmt.Printf("[DEBUG] %s\n", fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) Info(msg string, args ...interface{}) {
	if l.shouldLog(LevelInfo) {
		fmt.Printf("[INFO] %s\n", fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) Warn(msg string, args ...interface{}) {
	if l.shouldLog(LevelWarn) {
		fmt.Printf("[WARN] %s\n", fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) Error(msg string, args ...interface{}) {
	if l.shouldLog(LevelError) {
		fmt.Printf("[ERROR] %s\n", fmt.Sprintf(msg, args...))
	}
}
