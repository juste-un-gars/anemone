// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package scheduler provides automatic P2P synchronization scheduling based on peer configurations.
package scheduler

import (
	"database/sql"
	"github.com/juste-un-gars/anemone/internal/logger"
	"time"

	"github.com/juste-un-gars/anemone/internal/peers"
	"github.com/juste-un-gars/anemone/internal/sync"
)

// Start launches the automatic synchronization scheduler in a goroutine
// It checks every minute if a sync should be triggered for each peer based on their individual configuration
func Start(db *sql.DB) {
	logger.Info("🔄 Starting automatic synchronization scheduler...")

	// Run scheduler in background
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			<-ticker.C // Wait for next tick

			// Get all peers
			allPeers, err := peers.GetAll(db)
			if err != nil {
				logger.Info("Scheduler: Failed to get peers", "error", err)
				continue
			}

			// Check each peer individually
			for _, peer := range allPeers {
				// Check if this peer should be synced now
				if !peers.ShouldSyncPeer(peer) {
					continue
				}

				logger.Info("Scheduler: Triggering sync to peer '' (frequency: )...", "name", peer.Name, "sync_frequency", peer.SyncFrequency)

				// Perform sync for this peer
				successCount, errorCount, lastError := sync.SyncPeer(db, peer.ID, peer.Name, peer.Address, peer.Port, peer.Password, peer.SyncTimeoutHours)

				// Update last sync timestamp for this peer
				if err := peers.UpdateLastSync(db, peer.ID); err != nil {
					logger.Info("Scheduler: Failed to update last_sync for peer", "name", peer.Name, "error", err)
				}

				// Log results
				if errorCount > 0 {
					logger.Info("Scheduler: Sync to completed with errors - Success: , Errors: , Last error", "name", peer.Name, "success_count", successCount, "error_count", errorCount, "last_error", lastError)
				} else {
					logger.Info("Scheduler: Sync to completed successfully - shares synchronized", "name", peer.Name, "success_count", successCount)
				}
			}
		}
	}()

	logger.Info("✅ Automatic synchronization scheduler started (checks every 1 minute)")
}
