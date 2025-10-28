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
	HTTPSPort    string
	Language     string // "fr" or "en"

	// TLS configuration
	EnableHTTPS   bool
	EnableHTTP    bool   // Disabled by default for security
	TLSCertPath   string // Path to custom TLS certificate
	TLSKeyPath    string // Path to custom TLS private key
}

// Load reads configuration from environment variables or defaults
func Load() (*Config, error) {
	dataDir := os.Getenv("ANEMONE_DATA_DIR")
	if dataDir == "" {
		dataDir = "/app/data"
	}

	// TLS is enabled by default, HTTP is disabled by default for security
	enableHTTPS := getBoolEnv("ENABLE_HTTPS", true)
	enableHTTP := getBoolEnv("ENABLE_HTTP", false)

	// Ensure at least one protocol is enabled
	if !enableHTTPS && !enableHTTP {
		enableHTTPS = true
	}

	cfg := &Config{
		DatabasePath: filepath.Join(dataDir, "db", "anemone.db"),
		DataDir:      dataDir,
		SharesDir:    filepath.Join(dataDir, "shares"),
		Port:         getEnv("PORT", "8080"),
		HTTPSPort:    getEnv("HTTPS_PORT", "8443"),
		Language:     getEnv("LANGUAGE", "fr"),
		EnableHTTPS:  enableHTTPS,
		EnableHTTP:   enableHTTP,
		TLSCertPath:  getEnv("TLS_CERT_PATH", ""),
		TLSKeyPath:   getEnv("TLS_KEY_PATH", ""),
	}

	// If custom cert/key not provided, use auto-generated ones
	if cfg.TLSCertPath == "" {
		cfg.TLSCertPath = filepath.Join(dataDir, "certs", "server.crt")
	}
	if cfg.TLSKeyPath == "" {
		cfg.TLSKeyPath = filepath.Join(dataDir, "certs", "server.key")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}
