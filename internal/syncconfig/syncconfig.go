// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package syncconfig

import (
	"database/sql"
	"fmt"
	"time"
)

// SyncConfig represents the automatic synchronization configuration
type SyncConfig struct {
	ID        int
	Enabled   bool
	Interval  string // "30min", "1h", "2h", "6h", "fixed"
	FixedHour int    // If Interval="fixed", hour to sync (0-23)
	LastSync  *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Get retrieves the sync configuration
func Get(db *sql.DB) (*SyncConfig, error) {
	config := &SyncConfig{}
	query := `SELECT id, enabled, interval, fixed_hour, last_sync, created_at, updated_at
	          FROM sync_config WHERE id = 1`

	err := db.QueryRow(query).Scan(
		&config.ID, &config.Enabled, &config.Interval, &config.FixedHour,
		&config.LastSync, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync config: %w", err)
	}

	return config, nil
}

// Update updates the sync configuration
func Update(db *sql.DB, config *SyncConfig) error {
	query := `UPDATE sync_config
	          SET enabled = ?, interval = ?, fixed_hour = ?, updated_at = CURRENT_TIMESTAMP
	          WHERE id = 1`

	_, err := db.Exec(query, config.Enabled, config.Interval, config.FixedHour)
	if err != nil {
		return fmt.Errorf("failed to update sync config: %w", err)
	}

	return nil
}

// UpdateLastSync updates the last sync timestamp
func UpdateLastSync(db *sql.DB) error {
	query := `UPDATE sync_config SET last_sync = CURRENT_TIMESTAMP WHERE id = 1`

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to update last_sync: %w", err)
	}

	return nil
}

// ShouldSync determines if a sync should happen based on config
func ShouldSync(config *SyncConfig) bool {
	if !config.Enabled {
		return false
	}

	if config.LastSync == nil {
		// First sync ever
		return true
	}

	now := time.Now()

	if config.Interval == "fixed" {
		// Fixed hour sync (e.g., 23h daily)
		// Check if we're past the fixed hour today and haven't synced today
		lastSyncDate := config.LastSync.Format("2006-01-02")
		todayDate := now.Format("2006-01-02")

		if lastSyncDate != todayDate && now.Hour() >= config.FixedHour {
			return true
		}
		return false
	}

	// Interval-based sync
	var interval time.Duration
	switch config.Interval {
	case "30min":
		interval = 30 * time.Minute
	case "1h":
		interval = 1 * time.Hour
	case "2h":
		interval = 2 * time.Hour
	case "6h":
		interval = 6 * time.Hour
	default:
		interval = 1 * time.Hour
	}

	return now.Sub(*config.LastSync) >= interval
}
