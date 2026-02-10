// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package rclone handles cloud backup to SFTP servers using rclone.
// This module provides push-only backup of user backup directories to remote SFTP servers.
package rclone

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Provider type constants
const (
	ProviderSFTP   = "sftp"
	ProviderS3     = "s3"
	ProviderWebDAV = "webdav"
	ProviderRemote = "remote"
)

// RcloneBackup represents a cloud backup destination configuration
type RcloneBackup struct {
	ID       int
	Name     string // User-friendly name (e.g., "Backup SFTP Principal")

	// Provider type: "sftp", "s3", "webdav", "remote"
	ProviderType   string            // Provider type constant
	ProviderConfig map[string]string // Provider-specific config (JSON in DB)

	// SFTP configuration (legacy, still used for SFTP provider)
	SFTPHost     string // Hostname or IP
	SFTPPort     int    // Default 22
	SFTPUser     string // SSH username
	SFTPKeyPath  string // Path to SSH private key (optional)
	SFTPPassword string // Password (encrypted, optional)
	RemotePath   string // Remote path (e.g., /backups/anemone)

	// Options
	Enabled bool

	// Scheduling fields
	SyncEnabled         bool   // Enable automatic sync
	SyncFrequency       string // "daily", "weekly", "monthly", "interval"
	SyncTime            string // "HH:MM" format for daily/weekly/monthly
	SyncDayOfWeek       *int   // 0-6 (0=Sunday) for weekly
	SyncDayOfMonth      *int   // 1-31 for monthly
	SyncIntervalMinutes int    // Interval in minutes for interval mode

	// Status
	LastSync    *time.Time
	LastStatus  string // "success", "error", "running", "unknown"
	LastError   string
	FilesSynced int
	BytesSynced int64

	CreatedAt time.Time
	UpdatedAt time.Time
}

// DisplayHost returns a display label for the backup destination depending on provider type.
func (b *RcloneBackup) DisplayHost() string {
	switch b.ProviderType {
	case ProviderS3:
		if ep, ok := b.ProviderConfig["endpoint"]; ok && ep != "" {
			return ep
		}
		return "S3"
	case ProviderWebDAV:
		if url, ok := b.ProviderConfig["url"]; ok && url != "" {
			return url
		}
		return "WebDAV"
	case ProviderRemote:
		if name, ok := b.ProviderConfig["remote_name"]; ok && name != "" {
			return name + ":"
		}
		return "Remote"
	default:
		return b.SFTPHost
	}
}

// marshalProviderConfig serializes ProviderConfig to JSON string for DB storage.
func marshalProviderConfig(config map[string]string) string {
	if config == nil {
		return "{}"
	}
	data, err := json.Marshal(config)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// unmarshalProviderConfig deserializes JSON string from DB to ProviderConfig map.
func unmarshalProviderConfig(data string) map[string]string {
	if data == "" {
		return map[string]string{}
	}
	var config map[string]string
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return map[string]string{}
	}
	return config
}

// Create creates a new rclone backup configuration
func Create(db *sql.DB, backup *RcloneBackup) error {
	// Set defaults
	if backup.SFTPPort == 0 {
		backup.SFTPPort = 22
	}
	if backup.SyncFrequency == "" {
		backup.SyncFrequency = "daily"
	}
	if backup.SyncTime == "" {
		backup.SyncTime = "02:00"
	}
	if backup.SyncIntervalMinutes == 0 {
		backup.SyncIntervalMinutes = 60
	}
	if backup.ProviderType == "" {
		backup.ProviderType = ProviderSFTP
	}

	query := `INSERT INTO rclone_backups (
		name, sftp_host, sftp_port, sftp_user, sftp_key_path, sftp_password, remote_path,
		enabled, sync_enabled, sync_frequency, sync_time, sync_day_of_week, sync_day_of_month,
		sync_interval_minutes, provider_type, provider_config, last_status, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'unknown', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	result, err := db.Exec(query,
		backup.Name, backup.SFTPHost, backup.SFTPPort, backup.SFTPUser,
		backup.SFTPKeyPath, backup.SFTPPassword, backup.RemotePath,
		backup.Enabled, backup.SyncEnabled, backup.SyncFrequency, backup.SyncTime,
		backup.SyncDayOfWeek, backup.SyncDayOfMonth, backup.SyncIntervalMinutes,
		backup.ProviderType, marshalProviderConfig(backup.ProviderConfig),
	)
	if err != nil {
		return fmt.Errorf("failed to create rclone backup: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get rclone backup ID: %w", err)
	}

	backup.ID = int(id)
	return nil
}

// GetByID retrieves a rclone backup configuration by ID
func GetByID(db *sql.DB, id int) (*RcloneBackup, error) {
	backup := &RcloneBackup{}
	query := `SELECT id, name, sftp_host, sftp_port, sftp_user, sftp_key_path, sftp_password,
		remote_path, enabled, sync_enabled, sync_frequency, sync_time, sync_day_of_week,
		sync_day_of_month, sync_interval_minutes, last_sync, last_status, last_error,
		files_synced, bytes_synced, created_at, updated_at, provider_type, provider_config
		FROM rclone_backups WHERE id = ?`

	var syncFrequency, syncTime, lastStatus, lastError sql.NullString
	var syncDayOfWeek, syncDayOfMonth, syncIntervalMinutes sql.NullInt64
	var sftpKeyPath, sftpPassword sql.NullString
	var providerType, providerConfig sql.NullString

	err := db.QueryRow(query, id).Scan(
		&backup.ID, &backup.Name, &backup.SFTPHost, &backup.SFTPPort, &backup.SFTPUser,
		&sftpKeyPath, &sftpPassword, &backup.RemotePath, &backup.Enabled,
		&backup.SyncEnabled, &syncFrequency, &syncTime, &syncDayOfWeek,
		&syncDayOfMonth, &syncIntervalMinutes, &backup.LastSync, &lastStatus,
		&lastError, &backup.FilesSynced, &backup.BytesSynced,
		&backup.CreatedAt, &backup.UpdatedAt, &providerType, &providerConfig,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("rclone backup not found")
		}
		return nil, fmt.Errorf("failed to get rclone backup: %w", err)
	}

	// Handle nullable fields
	if sftpKeyPath.Valid {
		backup.SFTPKeyPath = sftpKeyPath.String
	}
	if sftpPassword.Valid {
		backup.SFTPPassword = sftpPassword.String
	}

	// Handle scheduling fields with defaults
	backup.SyncFrequency = "daily"
	if syncFrequency.Valid && syncFrequency.String != "" {
		backup.SyncFrequency = syncFrequency.String
	}
	backup.SyncTime = "02:00"
	if syncTime.Valid && syncTime.String != "" {
		backup.SyncTime = syncTime.String
	}
	if syncDayOfWeek.Valid {
		dow := int(syncDayOfWeek.Int64)
		backup.SyncDayOfWeek = &dow
	}
	if syncDayOfMonth.Valid {
		dom := int(syncDayOfMonth.Int64)
		backup.SyncDayOfMonth = &dom
	}
	backup.SyncIntervalMinutes = 60
	if syncIntervalMinutes.Valid {
		backup.SyncIntervalMinutes = int(syncIntervalMinutes.Int64)
	}

	backup.LastStatus = "unknown"
	if lastStatus.Valid {
		backup.LastStatus = lastStatus.String
	}
	if lastError.Valid {
		backup.LastError = lastError.String
	}

	// Handle provider fields
	backup.ProviderType = ProviderSFTP
	if providerType.Valid && providerType.String != "" {
		backup.ProviderType = providerType.String
	}
	backup.ProviderConfig = unmarshalProviderConfig(providerConfig.String)

	return backup, nil
}

// GetAll retrieves all rclone backup configurations
func GetAll(db *sql.DB) ([]*RcloneBackup, error) {
	query := `SELECT id, name, sftp_host, sftp_port, sftp_user, sftp_key_path, sftp_password,
		remote_path, enabled, sync_enabled, sync_frequency, sync_time, sync_day_of_week,
		sync_day_of_month, sync_interval_minutes, last_sync, last_status, last_error,
		files_synced, bytes_synced, created_at, updated_at, provider_type, provider_config
		FROM rclone_backups ORDER BY created_at DESC`

	return queryBackups(db, query)
}

// GetEnabled retrieves all enabled rclone backup configurations
func GetEnabled(db *sql.DB) ([]*RcloneBackup, error) {
	query := `SELECT id, name, sftp_host, sftp_port, sftp_user, sftp_key_path, sftp_password,
		remote_path, enabled, sync_enabled, sync_frequency, sync_time, sync_day_of_week,
		sync_day_of_month, sync_interval_minutes, last_sync, last_status, last_error,
		files_synced, bytes_synced, created_at, updated_at, provider_type, provider_config
		FROM rclone_backups WHERE enabled = 1 ORDER BY created_at DESC`

	return queryBackups(db, query)
}

// queryBackups executes a query and scans rows into RcloneBackup slices.
func queryBackups(db *sql.DB, query string, args ...interface{}) ([]*RcloneBackup, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query rclone backups: %w", err)
	}
	defer rows.Close()

	var backups []*RcloneBackup
	for rows.Next() {
		backup := &RcloneBackup{}
		var syncFrequency, syncTime, lastStatus, lastError sql.NullString
		var syncDayOfWeek, syncDayOfMonth, syncIntervalMinutes sql.NullInt64
		var sftpKeyPath, sftpPassword sql.NullString
		var providerType, providerConfig sql.NullString

		err := rows.Scan(
			&backup.ID, &backup.Name, &backup.SFTPHost, &backup.SFTPPort, &backup.SFTPUser,
			&sftpKeyPath, &sftpPassword, &backup.RemotePath, &backup.Enabled,
			&backup.SyncEnabled, &syncFrequency, &syncTime, &syncDayOfWeek,
			&syncDayOfMonth, &syncIntervalMinutes, &backup.LastSync, &lastStatus,
			&lastError, &backup.FilesSynced, &backup.BytesSynced,
			&backup.CreatedAt, &backup.UpdatedAt, &providerType, &providerConfig,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rclone backup: %w", err)
		}

		if sftpKeyPath.Valid {
			backup.SFTPKeyPath = sftpKeyPath.String
		}
		if sftpPassword.Valid {
			backup.SFTPPassword = sftpPassword.String
		}

		backup.SyncFrequency = "daily"
		if syncFrequency.Valid && syncFrequency.String != "" {
			backup.SyncFrequency = syncFrequency.String
		}
		backup.SyncTime = "02:00"
		if syncTime.Valid && syncTime.String != "" {
			backup.SyncTime = syncTime.String
		}
		if syncDayOfWeek.Valid {
			dow := int(syncDayOfWeek.Int64)
			backup.SyncDayOfWeek = &dow
		}
		if syncDayOfMonth.Valid {
			dom := int(syncDayOfMonth.Int64)
			backup.SyncDayOfMonth = &dom
		}
		backup.SyncIntervalMinutes = 60
		if syncIntervalMinutes.Valid {
			backup.SyncIntervalMinutes = int(syncIntervalMinutes.Int64)
		}

		backup.LastStatus = "unknown"
		if lastStatus.Valid {
			backup.LastStatus = lastStatus.String
		}
		if lastError.Valid {
			backup.LastError = lastError.String
		}

		backup.ProviderType = ProviderSFTP
		if providerType.Valid && providerType.String != "" {
			backup.ProviderType = providerType.String
		}
		backup.ProviderConfig = unmarshalProviderConfig(providerConfig.String)

		backups = append(backups, backup)
	}

	return backups, nil
}

// Update updates a rclone backup configuration
func Update(db *sql.DB, backup *RcloneBackup) error {
	query := `UPDATE rclone_backups SET
		name = ?, sftp_host = ?, sftp_port = ?, sftp_user = ?, sftp_key_path = ?,
		sftp_password = ?, remote_path = ?, enabled = ?, sync_enabled = ?,
		sync_frequency = ?, sync_time = ?, sync_day_of_week = ?, sync_day_of_month = ?,
		sync_interval_minutes = ?, provider_type = ?, provider_config = ?,
		updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

	_, err := db.Exec(query,
		backup.Name, backup.SFTPHost, backup.SFTPPort, backup.SFTPUser,
		backup.SFTPKeyPath, backup.SFTPPassword, backup.RemotePath,
		backup.Enabled, backup.SyncEnabled, backup.SyncFrequency, backup.SyncTime,
		backup.SyncDayOfWeek, backup.SyncDayOfMonth, backup.SyncIntervalMinutes,
		backup.ProviderType, marshalProviderConfig(backup.ProviderConfig),
		backup.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update rclone backup: %w", err)
	}

	return nil
}

// Delete deletes a rclone backup configuration
func Delete(db *sql.DB, id int) error {
	query := `DELETE FROM rclone_backups WHERE id = ?`
	_, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete rclone backup: %w", err)
	}
	return nil
}

// UpdateSyncStatus updates the sync status after a backup operation
func UpdateSyncStatus(db *sql.DB, id int, status string, errorMsg string, filesSynced int, bytesSynced int64) error {
	var query string
	var args []interface{}

	if status == "success" {
		query = `UPDATE rclone_backups SET last_sync = CURRENT_TIMESTAMP, last_status = ?,
			last_error = '', files_synced = ?, bytes_synced = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`
		args = []interface{}{status, filesSynced, bytesSynced, id}
	} else {
		query = `UPDATE rclone_backups SET last_status = ?, last_error = ?,
			updated_at = CURRENT_TIMESTAMP WHERE id = ?`
		args = []interface{}{status, errorMsg, id}
	}

	_, err := db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update rclone backup status: %w", err)
	}

	return nil
}

// Count returns the total number of rclone backup configurations
func Count(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM rclone_backups").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count rclone backups: %w", err)
	}
	return count, nil
}

// CountEnabled returns the number of enabled rclone backup configurations
func CountEnabled(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM rclone_backups WHERE enabled = 1").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count enabled rclone backups: %w", err)
	}
	return count, nil
}

// ShouldSync determines if this rclone backup should be synchronized based on its schedule
func (b *RcloneBackup) ShouldSync() bool {
	// Check if backup and sync are enabled
	if !b.Enabled || !b.SyncEnabled {
		return false
	}

	// First sync ever
	if b.LastSync == nil {
		return true
	}

	now := time.Now()
	lastSync := *b.LastSync

	// Parse sync time (format: "HH:MM")
	var syncHour, syncMinute int
	fmt.Sscanf(b.SyncTime, "%d:%d", &syncHour, &syncMinute)

	switch b.SyncFrequency {
	case "interval":
		// Interval-based sync: check if enough time has passed since last sync
		if b.SyncIntervalMinutes <= 0 {
			return false
		}
		interval := time.Duration(b.SyncIntervalMinutes) * time.Minute
		return now.Sub(lastSync) >= interval

	case "daily":
		// Daily sync: check if we've passed the sync time today and haven't synced today
		lastSyncDate := lastSync.Format("2006-01-02")
		todayDate := now.Format("2006-01-02")

		// If last sync was on a different day and we've passed the sync time
		if lastSyncDate != todayDate && (now.Hour() > syncHour || (now.Hour() == syncHour && now.Minute() >= syncMinute)) {
			return true
		}
		return false

	case "weekly":
		// Weekly sync: check if we're on the right day of week and past sync time
		if b.SyncDayOfWeek == nil {
			return false
		}

		currentDayOfWeek := int(now.Weekday()) // 0=Sunday, 1=Monday, ..., 6=Saturday
		if currentDayOfWeek != *b.SyncDayOfWeek {
			return false
		}

		// Check if we've passed the sync time today
		if now.Hour() < syncHour || (now.Hour() == syncHour && now.Minute() < syncMinute) {
			return false
		}

		// Check if last sync was before today
		lastSyncDate := lastSync.Format("2006-01-02")
		todayDate := now.Format("2006-01-02")
		return lastSyncDate != todayDate

	case "monthly":
		// Monthly sync: check if we're on the right day of month and past sync time
		if b.SyncDayOfMonth == nil {
			return false
		}

		if now.Day() != *b.SyncDayOfMonth {
			return false
		}

		// Check if we've passed the sync time today
		if now.Hour() < syncHour || (now.Hour() == syncHour && now.Minute() < syncMinute) {
			return false
		}

		// Check if last sync was before today
		lastSyncDate := lastSync.Format("2006-01-02")
		todayDate := now.Format("2006-01-02")
		return lastSyncDate != todayDate

	default:
		return false
	}
}

// FormatBytes formats bytes to human-readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
