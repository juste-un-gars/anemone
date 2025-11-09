// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package syncauth

import (
	"database/sql"
	"fmt"

	"github.com/juste-un-gars/anemone/internal/crypto"
)

// GetSyncAuthPassword retrieves the sync authentication password hash from system_config
// Returns empty string if not configured
func GetSyncAuthPassword(db *sql.DB) (string, error) {
	var passwordHash string
	err := db.QueryRow("SELECT value FROM system_config WHERE key = 'sync_auth_password'").Scan(&passwordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			// No password configured yet - this is OK
			return "", nil
		}
		return "", fmt.Errorf("failed to get sync auth password: %w", err)
	}
	return passwordHash, nil
}

// SetSyncAuthPassword sets the sync authentication password in system_config
// Password is hashed with bcrypt before storage
func SetSyncAuthPassword(db *sql.DB, password string) error {
	// Hash the password
	hashedPassword, err := crypto.HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert or update
	query := `INSERT INTO system_config (key, value, updated_at)
	          VALUES ('sync_auth_password', ?, CURRENT_TIMESTAMP)
	          ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = CURRENT_TIMESTAMP`

	_, err = db.Exec(query, hashedPassword, hashedPassword)
	if err != nil {
		return fmt.Errorf("failed to set sync auth password: %w", err)
	}

	return nil
}

// CheckSyncAuthPassword checks if the provided password matches the stored hash
// Returns true if password is correct, false otherwise
func CheckSyncAuthPassword(db *sql.DB, password string) (bool, error) {
	passwordHash, err := GetSyncAuthPassword(db)
	if err != nil {
		return false, err
	}

	// If no password is configured, allow access (backward compatibility)
	if passwordHash == "" {
		return true, nil
	}

	// Check if password matches
	return crypto.CheckPassword(password, passwordHash), nil
}

// IsConfigured checks if a sync auth password is configured
func IsConfigured(db *sql.DB) (bool, error) {
	passwordHash, err := GetSyncAuthPassword(db)
	if err != nil {
		return false, err
	}
	return passwordHash != "", nil
}
