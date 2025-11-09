// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package scheduler

import (
	"database/sql"
	"log"
	"time"

	"github.com/juste-un-gars/anemone/internal/sync"
	"github.com/juste-un-gars/anemone/internal/syncconfig"
)

// Start launches the automatic synchronization scheduler in a goroutine
// It checks every minute if a sync should be triggered based on the configuration
func Start(db *sql.DB) {
	log.Println("🔄 Starting automatic synchronization scheduler...")

	// Run scheduler in background
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			<-ticker.C // Wait for next tick

			// Get sync configuration
			config, err := syncconfig.Get(db)
			if err != nil {
				log.Printf("⚠️  Scheduler: Failed to get sync config: %v", err)
				continue
			}

			// Check if we should sync
			if !syncconfig.ShouldSync(config) {
				continue
			}

			log.Println("🔄 Scheduler: Triggering automatic synchronization...")

			// Perform sync for all users
			successCount, errorCount, lastError := sync.SyncAllUsers(db)

			// Update last sync timestamp
			if err := syncconfig.UpdateLastSync(db); err != nil {
				log.Printf("⚠️  Scheduler: Failed to update last_sync: %v", err)
			}

			// Log results
			if errorCount > 0 {
				log.Printf("⚠️  Scheduler: Sync completed with errors - Success: %d, Errors: %d, Last error: %s",
					successCount, errorCount, lastError)
			} else {
				log.Printf("✅ Scheduler: Sync completed successfully - %d shares synchronized",
					successCount)
			}
		}
	}()

	log.Println("✅ Automatic synchronization scheduler started (checks every 1 minute)")
}
