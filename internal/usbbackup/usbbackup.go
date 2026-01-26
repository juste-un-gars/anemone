// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package usbbackup handles local backup to USB drives and external storage.
// This is a separate module from P2P sync, designed for simple local backups.
package usbbackup

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BackupType constants
const (
	BackupTypeConfig = "config" // Config only: DB, certs, smb.conf
	BackupTypeFull   = "full"   // Config + selected shares data
)

// USBBackup represents a USB/external drive backup configuration
type USBBackup struct {
	ID             int
	Name           string     // User-friendly name (e.g., "USB Backup Drive")
	MountPath      string     // Mount point (e.g., "/media/usb-backup")
	BackupPath     string     // Subdirectory for backups (e.g., "anemone-backup")
	BackupType     string     // "config" or "full"
	SelectedShares string     // JSON array of share IDs (empty = all with sync_enabled)
	Enabled        bool       // Whether this backup is active
	AutoDetect     bool       // Auto-start backup when drive is mounted
	LastSync       *time.Time // Last successful backup
	LastStatus     string     // "success", "error", "running", "unknown"
	LastError      string     // Last error message if any
	FilesSynced    int        // Files synced in last backup
	BytesSynced    int64      // Bytes synced in last backup
	CreatedAt      time.Time
	UpdatedAt      time.Time

	// Scheduling fields
	SyncEnabled         bool   // Enable automatic sync
	SyncFrequency       string // "daily", "weekly", "monthly", "interval"
	SyncTime            string // "HH:MM" format for daily/weekly/monthly
	SyncDayOfWeek       *int   // 0-6 (0=Sunday) for weekly
	SyncDayOfMonth      *int   // 1-31 for monthly
	SyncIntervalMinutes int    // Interval in minutes for interval mode
}

// DriveInfo represents detected USB/external drive information
type DriveInfo struct {
	MountPath   string
	DevicePath  string // e.g., /dev/sdb1
	Label       string // Drive label if available
	Filesystem  string // e.g., ext4, ntfs, vfat
	TotalBytes  int64
	FreeBytes   int64
	IsRemovable bool
}

// Create creates a new USB backup configuration
func Create(db *sql.DB, backup *USBBackup) error {
	// Default to full backup if not specified
	if backup.BackupType == "" {
		backup.BackupType = BackupTypeFull
	}
	// Default sync frequency
	if backup.SyncFrequency == "" {
		backup.SyncFrequency = "daily"
	}
	if backup.SyncTime == "" {
		backup.SyncTime = "23:00"
	}
	if backup.SyncIntervalMinutes == 0 {
		backup.SyncIntervalMinutes = 60
	}

	query := `INSERT INTO usb_backups (name, mount_path, backup_path, backup_type, selected_shares,
	          enabled, auto_detect, sync_enabled, sync_frequency, sync_time,
	          sync_day_of_week, sync_day_of_month, sync_interval_minutes,
	          last_status, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'unknown', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	result, err := db.Exec(query, backup.Name, backup.MountPath, backup.BackupPath,
		backup.BackupType, backup.SelectedShares, backup.Enabled, backup.AutoDetect,
		backup.SyncEnabled, backup.SyncFrequency, backup.SyncTime,
		backup.SyncDayOfWeek, backup.SyncDayOfMonth, backup.SyncIntervalMinutes)
	if err != nil {
		return fmt.Errorf("failed to create USB backup: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get USB backup ID: %w", err)
	}

	backup.ID = int(id)
	return nil
}

// GetByID retrieves a USB backup configuration by ID
func GetByID(db *sql.DB, id int) (*USBBackup, error) {
	backup := &USBBackup{}
	query := `SELECT id, name, mount_path, backup_path, backup_type, selected_shares,
	          enabled, auto_detect, last_sync, last_status, last_error, files_synced, bytes_synced,
	          sync_enabled, sync_frequency, sync_time, sync_day_of_week, sync_day_of_month, sync_interval_minutes,
	          created_at, updated_at
	          FROM usb_backups WHERE id = ?`

	var backupType, selectedShares, syncFrequency, syncTime sql.NullString
	var syncDayOfWeek, syncDayOfMonth, syncIntervalMinutes sql.NullInt64
	err := db.QueryRow(query, id).Scan(
		&backup.ID, &backup.Name, &backup.MountPath, &backup.BackupPath,
		&backupType, &selectedShares,
		&backup.Enabled, &backup.AutoDetect, &backup.LastSync, &backup.LastStatus,
		&backup.LastError, &backup.FilesSynced, &backup.BytesSynced,
		&backup.SyncEnabled, &syncFrequency, &syncTime, &syncDayOfWeek, &syncDayOfMonth, &syncIntervalMinutes,
		&backup.CreatedAt, &backup.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("USB backup not found")
		}
		return nil, fmt.Errorf("failed to get USB backup: %w", err)
	}

	// Handle nullable fields with defaults
	backup.BackupType = BackupTypeFull
	if backupType.Valid && backupType.String != "" {
		backup.BackupType = backupType.String
	}
	if selectedShares.Valid {
		backup.SelectedShares = selectedShares.String
	}

	// Handle scheduling fields with defaults
	backup.SyncFrequency = "daily"
	if syncFrequency.Valid && syncFrequency.String != "" {
		backup.SyncFrequency = syncFrequency.String
	}
	backup.SyncTime = "23:00"
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

	return backup, nil
}

// GetAll retrieves all USB backup configurations
func GetAll(db *sql.DB) ([]*USBBackup, error) {
	query := `SELECT id, name, mount_path, backup_path, backup_type, selected_shares,
	          enabled, auto_detect, last_sync, last_status, last_error, files_synced, bytes_synced,
	          sync_enabled, sync_frequency, sync_time, sync_day_of_week, sync_day_of_month, sync_interval_minutes,
	          created_at, updated_at
	          FROM usb_backups ORDER BY created_at DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query USB backups: %w", err)
	}
	defer rows.Close()

	var backups []*USBBackup
	for rows.Next() {
		backup := &USBBackup{}
		var backupType, selectedShares, syncFrequency, syncTime sql.NullString
		var syncDayOfWeek, syncDayOfMonth, syncIntervalMinutes sql.NullInt64
		err := rows.Scan(
			&backup.ID, &backup.Name, &backup.MountPath, &backup.BackupPath,
			&backupType, &selectedShares,
			&backup.Enabled, &backup.AutoDetect, &backup.LastSync, &backup.LastStatus,
			&backup.LastError, &backup.FilesSynced, &backup.BytesSynced,
			&backup.SyncEnabled, &syncFrequency, &syncTime, &syncDayOfWeek, &syncDayOfMonth, &syncIntervalMinutes,
			&backup.CreatedAt, &backup.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan USB backup: %w", err)
		}

		// Handle nullable fields with defaults
		backup.BackupType = BackupTypeFull
		if backupType.Valid && backupType.String != "" {
			backup.BackupType = backupType.String
		}
		if selectedShares.Valid {
			backup.SelectedShares = selectedShares.String
		}

		// Handle scheduling fields with defaults
		backup.SyncFrequency = "daily"
		if syncFrequency.Valid && syncFrequency.String != "" {
			backup.SyncFrequency = syncFrequency.String
		}
		backup.SyncTime = "23:00"
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

		backups = append(backups, backup)
	}

	return backups, nil
}

// GetEnabled retrieves all enabled USB backup configurations
func GetEnabled(db *sql.DB) ([]*USBBackup, error) {
	query := `SELECT id, name, mount_path, backup_path, backup_type, selected_shares,
	          enabled, auto_detect, last_sync, last_status, last_error, files_synced, bytes_synced,
	          sync_enabled, sync_frequency, sync_time, sync_day_of_week, sync_day_of_month, sync_interval_minutes,
	          created_at, updated_at
	          FROM usb_backups WHERE enabled = 1 ORDER BY created_at DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query enabled USB backups: %w", err)
	}
	defer rows.Close()

	var backups []*USBBackup
	for rows.Next() {
		backup := &USBBackup{}
		var backupType, selectedShares, syncFrequency, syncTime sql.NullString
		var syncDayOfWeek, syncDayOfMonth, syncIntervalMinutes sql.NullInt64
		err := rows.Scan(
			&backup.ID, &backup.Name, &backup.MountPath, &backup.BackupPath,
			&backupType, &selectedShares,
			&backup.Enabled, &backup.AutoDetect, &backup.LastSync, &backup.LastStatus,
			&backup.LastError, &backup.FilesSynced, &backup.BytesSynced,
			&backup.SyncEnabled, &syncFrequency, &syncTime, &syncDayOfWeek, &syncDayOfMonth, &syncIntervalMinutes,
			&backup.CreatedAt, &backup.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan USB backup: %w", err)
		}

		// Handle nullable fields with defaults
		backup.BackupType = BackupTypeFull
		if backupType.Valid && backupType.String != "" {
			backup.BackupType = backupType.String
		}
		if selectedShares.Valid {
			backup.SelectedShares = selectedShares.String
		}

		// Handle scheduling fields with defaults
		backup.SyncFrequency = "daily"
		if syncFrequency.Valid && syncFrequency.String != "" {
			backup.SyncFrequency = syncFrequency.String
		}
		backup.SyncTime = "23:00"
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

		backups = append(backups, backup)
	}

	return backups, nil
}

// Update updates a USB backup configuration
func Update(db *sql.DB, backup *USBBackup) error {
	query := `UPDATE usb_backups SET name = ?, mount_path = ?, backup_path = ?,
	          backup_type = ?, selected_shares = ?,
	          enabled = ?, auto_detect = ?,
	          sync_enabled = ?, sync_frequency = ?, sync_time = ?,
	          sync_day_of_week = ?, sync_day_of_month = ?, sync_interval_minutes = ?,
	          updated_at = CURRENT_TIMESTAMP
	          WHERE id = ?`

	_, err := db.Exec(query, backup.Name, backup.MountPath, backup.BackupPath,
		backup.BackupType, backup.SelectedShares,
		backup.Enabled, backup.AutoDetect,
		backup.SyncEnabled, backup.SyncFrequency, backup.SyncTime,
		backup.SyncDayOfWeek, backup.SyncDayOfMonth, backup.SyncIntervalMinutes,
		backup.ID)
	if err != nil {
		return fmt.Errorf("failed to update USB backup: %w", err)
	}

	return nil
}

// Delete deletes a USB backup configuration
func Delete(db *sql.DB, id int) error {
	query := `DELETE FROM usb_backups WHERE id = ?`
	_, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete USB backup: %w", err)
	}

	return nil
}

// UpdateSyncStatus updates the sync status after a backup operation
func UpdateSyncStatus(db *sql.DB, id int, status string, errorMsg string, filesSynced int, bytesSynced int64) error {
	var query string
	var args []interface{}

	if status == "success" {
		query = `UPDATE usb_backups SET last_sync = CURRENT_TIMESTAMP, last_status = ?,
		         last_error = '', files_synced = ?, bytes_synced = ?, updated_at = CURRENT_TIMESTAMP
		         WHERE id = ?`
		args = []interface{}{status, filesSynced, bytesSynced, id}
	} else {
		query = `UPDATE usb_backups SET last_status = ?, last_error = ?,
		         updated_at = CURRENT_TIMESTAMP WHERE id = ?`
		args = []interface{}{status, errorMsg, id}
	}

	_, err := db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update USB backup status: %w", err)
	}

	return nil
}

// IsMounted checks if the backup drive is currently mounted
func (b *USBBackup) IsMounted() bool {
	info, err := os.Stat(b.MountPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetFullBackupPath returns the complete path for backups on this drive
func (b *USBBackup) GetFullBackupPath() string {
	if b.BackupPath == "" {
		return b.MountPath
	}
	return filepath.Join(b.MountPath, b.BackupPath)
}

// EnsureBackupDir creates the backup directory if it doesn't exist
func (b *USBBackup) EnsureBackupDir() error {
	path := b.GetFullBackupPath()
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	return nil
}

// Count returns the total number of USB backup configurations
func Count(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM usb_backups").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count USB backups: %w", err)
	}
	return count, nil
}

// CountEnabled returns the number of enabled USB backup configurations
func CountEnabled(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM usb_backups WHERE enabled = 1").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count enabled USB backups: %w", err)
	}
	return count, nil
}

// GetSelectedShareIDs returns the list of selected share IDs from the JSON string
// Returns empty slice if no shares are selected (meaning all shares with sync_enabled)
func (b *USBBackup) GetSelectedShareIDs() []int {
	if b.SelectedShares == "" {
		return []int{}
	}

	var ids []int
	if err := json.Unmarshal([]byte(b.SelectedShares), &ids); err != nil {
		return []int{}
	}
	return ids
}

// SetSelectedShareIDs sets the selected shares from a slice of IDs
func (b *USBBackup) SetSelectedShareIDs(ids []int) {
	if len(ids) == 0 {
		b.SelectedShares = ""
		return
	}

	data, err := json.Marshal(ids)
	if err != nil {
		b.SelectedShares = ""
		return
	}
	b.SelectedShares = string(data)
}

// IsShareSelected checks if a specific share ID is selected for backup
// Returns true if no shares are explicitly selected (all shares mode)
func (b *USBBackup) IsShareSelected(shareID int) bool {
	ids := b.GetSelectedShareIDs()
	if len(ids) == 0 {
		return true // No selection = all shares
	}

	for _, id := range ids {
		if id == shareID {
			return true
		}
	}
	return false
}

// IsConfigOnly returns true if this is a config-only backup
func (b *USBBackup) IsConfigOnly() bool {
	return b.BackupType == BackupTypeConfig
}

// ShouldSync determines if this USB backup should be synchronized based on its schedule
func (b *USBBackup) ShouldSync() bool {
	// Check if backup and sync are enabled
	if !b.Enabled || !b.SyncEnabled {
		return false
	}

	// Check if drive is mounted
	if !b.IsMounted() {
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
