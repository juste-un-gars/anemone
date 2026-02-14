// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package config handles application configuration loading from environment variables.
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
	IncomingDir  string // Directory for incoming backups from peers (can be on separate disk)
	Port         string
	HTTPSPort    string
	Language     string // "fr" or "en"

	// TLS configuration
	EnableHTTPS bool
	EnableHTTP  bool   // Disabled by default for security
	TLSCertPath string // Path to custom TLS certificate
	TLSKeyPath  string // Path to custom TLS private key

	// Logging configuration
	LogLevel string // "debug", "info", "warn", "error" (default: "warn")
	LogDir   string // Directory for log files (default: DataDir/logs)

	// OnlyOffice integration
	OnlyOfficeEnabled bool   // Enable OnlyOffice document editing
	OnlyOfficeURL     string // Internal URL of OnlyOffice container (e.g., http://localhost:8443)
	OnlyOfficeSecret  string // JWT shared secret for OnlyOffice communication
}

// Load reads configuration from environment variables or defaults
func Load() (*Config, error) {
	dataDir := os.Getenv("ANEMONE_DATA_DIR")
	if dataDir == "" {
		dataDir = "/srv/anemone"
	}

	// TLS is enabled by default, HTTP is disabled by default for security
	enableHTTPS := getBoolEnv("ENABLE_HTTPS", true)
	enableHTTP := getBoolEnv("ENABLE_HTTP", false)

	// Ensure at least one protocol is enabled
	if !enableHTTPS && !enableHTTP {
		enableHTTPS = true
	}

	// SharesDir and IncomingDir can be on separate disks
	// IncomingDir is for backups from peers (doesn't need ZFS redundancy)
	sharesDir := os.Getenv("ANEMONE_SHARES_DIR")
	if sharesDir == "" {
		sharesDir = filepath.Join(dataDir, "shares")
	}
	incomingDir := os.Getenv("ANEMONE_INCOMING_DIR")
	if incomingDir == "" {
		incomingDir = filepath.Join(dataDir, "backups", "incoming")
	}

	// Log directory can be customized, defaults to DataDir/logs
	logDir := os.Getenv("ANEMONE_LOG_DIR")
	if logDir == "" {
		logDir = filepath.Join(dataDir, "logs")
	}

	cfg := &Config{
		DatabasePath: filepath.Join(dataDir, "db", "anemone.db"),
		DataDir:      dataDir,
		SharesDir:    sharesDir,
		IncomingDir:  incomingDir,
		Port:         getEnv("PORT", "8080"),
		HTTPSPort:    getEnv("HTTPS_PORT", "8443"),
		Language:     getEnv("LANGUAGE", "fr"),
		EnableHTTPS:  enableHTTPS,
		EnableHTTP:   enableHTTP,
		TLSCertPath:  getEnv("TLS_CERT_PATH", ""),
		TLSKeyPath:   getEnv("TLS_KEY_PATH", ""),
		LogLevel:     getEnv("ANEMONE_LOG_LEVEL", ""), // Empty = use DB setting or default
		LogDir:       logDir,

		OnlyOfficeEnabled: getBoolEnv("ANEMONE_OO_ENABLED", false),
		OnlyOfficeURL:     getEnv("ANEMONE_OO_URL", "http://localhost:9980"),
		OnlyOfficeSecret:  getEnv("ANEMONE_OO_SECRET", ""),
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

// ValidateDirs ensures that required directories exist and are writable.
// Creates directories if they don't exist.
// Returns warnings for any issues found (doesn't fail - setup wizard may handle later).
func (c *Config) ValidateDirs() []string {
	var warnings []string

	dirs := map[string]string{
		"DataDir":     c.DataDir,
		"SharesDir":   c.SharesDir,
		"IncomingDir": c.IncomingDir,
	}

	for name, path := range dirs {
		// Check if directory exists
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			// Try to create it
			if err := os.MkdirAll(path, 0755); err != nil {
				warnings = append(warnings, name+": cannot create directory: "+err.Error())
				continue
			}
		} else if err != nil {
			warnings = append(warnings, name+": cannot access directory: "+err.Error())
			continue
		} else if !info.IsDir() {
			warnings = append(warnings, name+": path exists but is not a directory")
			continue
		}

		// Check write permissions by creating a temp file
		testFile := filepath.Join(path, ".anemone-write-test")
		f, err := os.Create(testFile)
		if err != nil {
			warnings = append(warnings, name+": directory not writable: "+err.Error())
		} else {
			f.Close()
			os.Remove(testFile)
		}
	}

	return warnings
}
