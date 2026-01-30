// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package updater

import (
	"database/sql"
	"github.com/juste-un-gars/anemone/internal/logger"
	"time"
)

// StartUpdateChecker launches the automatic update checker in a goroutine
// It checks for updates once per day
func StartUpdateChecker(db *sql.DB) {
	logger.Info("🔔 Starting automatic update checker...")

	// Run initial check after 1 minute (to avoid blocking startup)
	go func() {
		time.Sleep(1 * time.Minute)
		checkForUpdates(db)
	}()

	// Run scheduler in background
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			<-ticker.C // Wait for next tick (24 hours)
			checkForUpdates(db)
		}
	}()

	logger.Info("✅ Automatic update checker started (checks every 24 hours)")
}

// checkForUpdates performs the actual update check
func checkForUpdates(db *sql.DB) {
	logger.Info("🔍 Checking for updates...")

	// Check if we already checked recently (skip if checked within last 6 hours)
	lastCheck, err := GetLastUpdateCheck(db)
	if err != nil {
		logger.Info("⚠️  Failed to get last update check time: %v", err)
	} else if !lastCheck.IsZero() && time.Since(lastCheck) < 6*time.Hour {
		logger.Info("⏭️  Skipping update check (last check was %v ago)", time.Since(lastCheck).Round(time.Minute))
		return
	}

	// Perform the check
	info, err := CheckUpdate()
	if err != nil {
		logger.Info("⚠️  Failed to check for updates: %v", err)
		return
	}

	// Save to database
	if err := SaveUpdateInfo(db, info); err != nil {
		logger.Info("⚠️  Failed to save update info: %v", err)
		return
	}

	// Log result
	if info.Available {
		logger.Info("🎉 New version available: %s → %s", info.CurrentVersion, info.LatestVersion)
		logger.Info("📦 Release URL: %s", info.ReleaseURL)
	} else {
		logger.Info("✅ You are running the latest version (%s)", info.CurrentVersion)
	}
}
