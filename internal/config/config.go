// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package config

import (
	"os"
	"path/filepath"
)

// Config holds the application configuration
type Config struct {
	DatabasePath string
	DataDir      string
	SharesDir    string
	Port         string
	Language     string // "fr" or "en"
}

// Load reads configuration from environment variables or defaults
func Load() (*Config, error) {
	dataDir := os.Getenv("ANEMONE_DATA_DIR")
	if dataDir == "" {
		dataDir = "/app/data"
	}

	return &Config{
		DatabasePath: filepath.Join(dataDir, "db", "anemone.db"),
		DataDir:      dataDir,
		SharesDir:    filepath.Join(dataDir, "shares"),
		Port:         getEnv("PORT", "8080"),
		Language:     getEnv("LANGUAGE", "fr"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
