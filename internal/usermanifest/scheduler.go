// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package usermanifest

import (
	"database/sql"
	"github.com/juste-un-gars/anemone/internal/logger"
	"strings"
	"time"

	"github.com/juste-un-gars/anemone/internal/shares"
)

// DefaultIntervalMinutes is the default interval for manifest generation
const DefaultIntervalMinutes = 5

// StartScheduler starts the periodic manifest generation scheduler.
// It generates manifests for all user shares at the specified interval.
//
// Parameters:
//   - db: database connection for querying shares
//   - sharesDir: base directory for shares (used for logging)
//   - intervalMinutes: interval between manifest generations (0 = disabled)
//
// The scheduler runs in the background and does not block.
func StartScheduler(db *sql.DB, sharesDir string, intervalMinutes int) {
	if intervalMinutes <= 0 {
		logger.Info("📋 User manifest scheduler disabled (interval=0)")
		return
	}

	logger.Info("Starting user manifest scheduler (every minutes)...", "interval_minutes", intervalMinutes)

	// Run initial generation after a short delay (let server start fully)
	go func() {
		time.Sleep(30 * time.Second)
		logger.Info("📋 Running initial manifest generation...")
		if err := GenerateAllManifests(db); err != nil {
			logger.Info("Initial manifest generation failed", "error", err)
		}
	}()

	// Start periodic ticker
	go func() {
		ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			logger.Info("📋 Running scheduled manifest generation...")
			if err := GenerateAllManifests(db); err != nil {
				logger.Info("Scheduled manifest generation failed", "error", err)
			}
		}
	}()

	logger.Info("✅ User manifest scheduler started")
}

// GenerateAllManifests generates manifests for all user shares.
// It queries all shares from the database and generates a manifest for each.
//
// This function can also be called manually (e.g., via an admin API endpoint).
func GenerateAllManifests(db *sql.DB) error {
	startTime := time.Now()

	// Get all shares
	allShares, err := shares.GetAll(db)
	if err != nil {
		return err
	}

	if len(allShares) == 0 {
		logger.Info("   📋 No shares found, skipping manifest generation")
		return nil
	}

	successCount := 0
	errorCount := 0
	totalFiles := 0
	var totalSize int64

	for _, share := range allShares {
		// Determine share type from name
		shareType := determineShareType(share.Name)

		// Get username for this share
		username, err := getUsername(db, share.UserID)
		if err != nil {
			logger.Info("Failed to get username for share", "name", share.Name, "error", err)
			errorCount++
			continue
		}

		// Build manifest
		manifest, err := BuildUserManifest(share.Path, share.Name, shareType, username)
		if err != nil {
			logger.Info("Failed to build manifest for", "name", share.Name, "error", err)
			errorCount++
			continue
		}

		// Write manifest
		if err := WriteManifest(manifest, share.Path); err != nil {
			logger.Info("Failed to write manifest for", "name", share.Name, "error", err)
			errorCount++
			continue
		}

		successCount++
		totalFiles += manifest.FileCount
		totalSize += manifest.TotalSize
	}

	elapsed := time.Since(startTime)

	logger.Info("Manifest generation complete in : shares processed ( files, )", "round", elapsed.Round(time.Millisecond), "success_count", successCount, "total_files", totalFiles, "total_size", FormatSize(totalSize))

	if errorCount > 0 {
		logger.Info("shares had errors", "error_count", errorCount)
	}

	return nil
}

// GenerateManifestForShare generates a manifest for a single share.
// This can be called when a share is created or modified.
//
// Parameters:
//   - db: database connection
//   - shareID: ID of the share to generate manifest for
func GenerateManifestForShare(db *sql.DB, shareID int) error {
	share, err := shares.GetByID(db, shareID)
	if err != nil {
		return err
	}

	shareType := determineShareType(share.Name)

	username, err := getUsername(db, share.UserID)
	if err != nil {
		return err
	}

	manifest, err := BuildUserManifest(share.Path, share.Name, shareType, username)
	if err != nil {
		return err
	}

	return WriteManifest(manifest, share.Path)
}

// determineShareType determines if a share is "data" or "backup" based on its name.
func determineShareType(shareName string) string {
	if strings.HasPrefix(shareName, "backup_") || shareName == "backup" {
		return "backup"
	}
	return "data"
}

// getUsername retrieves the username for a user ID.
func getUsername(db *sql.DB, userID int) (string, error) {
	var username string
	err := db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		return "", err
	}
	return username, nil
}
