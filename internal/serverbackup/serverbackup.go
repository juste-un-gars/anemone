// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package serverbackup handles automated server configuration backups with rotation.
package serverbackup

import (
	"database/sql"
	"fmt"
	"github.com/juste-un-gars/anemone/internal/logger"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/juste-un-gars/anemone/internal/backup"
)

const (
	MaxBackups = 10 // Maximum number of backups to keep
)

// BackupFile represents a server backup file
type BackupFile struct {
	Filename  string
	Path      string
	Size      int64
	CreatedAt time.Time
}

// CreateServerBackup creates a server backup encrypted with master key
func CreateServerBackup(db *sql.DB, backupDir string) (string, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Get master key
	var masterKey string
	err := db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to get master key: %w", err)
	}

	// Export configuration
	serverBackup, err := backup.ExportConfiguration(db, "Anemone Server")
	if err != nil {
		return "", fmt.Errorf("failed to export configuration: %w", err)
	}

	// Encrypt with master key
	encryptedData, err := backup.EncryptBackup(serverBackup, masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt backup: %w", err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("backup_%s.enc", timestamp)
	filepath := filepath.Join(backupDir, filename)

	// Write to file
	if err := os.WriteFile(filepath, encryptedData, 0600); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	logger.Info("Server backup created: %s (%d bytes)", filename, len(encryptedData))

	// Clean old backups
	if err := CleanOldBackups(backupDir, MaxBackups); err != nil {
		logger.Info("Warning: failed to clean old backups: %v", err)
	}

	return filepath, nil
}

// ListBackups lists all server backups sorted by creation date (newest first)
func ListBackups(backupDir string) ([]BackupFile, error) {
	// Check if directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return []BackupFile{}, nil
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []BackupFile
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".enc" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backups = append(backups, BackupFile{
			Filename:  entry.Name(),
			Path:      filepath.Join(backupDir, entry.Name()),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}

	// Sort by creation date (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// CleanOldBackups removes old backups keeping only the maxBackups newest ones
func CleanOldBackups(backupDir string, maxBackups int) error {
	backups, err := ListBackups(backupDir)
	if err != nil {
		return err
	}

	if len(backups) <= maxBackups {
		return nil
	}

	// Delete old backups
	for i := maxBackups; i < len(backups); i++ {
		if err := os.Remove(backups[i].Path); err != nil {
			logger.Info("Warning: failed to remove old backup %s: %v", backups[i].Filename, err)
		} else {
			logger.Info("Removed old backup: %s", backups[i].Filename)
		}
	}

	return nil
}

// ReEncryptBackup re-encrypts a backup from master key to user-provided passphrase
func ReEncryptBackup(db *sql.DB, backupPath string, newPassphrase string) ([]byte, error) {
	// Get master key
	var masterKey string
	err := db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get master key: %w", err)
	}

	// Read encrypted backup
	encryptedData, err := os.ReadFile(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}

	// Decrypt with master key
	serverBackup, err := backup.DecryptBackup(encryptedData, masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt backup: %w", err)
	}

	// Re-encrypt with new passphrase
	reEncryptedData, err := backup.EncryptBackup(serverBackup, newPassphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to re-encrypt backup: %w", err)
	}

	return reEncryptedData, nil
}

// StartScheduler starts the automatic backup scheduler (runs at 4 AM daily)
func StartScheduler(db *sql.DB, dataDir string) {
	backupDir := filepath.Join(dataDir, "backups", "server")

	go func() {
		logger.Info("🕐 Server backup scheduler started (daily at 4:00 AM)")

		for {
			now := time.Now()

			// Calculate next 4 AM
			next4AM := time.Date(now.Year(), now.Month(), now.Day(), 4, 0, 0, 0, now.Location())
			if now.After(next4AM) {
				// If it's past 4 AM today, schedule for tomorrow
				next4AM = next4AM.Add(24 * time.Hour)
			}

			// Sleep until next 4 AM
			duration := time.Until(next4AM)
			logger.Info("Next automatic server backup scheduled", "at", next4AM.Format("2006-01-02 15:04:05"), "in", duration.Round(time.Minute))

			time.Sleep(duration)

			// Create backup
			logger.Info("Creating automatic server backup...")
			backupPath, err := CreateServerBackup(db, backupDir)
			if err != nil {
				logger.Info("❌ Automatic backup failed: %v", err)
			} else {
				logger.Info("✅ Automatic server backup created: %s", backupPath)
			}

			// Sleep a bit to avoid creating multiple backups if the clock changes
			time.Sleep(1 * time.Minute)
		}
	}()
}
