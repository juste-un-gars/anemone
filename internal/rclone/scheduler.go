// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package rclone

import (
	"database/sql"
	"time"

	"github.com/juste-un-gars/anemone/internal/logger"
)

// CleanupStaleRunning resets any "running" status left over from a previous process.
// Should be called on startup before the scheduler starts.
func CleanupStaleRunning(db *sql.DB) {
	result, err := db.Exec("UPDATE rclone_backups SET last_status = 'error', last_error = 'sync interrupted (service restart)' WHERE last_status = 'running'")
	if err != nil {
		logger.Warn("Rclone: Failed to cleanup stale running statuses", "error", err)
		return
	}
	if rows, _ := result.RowsAffected(); rows > 0 {
		logger.Info("Rclone: Reset stale running sync(s) from previous run", "rows", rows)
	}
}

// checkStaleRunning detects backups stuck in "running" status without an active process.
func checkStaleRunning(db *sql.DB) {
	backups, err := GetAll(db)
	if err != nil {
		return
	}
	for _, b := range backups {
		if b.LastStatus != "running" {
			continue
		}
		if !IsBackupSyncing(b.ID) {
			logger.Info("Rclone: Detected stale running status for '' (no active process), marking as error", "name", b.Name)
			UpdateSyncStatus(db, b.ID, "error", "sync process terminated unexpectedly", 0, 0)
		}
	}
}

// StartScheduler launches the automatic rclone backup scheduler in a goroutine.
// It checks every minute if a sync should be triggered and monitors running syncs.
func StartScheduler(db *sql.DB, dataDir string) {
	logger.Info("🔄 Starting rclone backup scheduler...")

	// Run scheduler in background
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			<-ticker.C // Wait for next tick

			// Check for stale "running" statuses (process died without updating DB)
			checkStaleRunning(db)

			// Check if rclone is installed
			if !IsRcloneInstalled() {
				continue
			}

			// Get all enabled rclone backups
			backups, err := GetEnabled(db)
			if err != nil {
				logger.Info("Rclone Scheduler: Failed to get backups", "error", err)
				continue
			}

			// Check each backup individually
			for _, backup := range backups {
				// Skip if already syncing
				if backup.LastStatus == "running" {
					continue
				}

				// Check if this backup should be synced now
				if !backup.ShouldSync() {
					continue
				}

				logger.Info("Rclone Scheduler: Triggering sync for '' (frequency: )...", "name", backup.Name, "sync_frequency", backup.SyncFrequency)

				// Perform sync in a goroutine so we don't block other backups
				go func(b *RcloneBackup) {
					result, syncErr := Sync(db, b, dataDir)

					if syncErr != nil {
						logger.Info("Rclone Scheduler: Sync to failed", "name", b.Name, "sync_err", syncErr)
					} else if result != nil {
						if len(result.Errors) > 0 {
							logger.Info("Rclone Scheduler: Sync to completed with errors - Files: , Errors", "name", b.Name, "files_transferred", result.FilesTransferred, "errors", len(result.Errors))
						} else {
							logger.Info("Rclone Scheduler: Sync to completed - Files:", "name", b.Name, "files_transferred", result.FilesTransferred, "bytes_transferred", FormatBytes(result.BytesTransferred))
						}
					}
				}(backup)
			}
		}
	}()

	logger.Info("✅ Rclone backup scheduler started (checks every 1 minute)")
}
