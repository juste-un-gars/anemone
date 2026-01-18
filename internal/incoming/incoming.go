// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package incoming manages incoming backups received from remote P2P peers.
package incoming

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// IncomingBackup represents a backup stored on this server from a remote peer
type IncomingBackup struct {
	UserID       int       `json:"user_id"`
	Username     string    `json:"username"`
	ShareName    string    `json:"share_name"`
	SourceServer string    `json:"source_server"` // Name of the remote server that sent this backup
	Path         string    `json:"path"`
	FileCount    int       `json:"file_count"`
	TotalSize    int64     `json:"total_size"`
	LastModified time.Time `json:"last_modified"`
	HasManifest  bool      `json:"has_manifest"`
}

// ScanIncomingBackups scans the /srv/anemone/backups/incoming/ directory
// and returns information about all backups stored on this server
// Directory structure: /srv/anemone/backups/incoming/{source_server}/{user_id}_{share_name}/
func ScanIncomingBackups(db *sql.DB, backupsDir string) ([]*IncomingBackup, error) {
	// Check if backups directory exists
	if _, err := os.Stat(backupsDir); os.IsNotExist(err) {
		// Directory doesn't exist - no backups yet
		return []*IncomingBackup{}, nil
	}

	// Read all source server directories in backups/incoming/
	serverEntries, err := os.ReadDir(backupsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backups directory: %w", err)
	}

	var backups []*IncomingBackup

	// Iterate over each source server directory
	for _, serverEntry := range serverEntries {
		if !serverEntry.IsDir() {
			continue
		}

		sourceServer := serverEntry.Name()
		serverDir := filepath.Join(backupsDir, sourceServer)

		// Read all backup directories for this source server
		backupEntries, err := os.ReadDir(serverDir)
		if err != nil {
			continue // Skip if we can't read this server's directory
		}

		for _, entry := range backupEntries {
			if !entry.IsDir() {
				continue
			}

			// Parse directory name: {user_id}_{share_name}
			parts := strings.SplitN(entry.Name(), "_", 2)
			if len(parts) != 2 {
				continue // Invalid format
			}

			var userID int
			if _, err := fmt.Sscanf(parts[0], "%d", &userID); err != nil {
				continue // Invalid user_id
			}
			shareName := parts[1]

			// Get username from database
			username := "Unknown"
			err := db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
			if err != nil && err != sql.ErrNoRows {
				// Continue even if user doesn't exist locally
				username = fmt.Sprintf("User #%d", userID)
			}

			// Scan directory for files and stats
			backupPath := filepath.Join(serverDir, entry.Name())
			fileCount, totalSize, lastModified, hasManifest, err := scanBackupDir(backupPath)
			if err != nil {
				return nil, fmt.Errorf("failed to scan backup %s: %w", entry.Name(), err)
			}

			backup := &IncomingBackup{
				UserID:       userID,
				Username:     username,
				ShareName:    shareName,
				SourceServer: sourceServer,
				Path:         backupPath,
				FileCount:    fileCount,
				TotalSize:    totalSize,
				LastModified: lastModified,
				HasManifest:  hasManifest,
			}

			backups = append(backups, backup)
		}
	}

	return backups, nil
}

// scanBackupDir scans a backup directory and returns statistics
// Returns: fileCount, totalSize, lastModified, hasManifest, error
func scanBackupDir(path string) (int, int64, time.Time, bool, error) {
	var fileCount int
	var totalSize int64
	var lastModified time.Time
	hasManifest := false

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if this is the manifest file
		if filepath.Base(filePath) == "manifest.json.enc" {
			hasManifest = true
		}

		fileCount++
		totalSize += info.Size()

		// Track most recent modification
		if info.ModTime().After(lastModified) {
			lastModified = info.ModTime()
		}

		return nil
	})

	if err != nil {
		return 0, 0, time.Time{}, false, err
	}

	return fileCount, totalSize, lastModified, hasManifest, nil
}

// DeleteIncomingBackup deletes a backup directory from disk
func DeleteIncomingBackup(backupPath string) error {
	// Security check: ensure path is within backups/incoming/
	if !strings.Contains(backupPath, "backups/incoming/") {
		return fmt.Errorf("invalid backup path: must be in backups/incoming/")
	}

	// Delete the entire directory
	if err := os.RemoveAll(backupPath); err != nil {
		return fmt.Errorf("failed to delete backup directory: %w", err)
	}

	return nil
}

// FormatBytes converts bytes to human-readable format
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

// FormatTimeAgo formats a time as "X hours/days ago"
func FormatTimeAgo(t time.Time, lang string) string {
	duration := time.Since(t)

	if duration < time.Minute {
		if lang == "fr" {
			return "Il y a moins d'une minute"
		}
		return "Less than a minute ago"
	}

	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if lang == "fr" {
			if minutes == 1 {
				return "Il y a 1 minute"
			}
			return fmt.Sprintf("Il y a %d minutes", minutes)
		}
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	}

	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if lang == "fr" {
			if hours == 1 {
				return "Il y a 1 heure"
			}
			return fmt.Sprintf("Il y a %d heures", hours)
		}
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}

	days := int(duration.Hours() / 24)
	if lang == "fr" {
		if days == 1 {
			return "Il y a 1 jour"
		}
		return fmt.Sprintf("Il y a %d jours", days)
	}
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}
