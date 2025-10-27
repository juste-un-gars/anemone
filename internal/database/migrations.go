// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package database

import (
	"database/sql"
	"fmt"
)

// Migrate runs all database migrations
func Migrate(db *sql.DB) error {
	migrations := []string{
		// System configuration
		`CREATE TABLE IF NOT EXISTS system_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Users
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			email TEXT,
			encryption_key_hash TEXT NOT NULL,
			encryption_key_encrypted BLOB NOT NULL,
			is_admin BOOLEAN DEFAULT 0,
			quota_total_gb INTEGER DEFAULT 100,
			quota_backup_gb INTEGER DEFAULT 50,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			activated_at DATETIME,
			last_login DATETIME
		)`,

		// Activation tokens (temporary links for new users)
		`CREATE TABLE IF NOT EXISTS activation_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			token TEXT UNIQUE NOT NULL,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			email TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			used_at DATETIME,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		// Shares (backup + optional other shares)
		`CREATE TABLE IF NOT EXISTS shares (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			protocol TEXT DEFAULT 'smb',
			sync_enabled BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE(user_id, name)
		)`,

		// Trash items (deleted files)
		`CREATE TABLE IF NOT EXISTS trash_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			share_id INTEGER NOT NULL,
			original_path TEXT NOT NULL,
			deleted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			size_bytes INTEGER DEFAULT 0,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (share_id) REFERENCES shares(id) ON DELETE CASCADE
		)`,

		// Peers for P2P sync
		`CREATE TABLE IF NOT EXISTS peers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			url TEXT NOT NULL,
			enabled BOOLEAN DEFAULT 1,
			last_sync DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Sync log (track synchronization history)
		`CREATE TABLE IF NOT EXISTS sync_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			peer_id INTEGER NOT NULL,
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			status TEXT DEFAULT 'running',
			files_synced INTEGER DEFAULT 0,
			bytes_synced INTEGER DEFAULT 0,
			error_message TEXT,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (peer_id) REFERENCES peers(id) ON DELETE CASCADE
		)`,

		// Indexes for performance
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,
		`CREATE INDEX IF NOT EXISTS idx_activation_tokens_token ON activation_tokens(token)`,
		`CREATE INDEX IF NOT EXISTS idx_activation_tokens_expires ON activation_tokens(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_trash_expires ON trash_items(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_sync_log_user ON sync_log(user_id)`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	return nil
}
