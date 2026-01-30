// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package logger provides structured logging with configurable levels and file rotation.
//
// It uses Go's standard log/slog package and supports:
// - Multiple log levels (DEBUG, INFO, WARN, ERROR)
// - Dual output (stdout + file)
// - Daily log rotation with size and age retention
// - Runtime level changes
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// Level represents a log level.
type Level = slog.Level

// Log levels matching slog constants.
const (
	LevelDebug = slog.LevelDebug // -4
	LevelInfo  = slog.LevelInfo  // 0
	LevelWarn  = slog.LevelWarn  // 4
	LevelError = slog.LevelError // 8
)

// Default configuration values.
const (
	DefaultLevel         = LevelWarn
	DefaultRetentionDays = 30
	DefaultMaxSizeMB     = 200
)

// Logger wraps slog.Logger with level control and rotation support.
type Logger struct {
	slog    *slog.Logger
	level   *slog.LevelVar
	rotator *RotatingWriter
	mu      sync.RWMutex
}

// Config holds logger configuration.
type Config struct {
	// Level is the minimum log level (default: WARN)
	Level Level

	// LogDir is the directory for log files (empty = stdout only)
	LogDir string

	// RetentionDays is how many days to keep old logs (default: 30)
	RetentionDays int

	// MaxSizeMB is the maximum total size in MB before cleanup (default: 200)
	MaxSizeMB int

	// Prefix for log filenames (default: "anemone")
	Prefix string
}

var (
	// global is the default logger instance
	global *Logger
	once   sync.Once
)

// Init initializes the global logger with the given configuration.
// Safe to call multiple times; subsequent calls update the configuration.
func Init(cfg *Config) error {
	if cfg == nil {
		cfg = &Config{}
	}

	// Apply defaults
	if cfg.RetentionDays == 0 {
		cfg.RetentionDays = DefaultRetentionDays
	}
	if cfg.MaxSizeMB == 0 {
		cfg.MaxSizeMB = DefaultMaxSizeMB
	}
	if cfg.Prefix == "" {
		cfg.Prefix = "anemone"
	}

	level := &slog.LevelVar{}
	level.Set(cfg.Level)

	var writers []io.Writer
	var rotator *RotatingWriter

	// Always write to stdout
	writers = append(writers, os.Stdout)

	// Add file writer if LogDir is specified
	if cfg.LogDir != "" {
		var err error
		rotator, err = NewRotatingWriter(cfg.LogDir, cfg.Prefix, cfg.RetentionDays, cfg.MaxSizeMB)
		if err != nil {
			return err
		}
		writers = append(writers, rotator)
	}

	// Create multi-writer
	w := io.MultiWriter(writers...)

	// Create handler with human-readable format
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize time format
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.Time().Format("2006-01-02 15:04:05")),
				}
			}
			// Uppercase level names
			if a.Key == slog.LevelKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.String()),
				}
			}
			return a
		},
	})

	global = &Logger{
		slog:    slog.New(handler),
		level:   level,
		rotator: rotator,
	}

	// Also set as default slog logger for compatibility
	slog.SetDefault(global.slog)

	return nil
}

// Close closes the logger and releases resources.
func Close() error {
	if global != nil && global.rotator != nil {
		return global.rotator.Close()
	}
	return nil
}

// SetLevel changes the minimum log level at runtime.
func SetLevel(l Level) {
	if global != nil {
		global.mu.Lock()
		global.level.Set(l)
		global.mu.Unlock()
	}
}

// GetLevel returns the current minimum log level.
func GetLevel() Level {
	if global != nil {
		global.mu.RLock()
		defer global.mu.RUnlock()
		return global.level.Level()
	}
	return DefaultLevel
}

// ParseLevel converts a string to a Level.
// Accepts: "debug", "info", "warn", "error" (case-insensitive).
// Returns DefaultLevel if the string is invalid.
func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return DefaultLevel
	}
}

// LevelString returns the string representation of a Level.
func LevelString(l Level) string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "warn"
	}
}

// Debug logs a message at DEBUG level.
func Debug(msg string, args ...any) {
	if global != nil {
		global.slog.Debug(msg, args...)
	}
}

// Info logs a message at INFO level.
func Info(msg string, args ...any) {
	if global != nil {
		global.slog.Info(msg, args...)
	}
}

// Warn logs a message at WARN level.
func Warn(msg string, args ...any) {
	if global != nil {
		global.slog.Warn(msg, args...)
	}
}

// Error logs a message at ERROR level.
func Error(msg string, args ...any) {
	if global != nil {
		global.slog.Error(msg, args...)
	}
}

// With returns a new Logger with the given attributes.
func With(args ...any) *slog.Logger {
	if global != nil {
		return global.slog.With(args...)
	}
	return slog.Default()
}

// WithContext returns the logger from context, or the global logger.
func WithContext(ctx context.Context) *slog.Logger {
	if global != nil {
		return global.slog
	}
	return slog.Default()
}

// Printf provides compatibility with standard log.Printf.
// Logs at INFO level.
func Printf(format string, args ...any) {
	if global != nil {
		global.slog.Info(strings.TrimRight(format, "\n"), "args", args)
	}
}

// Println provides compatibility with standard log.Println.
// Logs at INFO level.
func Println(args ...any) {
	if global != nil {
		// Convert args to a single message string
		msg := ""
		for i, arg := range args {
			if i > 0 {
				msg += " "
			}
			msg += toString(arg)
		}
		global.slog.Info(msg)
	}
}

// Fatal logs at ERROR level and exits with code 1.
func Fatal(args ...any) {
	if global != nil {
		msg := ""
		for i, arg := range args {
			if i > 0 {
				msg += " "
			}
			msg += toString(arg)
		}
		global.slog.Error(msg)
	}
	os.Exit(1)
}

// Fatalf logs at ERROR level with formatting and exits with code 1.
func Fatalf(format string, args ...any) {
	if global != nil {
		global.slog.Error(strings.TrimRight(format, "\n"), "args", args)
	}
	os.Exit(1)
}

// toString converts any value to string for compatibility functions.
func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case error:
		return val.Error()
	default:
		return ""
	}
}
