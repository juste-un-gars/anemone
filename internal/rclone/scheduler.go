// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package rclone

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/juste-un-gars/anemone/internal/logger"
)

// StartScheduler launches the automatic rclone backup scheduler in a goroutine
// It checks every minute if a sync should be triggered for each backup based on their individual configuration
func StartScheduler(db *sql.DB, dataDir string) {
	logger.Info("🔄 Starting rclone backup scheduler...")

	// Run scheduler in background
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			<-ticker.C // Wait for next tick

			// Check if rclone is installed
			if !IsRcloneInstalled() {
				// Silently skip if rclone is not installed
				continue
			}

			// Get all enabled rclone backups
			backups, err := GetEnabled(db)
			if err != nil {
				logger.Info(fmt.Sprintf("Rclone Scheduler: Failed to get backups: %v", err))
				continue
			}

			// Check each backup individually
			for _, backup := range backups {
				// Check if this backup should be synced now
				if !backup.ShouldSync() {
					continue
				}

				logger.Info(fmt.Sprintf("Rclone Scheduler: Triggering sync for '%s' (frequency: %s)...",
					backup.Name, backup.SyncFrequency))

				// Perform sync
				result, syncErr := Sync(db, backup, dataDir)

				// Log results
				if syncErr != nil {
					logger.Info(fmt.Sprintf("Rclone Scheduler: Sync to %s failed: %v", backup.Name, syncErr))
				} else if result != nil {
					if len(result.Errors) > 0 {
						logger.Info(fmt.Sprintf("Rclone Scheduler: Sync to %s completed with errors - Files: %d, Errors: %d",
							backup.Name, result.FilesTransferred, len(result.Errors)))
					} else {
						logger.Info(fmt.Sprintf("Rclone Scheduler: Sync to %s completed - Files: %d, %s",
							backup.Name, result.FilesTransferred, FormatBytes(result.BytesTransferred)))
					}
				}
			}
		}
	}()

	logger.Info("✅ Rclone backup scheduler started (checks every 1 minute)")
}
