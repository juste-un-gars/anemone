// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package usbbackup

import (
	"database/sql"
	"github.com/juste-un-gars/anemone/internal/logger"
	"path/filepath"
	"time"

	"github.com/juste-un-gars/anemone/internal/sync"
)

// StartScheduler launches the automatic USB backup scheduler in a goroutine
// It checks every minute if a sync should be triggered for each USB backup based on their individual configuration
func StartScheduler(db *sql.DB, dataDir string) {
	logger.Info("🔄 Starting USB backup scheduler...")

	// Run scheduler in background
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			<-ticker.C // Wait for next tick

			// Get all enabled USB backups
			backups, err := GetEnabled(db)
			if err != nil {
				logger.Info("USB Scheduler: Failed to get backups", "error", err)
				continue
			}

			// Check each backup individually
			for _, backup := range backups {
				// Check if this backup should be synced now
				if !backup.ShouldSync() {
					continue
				}

				logger.Info("USB Scheduler: Triggering sync for '' (frequency: )...", "name", backup.Name, "sync_frequency", backup.SyncFrequency)

				// Get master key
				var masterKey string
				if err := db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey); err != nil {
					logger.Info("USB Scheduler: Failed to get master key", "error", err)
					continue
				}

				// Get server name
				serverName, _ := sync.GetServerName(db)
				if serverName == "" {
					serverName = "anemone"
				}

				// Build config info
				configInfo := &ConfigBackupInfo{
					DataDir:  dataDir,
					DBPath:   filepath.Join(dataDir, "db", "anemone.db"),
					CertsDir: filepath.Join(dataDir, "certs"),
					SMBConf:  filepath.Join(dataDir, "smb", "smb.conf"),
				}

				// Perform sync
				var result *SyncResult
				var syncErr error

				if backup.BackupType == BackupTypeConfig {
					// Config-only backup
					result, syncErr = SyncConfig(db, backup, configInfo, masterKey, serverName)
				} else {
					// Full backup (config + data)
					configResult, _ := SyncConfig(db, backup, configInfo, masterKey, serverName)
					result, syncErr = SyncAllShares(db, backup, masterKey, serverName)
					if result != nil && configResult != nil {
						result.FilesAdded += configResult.FilesAdded
						result.BytesSynced += configResult.BytesSynced
					}
				}

				// Log results
				if syncErr != nil {
					logger.Info("USB Scheduler: Sync to failed", "name", backup.Name, "sync_err", syncErr)
				} else if result != nil {
					if len(result.Errors) > 0 {
						logger.Info("USB Scheduler: Sync to completed with errors - Added: , Updated: , Deleted: , Errors", "name", backup.Name, "files_added", result.FilesAdded, "files_updated", result.FilesUpdated, "files_deleted", result.FilesDeleted, "errors", len(result.Errors))
					} else {
						logger.Info("USB Scheduler: Sync to completed - Added: , Updated: , Deleted:", "name", backup.Name, "files_added", result.FilesAdded, "files_updated", result.FilesUpdated, "files_deleted", result.FilesDeleted, "bytes_synced", FormatBytes(result.BytesSynced))
					}
				}
			}
		}
	}()

	logger.Info("✅ USB backup scheduler started (checks every 1 minute)")
}
