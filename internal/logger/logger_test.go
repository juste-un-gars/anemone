// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"WARN", LevelWarn},
		{"warning", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"invalid", DefaultLevel},
		{"", DefaultLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseLevel(tt.input)
			if got != tt.expected {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "debug"},
		{LevelInfo, "info"},
		{LevelWarn, "warn"},
		{LevelError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := LevelString(tt.level)
			if got != tt.expected {
				t.Errorf("LevelString(%v) = %q, want %q", tt.level, got, tt.expected)
			}
		})
	}
}

func TestInitAndSetLevel(t *testing.T) {
	// Initialize with INFO level
	err := Init(&Config{Level: LevelInfo})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	// Check initial level
	if got := GetLevel(); got != LevelInfo {
		t.Errorf("Initial level = %v, want %v", got, LevelInfo)
	}

	// Change level
	SetLevel(LevelDebug)
	if got := GetLevel(); got != LevelDebug {
		t.Errorf("After SetLevel(DEBUG) = %v, want %v", got, LevelDebug)
	}

	// Change to ERROR
	SetLevel(LevelError)
	if got := GetLevel(); got != LevelError {
		t.Errorf("After SetLevel(ERROR) = %v, want %v", got, LevelError)
	}
}

func TestInitWithLogDir(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	// Initialize with log directory
	err := Init(&Config{
		Level:         LevelInfo,
		LogDir:        logDir,
		RetentionDays: 7,
		MaxSizeMB:     10,
	})
	if err != nil {
		t.Fatalf("Init with LogDir failed: %v", err)
	}
	defer Close()

	// Write some logs
	Info("Test message")

	// Check that log directory was created
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("Log directory was not created")
	}

	// Check that a log file was created
	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("Failed to read log directory: %v", err)
	}

	if len(entries) == 0 {
		t.Error("No log files were created")
	}
}

func TestLogFunctions(t *testing.T) {
	// Initialize with DEBUG level to capture all logs
	err := Init(&Config{Level: LevelDebug})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	// These should not panic
	Debug("debug message", "key", "value")
	Info("info message", "count", 42)
	Warn("warn message", "error", "something went wrong")
	Error("error message", "fatal", true)
}
