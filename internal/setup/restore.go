// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package setup

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/juste-un-gars/anemone/internal/backup"
	"github.com/juste-un-gars/anemone/internal/crypto"
	"github.com/juste-un-gars/anemone/internal/database"
)

// RestoreResult contains the result of a backup validation
type RestoreResult struct {
	Valid       bool              `json:"valid"`
	ServerName  string            `json:"server_name"`
	ExportedAt  string            `json:"exported_at"`
	Version     string            `json:"version"`
	UsersCount  int               `json:"users_count"`
	SharesCount int               `json:"shares_count"`
	PeersCount  int               `json:"peers_count"`
	Users       []RestoreUser     `json:"users"`
	Peers       []RestorePeer     `json:"peers"`
	Error       string            `json:"error,omitempty"`
}

// RestoreUser is a simplified user for display
type RestoreUser struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	IsAdmin  bool   `json:"is_admin"`
}

// RestorePeer is a simplified peer for display
type RestorePeer struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}

// ValidateBackup validates and decrypts a backup file
func ValidateBackup(encryptedData []byte, passphrase string) (*RestoreResult, *backup.ServerBackup, error) {
	result := &RestoreResult{}

	// Try to decrypt
	serverBackup, err := backup.DecryptBackup(encryptedData, passphrase)
	if err != nil {
		result.Valid = false
		result.Error = "invalid_passphrase"
		return result, nil, fmt.Errorf("failed to decrypt backup: %w", err)
	}

	// Populate result
	result.Valid = true
	result.ServerName = serverBackup.ServerName
	result.ExportedAt = serverBackup.ExportedAt.Format("2006-01-02 15:04")
	result.Version = serverBackup.Version
	result.UsersCount = len(serverBackup.Users)
	result.SharesCount = len(serverBackup.Shares)
	result.PeersCount = len(serverBackup.Peers)

	// Extract user info
	for _, u := range serverBackup.Users {
		result.Users = append(result.Users, RestoreUser{
			Username: u.Username,
			Email:    u.Email,
			IsAdmin:  u.IsAdmin,
		})
	}

	// Extract peer info
	for _, p := range serverBackup.Peers {
		result.Peers = append(result.Peers, RestorePeer{
			Name:    p.Name,
			Address: p.Address,
			Port:    p.Port,
		})
	}

	return result, serverBackup, nil
}

// RestoreOptions contains options for restoring a backup
type RestoreOptions struct {
	DataDir     string
	SharesDir   string
	IncomingDir string
}

// ExecuteRestore restores a backup to the database
func ExecuteRestore(serverBackup *backup.ServerBackup, opts RestoreOptions) error {
	// Ensure directories exist
	dbDir := filepath.Join(opts.DataDir, "db")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Initialize database
	dbPath := filepath.Join(dbDir, "anemone.db")

	// Remove existing database if present
	if _, err := os.Stat(dbPath); err == nil {
		if err := os.Remove(dbPath); err != nil {
			return fmt.Errorf("failed to remove existing database: %w", err)
		}
	}

	db, err := database.Init(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Restore in transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Restore system_config
	if err := restoreSystemConfig(tx, serverBackup.SystemConfig); err != nil {
		return fmt.Errorf("failed to restore system config: %w", err)
	}

	// Get master key for encrypting peer passwords
	var masterKey string
	for _, cfg := range serverBackup.SystemConfig {
		if cfg.Key == "master_key" {
			masterKey = cfg.Value
			break
		}
	}

	// 2. Restore users
	if err := restoreUsers(tx, serverBackup.Users); err != nil {
		return fmt.Errorf("failed to restore users: %w", err)
	}

	// 3. Restore shares
	if err := restoreShares(tx, serverBackup.Shares, opts.SharesDir); err != nil {
		return fmt.Errorf("failed to restore shares: %w", err)
	}

	// 4. Restore peers
	if err := restorePeers(tx, serverBackup.Peers, masterKey); err != nil {
		return fmt.Errorf("failed to restore peers: %w", err)
	}

	// 5. Restore sync_config
	if serverBackup.SyncConfig != nil {
		if err := restoreSyncConfig(tx, serverBackup.SyncConfig); err != nil {
			return fmt.Errorf("failed to restore sync config: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func restoreSystemConfig(tx *sql.Tx, configs []backup.ConfigItem) error {
	for _, cfg := range configs {
		_, err := tx.Exec(
			`INSERT OR REPLACE INTO system_config (key, value, updated_at) VALUES (?, ?, ?)`,
			cfg.Key, cfg.Value, cfg.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to restore config %s: %w", cfg.Key, err)
		}
	}
	return nil
}

func restoreUsers(tx *sql.Tx, users []backup.UserBackup) error {
	for _, u := range users {
		var activatedAt interface{}
		if u.ActivatedAt != nil {
			activatedAt = *u.ActivatedAt
		}

		_, err := tx.Exec(
			`INSERT INTO users (id, username, password_hash, password_encrypted, email,
				encryption_key_hash, encryption_key_encrypted, is_admin,
				quota_total_gb, quota_backup_gb, language, created_at, activated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			u.ID, u.Username, u.PasswordHash, u.PasswordEncrypted, nullString(u.Email),
			u.EncryptionKeyHash, nullString(u.EncryptionKeyEncrypted), u.IsAdmin,
			u.QuotaTotalGB, u.QuotaBackupGB, u.Language, u.CreatedAt, activatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to restore user %s: %w", u.Username, err)
		}
	}
	return nil
}

func restoreShares(tx *sql.Tx, shares []backup.ShareBackup, sharesDir string) error {
	for _, s := range shares {
		// Update path if sharesDir is different
		path := s.Path
		if sharesDir != "" {
			// Extract relative path and rebuild with new sharesDir
			// Original path format: /srv/anemone/shares/username/sharename
			// We want to replace the base with new sharesDir
			baseName := filepath.Base(filepath.Dir(s.Path)) // username
			shareName := filepath.Base(s.Path)              // sharename
			path = filepath.Join(sharesDir, baseName, shareName)
		}

		_, err := tx.Exec(
			`INSERT INTO shares (id, user_id, name, path, protocol, sync_enabled, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			s.ID, s.UserID, s.Name, path, s.Protocol, s.SyncEnabled, s.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to restore share %s: %w", s.Name, err)
		}
	}
	return nil
}

func restorePeers(tx *sql.Tx, peers []backup.PeerBackup, masterKey string) error {
	for _, p := range peers {
		// Re-encrypt password with master key
		var encryptedPassword []byte
		if p.Password != "" && masterKey != "" {
			var err error
			encryptedPassword, err = crypto.EncryptPassword(p.Password, masterKey)
			if err != nil {
				return fmt.Errorf("failed to encrypt password for peer %s: %w", p.Name, err)
			}
		}

		var lastSeen, lastSync interface{}
		if p.LastSeen != nil {
			lastSeen = *p.LastSeen
		}
		if p.LastSync != nil {
			lastSync = *p.LastSync
		}

		_, err := tx.Exec(
			`INSERT INTO peers (id, name, address, port, public_key, password, enabled, status,
				sync_enabled, sync_frequency, sync_time, sync_day_of_week, sync_day_of_month,
				sync_interval_minutes, last_seen, last_sync, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.ID, p.Name, p.Address, p.Port, nullString(p.PublicKey), encryptedPassword,
			p.Enabled, p.Status, p.SyncEnabled, p.SyncFrequency, p.SyncTime,
			nullInt(p.SyncDayOfWeek), nullInt(p.SyncDayOfMonth), p.SyncIntervalMinutes,
			lastSeen, lastSync, p.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to restore peer %s: %w", p.Name, err)
		}
	}
	return nil
}

func restoreSyncConfig(tx *sql.Tx, cfg *backup.SyncConfig) error {
	var lastSync interface{}
	if cfg.LastSync != nil {
		lastSync = *cfg.LastSync
	}

	_, err := tx.Exec(
		`INSERT OR REPLACE INTO sync_config (id, enabled, interval, fixed_hour, last_sync)
		VALUES (?, ?, ?, ?, ?)`,
		cfg.ID, cfg.Enabled, cfg.Interval, cfg.FixedHour, lastSync,
	)
	return err
}

// Helper functions
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullInt(i *int) interface{} {
	if i == nil {
		return nil
	}
	return *i
}
