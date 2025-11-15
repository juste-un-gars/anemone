// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package backup

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// ServerBackup represents the complete server configuration
type ServerBackup struct {
	Version       string          `json:"version"`
	ExportedAt    time.Time       `json:"exported_at"`
	ServerName    string          `json:"server_name"`
	SystemConfig  []ConfigItem    `json:"system_config"`
	Users         []UserBackup    `json:"users"`
	Shares        []ShareBackup   `json:"shares"`
	Peers         []PeerBackup    `json:"peers"`
	SyncConfig    *SyncConfig     `json:"sync_config"`
}

// ConfigItem represents a key-value pair from system_config
type ConfigItem struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserBackup represents a user with all their data
type UserBackup struct {
	ID                     int        `json:"id"`
	Username               string     `json:"username"`
	PasswordHash           string     `json:"password_hash"`
	PasswordEncrypted      []byte     `json:"password_encrypted"`
	Email                  string     `json:"email"`
	EncryptionKeyHash      string     `json:"encryption_key_hash"`
	EncryptionKeyEncrypted string     `json:"encryption_key_encrypted"` // String because stored as TEXT (base64) in DB
	IsAdmin                bool       `json:"is_admin"`
	QuotaTotalGB           int        `json:"quota_total_gb"`
	QuotaBackupGB          int        `json:"quota_backup_gb"`
	Language               string     `json:"language"`
	CreatedAt              time.Time  `json:"created_at"`
	ActivatedAt            *time.Time `json:"activated_at"`
}

// ShareBackup represents a share configuration
type ShareBackup struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Protocol    string    `json:"protocol"`
	SyncEnabled bool      `json:"sync_enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

// PeerBackup represents a peer server configuration
type PeerBackup struct {
	ID                  int        `json:"id"`
	Name                string     `json:"name"`
	Address             string     `json:"address"`
	Port                int        `json:"port"`
	PublicKey           string     `json:"public_key"`
	Password            string     `json:"password"`
	Enabled             bool       `json:"enabled"`
	Status              string     `json:"status"`
	SyncEnabled         bool       `json:"sync_enabled"`
	SyncFrequency       string     `json:"sync_frequency"`
	SyncTime            string     `json:"sync_time"`
	SyncDayOfWeek       *int       `json:"sync_day_of_week"`
	SyncDayOfMonth      *int       `json:"sync_day_of_month"`
	SyncIntervalMinutes int        `json:"sync_interval_minutes"`
	LastSeen            *time.Time `json:"last_seen"`
	LastSync            *time.Time `json:"last_sync"`
	CreatedAt           time.Time  `json:"created_at"`
}

// SyncConfig represents the sync configuration
type SyncConfig struct {
	ID        int        `json:"id"`
	Enabled   bool       `json:"enabled"`
	Interval  string     `json:"interval"`
	FixedHour int        `json:"fixed_hour"`
	LastSync  *time.Time `json:"last_sync"`
}

// ExportConfiguration exports the complete server configuration to JSON
func ExportConfiguration(db *sql.DB, serverName string) (*ServerBackup, error) {
	backup := &ServerBackup{
		Version:    "1.0",
		ExportedAt: time.Now(),
		ServerName: serverName,
	}

	// Export system_config
	configRows, err := db.Query("SELECT key, value, updated_at FROM system_config")
	if err != nil {
		return nil, fmt.Errorf("failed to query system_config: %w", err)
	}
	defer configRows.Close()

	for configRows.Next() {
		var item ConfigItem
		if err := configRows.Scan(&item.Key, &item.Value, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan system_config row: %w", err)
		}
		backup.SystemConfig = append(backup.SystemConfig, item)
	}

	// Export users
	userRows, err := db.Query(`SELECT id, username, password_hash, password_encrypted, email, encryption_key_hash,
		encryption_key_encrypted, is_admin, quota_total_gb, quota_backup_gb, language,
		created_at, activated_at FROM users`)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer userRows.Close()

	for userRows.Next() {
		var user UserBackup
		var email, language sql.NullString
		var activatedAt sql.NullTime
		var passwordEncrypted []byte
		if err := userRows.Scan(&user.ID, &user.Username, &user.PasswordHash, &passwordEncrypted, &email,
			&user.EncryptionKeyHash, &user.EncryptionKeyEncrypted, &user.IsAdmin,
			&user.QuotaTotalGB, &user.QuotaBackupGB, &language, &user.CreatedAt, &activatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		user.PasswordEncrypted = passwordEncrypted
		if email.Valid {
			user.Email = email.String
		}
		if language.Valid {
			user.Language = language.String
		} else {
			user.Language = "fr"
		}
		if activatedAt.Valid {
			user.ActivatedAt = &activatedAt.Time
		}
		backup.Users = append(backup.Users, user)
	}

	// Export shares
	shareRows, err := db.Query(`SELECT id, user_id, name, path, protocol, sync_enabled, created_at FROM shares`)
	if err != nil {
		return nil, fmt.Errorf("failed to query shares: %w", err)
	}
	defer shareRows.Close()

	for shareRows.Next() {
		var share ShareBackup
		if err := shareRows.Scan(&share.ID, &share.UserID, &share.Name, &share.Path,
			&share.Protocol, &share.SyncEnabled, &share.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan share row: %w", err)
		}
		backup.Shares = append(backup.Shares, share)
	}

	// Export peers
	peerRows, err := db.Query(`SELECT id, name, address, port, public_key, password, enabled, status,
		sync_enabled, sync_frequency, sync_time, sync_day_of_week, sync_day_of_month, sync_interval_minutes,
		last_seen, last_sync, created_at FROM peers`)
	if err != nil {
		return nil, fmt.Errorf("failed to query peers: %w", err)
	}
	defer peerRows.Close()

	for peerRows.Next() {
		var peer PeerBackup
		var publicKey, password sql.NullString
		var dayOfWeek, dayOfMonth sql.NullInt64
		var lastSeen, lastSync sql.NullTime
		if err := peerRows.Scan(&peer.ID, &peer.Name, &peer.Address, &peer.Port, &publicKey, &password,
			&peer.Enabled, &peer.Status, &peer.SyncEnabled, &peer.SyncFrequency, &peer.SyncTime,
			&dayOfWeek, &dayOfMonth, &peer.SyncIntervalMinutes, &lastSeen, &lastSync, &peer.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan peer row: %w", err)
		}
		if publicKey.Valid {
			peer.PublicKey = publicKey.String
		}
		if password.Valid {
			peer.Password = password.String
		}
		if dayOfWeek.Valid {
			day := int(dayOfWeek.Int64)
			peer.SyncDayOfWeek = &day
		}
		if dayOfMonth.Valid {
			day := int(dayOfMonth.Int64)
			peer.SyncDayOfMonth = &day
		}
		if lastSeen.Valid {
			peer.LastSeen = &lastSeen.Time
		}
		if lastSync.Valid {
			peer.LastSync = &lastSync.Time
		}
		backup.Peers = append(backup.Peers, peer)
	}

	// Export sync_config
	var syncConfig SyncConfig
	var lastSync sql.NullTime
	err = db.QueryRow(`SELECT id, enabled, interval, fixed_hour, last_sync FROM sync_config WHERE id = 1`).
		Scan(&syncConfig.ID, &syncConfig.Enabled, &syncConfig.Interval, &syncConfig.FixedHour, &lastSync)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query sync_config: %w", err)
	}
	if err == nil {
		if lastSync.Valid {
			syncConfig.LastSync = &lastSync.Time
		}
		backup.SyncConfig = &syncConfig
	}

	return backup, nil
}

// EncryptBackup encrypts the backup JSON with AES-256-GCM
func EncryptBackup(backup *ServerBackup, passphrase string) ([]byte, error) {
	// Serialize to JSON
	jsonData, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backup: %w", err)
	}

	// Derive key from passphrase using SHA-256
	keyHash := sha256.Sum256([]byte(passphrase))
	key := keyHash[:]

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt data
	ciphertext := gcm.Seal(nonce, nonce, jsonData, nil)

	return ciphertext, nil
}

// DecryptBackup decrypts the encrypted backup data
func DecryptBackup(encryptedData []byte, passphrase string) (*ServerBackup, error) {
	// Derive key from passphrase using SHA-256
	keyHash := sha256.Sum256([]byte(passphrase))
	key := keyHash[:]

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Check minimum size
	nonceSize := gcm.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, fmt.Errorf("encrypted data too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]

	// Decrypt data
	jsonData, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w (wrong passphrase?)", err)
	}

	// Unmarshal JSON
	var backup ServerBackup
	if err := json.Unmarshal(jsonData, &backup); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup: %w", err)
	}

	return &backup, nil
}

// EncodeToBase64 encodes encrypted data to base64 for easier storage
func EncodeToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeFromBase64 decodes base64 data
func DecodeFromBase64(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}
