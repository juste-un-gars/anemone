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
			address TEXT NOT NULL,
			port INTEGER DEFAULT 8443,
			public_key TEXT,
			enabled BOOLEAN DEFAULT 1,
			status TEXT DEFAULT 'unknown',
			last_seen DATETIME,
			last_sync DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
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

	// Migration pour ajouter les nouvelles colonnes à la table peers si elles n'existent pas
	if err := migratePeersTable(db); err != nil {
		return fmt.Errorf("peers table migration failed: %w", err)
	}

	// Migration pour ajouter la colonne language à la table users
	if err := migrateUsersTable(db); err != nil {
		return fmt.Errorf("users table migration failed: %w", err)
	}

	return nil
}

// migratePeersTable adds missing columns to peers table
func migratePeersTable(db *sql.DB) error {
	// Check which columns exist
	rows, err := db.Query("PRAGMA table_info(peers)")
	if err != nil {
		return err
	}
	defer rows.Close()

	existingColumns := make(map[string]bool)
	hasUrlColumn := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		existingColumns[name] = true
		if name == "url" {
			hasUrlColumn = true
		}
	}

	// Add missing columns
	columnsToAdd := map[string]string{
		"address":    "ALTER TABLE peers ADD COLUMN address TEXT DEFAULT ''",
		"port":       "ALTER TABLE peers ADD COLUMN port INTEGER DEFAULT 8443",
		"public_key": "ALTER TABLE peers ADD COLUMN public_key TEXT",
		"enabled":    "ALTER TABLE peers ADD COLUMN enabled BOOLEAN DEFAULT 1",
		"status":     "ALTER TABLE peers ADD COLUMN status TEXT DEFAULT 'unknown'",
		"last_seen":  "ALTER TABLE peers ADD COLUMN last_seen DATETIME",
		"last_sync":  "ALTER TABLE peers ADD COLUMN last_sync DATETIME",
		"updated_at": "ALTER TABLE peers ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP",
	}

	for column, query := range columnsToAdd {
		if !existingColumns[column] {
			if _, err := db.Exec(query); err != nil {
				return fmt.Errorf("failed to add column %s: %w", column, err)
			}
		}
	}

	// If old 'url' column exists, we need to recreate the table without it
	// SQLite doesn't support DROP COLUMN in older versions, so we recreate
	if hasUrlColumn {
		// Create new table with correct schema
		_, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS peers_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT UNIQUE NOT NULL,
				address TEXT NOT NULL,
				port INTEGER DEFAULT 8443,
				public_key TEXT,
				enabled BOOLEAN DEFAULT 1,
				status TEXT DEFAULT 'unknown',
				last_seen DATETIME,
				last_sync DATETIME,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			return fmt.Errorf("failed to create peers_new table: %w", err)
		}

		// Copy data from old table, using address and port from url if needed
		_, err = db.Exec(`
			INSERT INTO peers_new (id, name, address, port, public_key, enabled, status, last_seen, last_sync, created_at, updated_at)
			SELECT id, name,
				CASE WHEN address = '' OR address IS NULL THEN '' ELSE address END,
				CASE WHEN port IS NULL THEN 8443 ELSE port END,
				public_key, enabled, status, last_seen, last_sync, created_at, updated_at
			FROM peers
		`)
		if err != nil {
			return fmt.Errorf("failed to copy peers data: %w", err)
		}

		// Drop old table and rename new one
		_, err = db.Exec("DROP TABLE peers")
		if err != nil {
			return fmt.Errorf("failed to drop old peers table: %w", err)
		}

		_, err = db.Exec("ALTER TABLE peers_new RENAME TO peers")
		if err != nil {
			return fmt.Errorf("failed to rename peers_new table: %w", err)
		}
	}

	return nil
}

// migrateUsersTable adds missing columns to users table
func migrateUsersTable(db *sql.DB) error {
	// Check which columns exist
	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		return err
	}
	defer rows.Close()

	existingColumns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		existingColumns[name] = true
	}

	// Add language column if it doesn't exist
	if !existingColumns["language"] {
		if _, err := db.Exec("ALTER TABLE users ADD COLUMN language VARCHAR(2) DEFAULT 'fr'"); err != nil {
			return fmt.Errorf("failed to add language column: %w", err)
		}
	}

	return nil
}
