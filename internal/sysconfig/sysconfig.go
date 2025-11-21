// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package sysconfig

import (
	"database/sql"
	"fmt"
	"strconv"
)

// GetTrashRetentionDays returns the number of days to keep files in trash
// Default is 30 days if not configured
func GetTrashRetentionDays(db *sql.DB) (int, error) {
	var value string
	err := db.QueryRow("SELECT value FROM system_config WHERE key = 'trash_retention_days'").Scan(&value)
	if err == sql.ErrNoRows {
		// Not configured yet, return default
		return 30, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get trash_retention_days: %w", err)
	}

	days, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid trash_retention_days value: %w", err)
	}

	return days, nil
}

// SetTrashRetentionDays sets the number of days to keep files in trash
func SetTrashRetentionDays(db *sql.DB, days int) error {
	query := `INSERT INTO system_config (key, value, updated_at)
		VALUES ('trash_retention_days', ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`

	_, err := db.Exec(query, strconv.Itoa(days))
	if err != nil {
		return fmt.Errorf("failed to set trash_retention_days: %w", err)
	}

	return nil
}
