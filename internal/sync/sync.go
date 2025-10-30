// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package sync

import (
	"database/sql"
	"fmt"
	"os/exec"
	"time"
)

// SyncLog represents a synchronization log entry
type SyncLog struct {
	ID           int
	UserID       int
	PeerID       int
	StartedAt    time.Time
	CompletedAt  *time.Time
	Status       string // "running", "success", "error"
	FilesSynced  int
	BytesSynced  int64
	ErrorMessage string
}

// SyncRequest represents a synchronization request
type SyncRequest struct {
	ShareID  int
	PeerID   int
	UserID   int
	SharePath string
	PeerAddress string
	PeerPort    int
}

// CreateSyncLog creates a new sync log entry and returns its ID
func CreateSyncLog(db *sql.DB, userID, peerID int) (int, error) {
	query := `INSERT INTO sync_log (user_id, peer_id, started_at, status)
	          VALUES (?, ?, CURRENT_TIMESTAMP, 'running')`

	result, err := db.Exec(query, userID, peerID)
	if err != nil {
		return 0, fmt.Errorf("failed to create sync log: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get sync log ID: %w", err)
	}

	return int(id), nil
}

// UpdateSyncLog updates a sync log entry with completion details
func UpdateSyncLog(db *sql.DB, logID int, status string, filesSynced int, bytesSynced int64, errorMsg string) error {
	query := `UPDATE sync_log
	          SET completed_at = CURRENT_TIMESTAMP, status = ?, files_synced = ?, bytes_synced = ?, error_message = ?
	          WHERE id = ?`

	_, err := db.Exec(query, status, filesSynced, bytesSynced, errorMsg, logID)
	if err != nil {
		return fmt.Errorf("failed to update sync log: %w", err)
	}

	return nil
}

// GetLastSyncByUser retrieves the last sync log for a user
func GetLastSyncByUser(db *sql.DB, userID int) (*SyncLog, error) {
	query := `SELECT id, user_id, peer_id, started_at, completed_at, status, files_synced, bytes_synced, error_message
	          FROM sync_log
	          WHERE user_id = ?
	          ORDER BY started_at DESC
	          LIMIT 1`

	log := &SyncLog{}
	err := db.QueryRow(query, userID).Scan(
		&log.ID, &log.UserID, &log.PeerID, &log.StartedAt, &log.CompletedAt,
		&log.Status, &log.FilesSynced, &log.BytesSynced, &log.ErrorMessage,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No sync yet
		}
		return nil, fmt.Errorf("failed to get last sync: %w", err)
	}

	return log, nil
}

// GetSyncLogs retrieves sync logs for a user with optional limit
func GetSyncLogs(db *sql.DB, userID int, limit int) ([]*SyncLog, error) {
	query := `SELECT id, user_id, peer_id, started_at, completed_at, status, files_synced, bytes_synced, error_message
	          FROM sync_log
	          WHERE user_id = ?
	          ORDER BY started_at DESC`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sync logs: %w", err)
	}
	defer rows.Close()

	var logs []*SyncLog
	for rows.Next() {
		log := &SyncLog{}
		err := rows.Scan(
			&log.ID, &log.UserID, &log.PeerID, &log.StartedAt, &log.CompletedAt,
			&log.Status, &log.FilesSynced, &log.BytesSynced, &log.ErrorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sync log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// SyncShare synchronizes a share to a peer using rsync over SSH
// This is a simple implementation for testing. Will be replaced with rclone + encryption
func SyncShare(db *sql.DB, req *SyncRequest) error {
	// Create sync log entry
	logID, err := CreateSyncLog(db, req.UserID, req.PeerID)
	if err != nil {
		return fmt.Errorf("failed to create sync log: %w", err)
	}

	// For now, we'll use a simple rsync approach
	// In production, this will use rclone with encryption via WebDAV/SFTP

	// Build rsync command (placeholder - needs SSH setup)
	// rsync -avz --delete /local/path/ user@remote:/remote/path/
	remoteTarget := fmt.Sprintf("root@%s:%s", req.PeerAddress, req.SharePath)
	cmd := exec.Command("rsync", "-avz", "--delete", req.SharePath+"/", remoteTarget)

	// Run rsync
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Update log with error
		UpdateSyncLog(db, logID, "error", 0, 0, fmt.Sprintf("rsync failed: %s - %s", err.Error(), string(output)))
		return fmt.Errorf("sync failed: %w - %s", err, string(output))
	}

	// Parse rsync output to count files (simplified for now)
	// In production, parse the actual output or use rsync --stats
	filesSynced := 0
	bytesSynced := int64(0)

	// Update log with success
	err = UpdateSyncLog(db, logID, "success", filesSynced, bytesSynced, "")
	if err != nil {
		return fmt.Errorf("failed to update sync log: %w", err)
	}

	// Update peer's last_sync timestamp
	updatePeerQuery := `UPDATE peers SET last_sync = CURRENT_TIMESTAMP WHERE id = ?`
	_, err = db.Exec(updatePeerQuery, req.PeerID)
	if err != nil {
		return fmt.Errorf("failed to update peer last_sync: %w", err)
	}

	return nil
}

// TestRsyncAvailable checks if rsync is available on the system
func TestRsyncAvailable() error {
	cmd := exec.Command("rsync", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rsync not found: %w", err)
	}
	return nil
}
