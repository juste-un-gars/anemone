// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package scheduler provides automatic P2P synchronization scheduling based on peer configurations.
package scheduler

import (
	"database/sql"
	"log"
	"time"

	"github.com/juste-un-gars/anemone/internal/peers"
	"github.com/juste-un-gars/anemone/internal/sync"
)

// Start launches the automatic synchronization scheduler in a goroutine
// It checks every minute if a sync should be triggered for each peer based on their individual configuration
func Start(db *sql.DB) {
	log.Println("🔄 Starting automatic synchronization scheduler...")

	// Run scheduler in background
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			<-ticker.C // Wait for next tick

			// Get all peers
			allPeers, err := peers.GetAll(db)
			if err != nil {
				log.Printf("⚠️  Scheduler: Failed to get peers: %v", err)
				continue
			}

			// Check each peer individually
			for _, peer := range allPeers {
				// Check if this peer should be synced now
				if !peers.ShouldSyncPeer(peer) {
					continue
				}

				log.Printf("🔄 Scheduler: Triggering sync to peer '%s' (frequency: %s)...", peer.Name, peer.SyncFrequency)

				// Perform sync for this peer
				successCount, errorCount, lastError := sync.SyncPeer(db, peer.ID, peer.Name, peer.Address, peer.Port, peer.Password, peer.SyncTimeoutHours)

				// Update last sync timestamp for this peer
				if err := peers.UpdateLastSync(db, peer.ID); err != nil {
					log.Printf("⚠️  Scheduler: Failed to update last_sync for peer %s: %v", peer.Name, err)
				}

				// Log results
				if errorCount > 0 {
					log.Printf("⚠️  Scheduler: Sync to %s completed with errors - Success: %d, Errors: %d, Last error: %s",
						peer.Name, successCount, errorCount, lastError)
				} else {
					log.Printf("✅ Scheduler: Sync to %s completed successfully - %d shares synchronized",
						peer.Name, successCount)
				}
			}
		}
	}()

	log.Println("✅ Automatic synchronization scheduler started (checks every 1 minute)")
}
