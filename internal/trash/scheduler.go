// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package trash

import (
	"database/sql"
	"log"
	"time"
)

// StartCleanupScheduler launches the automatic trash cleanup scheduler
// It runs cleanup daily at 3 AM (configurable retention days)
func StartCleanupScheduler(db *sql.DB, getRetentionDays func() (int, error)) {
	log.Println("🗑️  Starting automatic trash cleanup scheduler...")

	// Run initial cleanup on startup
	go func() {
		log.Println("🗑️  Running initial trash cleanup...")
		runCleanup(db, getRetentionDays)
	}()

	// Run scheduler in background
	go func() {
		// Calculate time until next 3 AM
		now := time.Now()
		next3AM := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
		if now.After(next3AM) {
			// If it's already past 3 AM today, schedule for tomorrow
			next3AM = next3AM.Add(24 * time.Hour)
		}

		// Wait until 3 AM
		duration := time.Until(next3AM)
		log.Printf("🗑️  Next trash cleanup scheduled for: %s (in %s)", next3AM.Format("2006-01-02 15:04:05"), duration.Round(time.Minute))
		time.Sleep(duration)

		// Create ticker for daily cleanup at 3 AM
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			// Run cleanup
			runCleanup(db, getRetentionDays)

			// Wait for next tick (24 hours)
			<-ticker.C
		}
	}()

	log.Println("✅ Automatic trash cleanup scheduler started (runs daily at 3 AM)")
}

// runCleanup executes the trash cleanup for all users
func runCleanup(db *sql.DB, getRetentionDays func() (int, error)) {
	// Get current retention days setting
	retentionDays, err := getRetentionDays()
	if err != nil {
		log.Printf("⚠️  Trash cleanup: Failed to get retention days: %v", err)
		return
	}

	if retentionDays == 0 {
		log.Println("🗑️  Trash cleanup: Retention disabled (0 days = infinite retention)")
		return
	}

	log.Printf("🗑️  Trash cleanup: Running with %d days retention...", retentionDays)

	// Run cleanup for all users
	totalDeleted, err := CleanupAllUserTrash(db, retentionDays)
	if err != nil {
		log.Printf("⚠️  Trash cleanup: Failed: %v", err)
		return
	}

	if totalDeleted > 0 {
		log.Printf("✅ Trash cleanup: Completed - Deleted %d old item(s)", totalDeleted)
	} else {
		log.Println("✅ Trash cleanup: Completed - No old items to delete")
	}
}
