// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package updater

import (
	"database/sql"
	"fmt"
	"github.com/juste-un-gars/anemone/internal/logger"
	"time"
)

// SaveUpdateInfo saves update information to the database
func SaveUpdateInfo(db *sql.DB, info *UpdateInfo) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update latest version
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO system_info (key, value, updated_at)
		VALUES ('latest_version', ?, CURRENT_TIMESTAMP)
	`, info.LatestVersion)
	if err != nil {
		return fmt.Errorf("failed to update latest_version: %w", err)
	}

	// Update availability flag
	updateAvailable := "false"
	if info.Available {
		updateAvailable = "true"
	}
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO system_info (key, value, updated_at)
		VALUES ('update_available', ?, CURRENT_TIMESTAMP)
	`, updateAvailable)
	if err != nil {
		return fmt.Errorf("failed to update update_available: %w", err)
	}

	// Update last check time
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO system_info (key, value, updated_at)
		VALUES ('last_update_check', ?, CURRENT_TIMESTAMP)
	`, time.Now().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to update last_update_check: %w", err)
	}

	// Save release URL
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO system_info (key, value, updated_at)
		VALUES ('release_url', ?, CURRENT_TIMESTAMP)
	`, info.ReleaseURL)
	if err != nil {
		return fmt.Errorf("failed to update release_url: %w", err)
	}

	// Save release notes (truncate if too long)
	releaseNotes := info.ReleaseNotes
	if len(releaseNotes) > 5000 {
		releaseNotes = releaseNotes[:5000] + "..."
	}
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO system_info (key, value, updated_at)
		VALUES ('release_notes', ?, CURRENT_TIMESTAMP)
	`, releaseNotes)
	if err != nil {
		return fmt.Errorf("failed to update release_notes: %w", err)
	}

	return tx.Commit()
}

// GetUpdateInfo retrieves update information from the database
func GetUpdateInfo(db *sql.DB) (*UpdateInfo, error) {
	info := &UpdateInfo{
		CurrentVersion: Version,
	}

	// Get latest version
	err := db.QueryRow("SELECT value FROM system_info WHERE key = 'latest_version'").Scan(&info.LatestVersion)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get latest_version: %w", err)
	}

	// Get update available flag
	var updateAvailable string
	err = db.QueryRow("SELECT value FROM system_info WHERE key = 'update_available'").Scan(&updateAvailable)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get update_available: %w", err)
	}
	info.Available = updateAvailable == "true"

	// Get release URL
	err = db.QueryRow("SELECT value FROM system_info WHERE key = 'release_url'").Scan(&info.ReleaseURL)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get release_url: %w", err)
	}

	// Get release notes
	err = db.QueryRow("SELECT value FROM system_info WHERE key = 'release_notes'").Scan(&info.ReleaseNotes)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get release_notes: %w", err)
	}

	return info, nil
}

// GetLastUpdateCheck retrieves the last time we checked for updates
func GetLastUpdateCheck(db *sql.DB) (time.Time, error) {
	var checkTime string
	err := db.QueryRow("SELECT value FROM system_info WHERE key = 'last_update_check'").Scan(&checkTime)
	if err == sql.ErrNoRows || checkTime == "" {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get last_update_check: %w", err)
	}

	t, err := time.Parse(time.RFC3339, checkTime)
	if err != nil {
		return time.Time{}, nil // Invalid format, return zero time
	}

	return t, nil
}

// SyncVersionWithDB ensures the database current_version matches the code version
// This is called at application startup to keep the DB in sync after updates
func SyncVersionWithDB(db *sql.DB) error {
	// Get current version from DB
	var dbVersion string
	err := db.QueryRow("SELECT value FROM system_info WHERE key = 'current_version'").Scan(&dbVersion)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get current_version from DB: %w", err)
	}

	// If DB version differs from code version, update it
	if dbVersion != Version {
		logger.Info("Syncing version: DB has '', code has '' - updating DB", "db_version", dbVersion, "version", Version)

		_, err = db.Exec(`
			INSERT OR REPLACE INTO system_info (key, value, updated_at)
			VALUES ('current_version', ?, CURRENT_TIMESTAMP)
		`, Version)
		if err != nil {
			return fmt.Errorf("failed to update current_version in DB: %w", err)
		}

		logger.Info("Database version updated to", "version", Version)
	}

	return nil
}
