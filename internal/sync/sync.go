// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package sync implements P2P encrypted file synchronization between Anemone instances.
//
// This file contains core types, database operations, and high-level sync functions.
// For incremental sync logic, see sync_incremental.go.
// For archive-based sync and tar utilities, see sync_archive.go.
package sync

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/juste-un-gars/anemone/internal/crypto"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/peers"
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
	ShareID          int
	PeerID           int
	UserID           int
	SharePath        string
	PeerAddress      string
	PeerPort         int
	PeerPassword     string // Optional password for peer authentication
	SourceServer     string // Name of the source server (for manifest identification)
	PeerTimeoutHours int    // Sync timeout in hours (0 = disabled)
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

// HasRunningSyncForPeer checks if there's a running sync for a specific peer
func HasRunningSyncForPeer(db *sql.DB, peerID int) (bool, error) {
	query := `SELECT COUNT(*) FROM sync_log WHERE peer_id = ? AND status = 'running'`

	var count int
	err := db.QueryRow(query, peerID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check running sync: %w", err)
	}

	return count > 0, nil
}

// CleanupZombieSyncs marks stale "running" syncs as errors
// A sync is considered zombie if it's been running for more than 2 hours
func CleanupZombieSyncs(db *sql.DB) error {
	// Find zombie syncs (running for more than 2 hours)
	findQuery := `SELECT id, user_id, peer_id, started_at
	              FROM sync_log
	              WHERE status = 'running'
	              AND datetime(started_at) < datetime('now', '-2 hours')`

	rows, err := db.Query(findQuery)
	if err != nil {
		return fmt.Errorf("failed to query zombie syncs: %w", err)
	}
	defer rows.Close()

	var zombieCount int
	for rows.Next() {
		var id, userID, peerID int
		var startedAt string
		if err := rows.Scan(&id, &userID, &peerID, &startedAt); err != nil {
			logger.Info("⚠️  Failed to scan zombie sync: %v", err)
			continue
		}

		// Mark as error
		updateQuery := `UPDATE sync_log
		                SET status = 'error',
		                    completed_at = CURRENT_TIMESTAMP,
		                    error_message = 'Sync timeout - automatically cleaned up (zombie sync)'
		                WHERE id = ?`

		_, err := db.Exec(updateQuery, id)
		if err != nil {
			logger.Info("⚠️  Failed to cleanup zombie sync ID %d: %v", id, err)
			continue
		}

		logger.Info("🧹 Cleaned up zombie sync: ID=%d, User=%d, Peer=%d, Started=%s", id, userID, peerID, startedAt)
		zombieCount++
	}

	if zombieCount > 0 {
		logger.Info("✅ Cleaned up %d zombie sync(s)", zombieCount)
	}

	return nil
}

// GetServerName retrieves the NAS name from system config
func GetServerName(db *sql.DB) (string, error) {
	var serverName string
	err := db.QueryRow("SELECT value FROM system_config WHERE key = 'nas_name'").Scan(&serverName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "Unknown", nil // Default if not configured
		}
		return "", fmt.Errorf("failed to get server name: %w", err)
	}
	return serverName, nil
}

// GetUserEncryptionKey retrieves and decrypts the user's encryption key
func GetUserEncryptionKey(db *sql.DB, userID int) (string, error) {
	// Get master key from system config
	var masterKey string
	err := db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to get master key: %w", err)
	}

	// Get user's encrypted encryption key
	var encryptedKey []byte
	err = db.QueryRow("SELECT encryption_key_encrypted FROM users WHERE id = ?", userID).Scan(&encryptedKey)
	if err != nil {
		return "", fmt.Errorf("failed to get user encryption key: %w", err)
	}

	// Decrypt the user's encryption key
	decryptedKey, err := crypto.DecryptKey(string(encryptedKey), masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt user encryption key: %w", err)
	}

	return decryptedKey, nil
}

// SyncAllUsers synchronizes all users with sync_enabled shares to all enabled peers
// Returns: successCount, errorCount, lastError
func SyncAllUsers(db *sql.DB) (int, int, string) {
	// Get all shares with sync enabled
	sharesQuery := `SELECT id, user_id, name, path FROM shares WHERE sync_enabled = 1`
	shareRows, err := db.Query(sharesQuery)
	if err != nil {
		return 0, 1, fmt.Sprintf("Failed to query shares: %v", err)
	}
	defer shareRows.Close()

	type ShareInfo struct {
		ID     int
		UserID int
		Name   string
		Path   string
	}

	var sharesList []ShareInfo
	for shareRows.Next() {
		var s ShareInfo
		if err := shareRows.Scan(&s.ID, &s.UserID, &s.Name, &s.Path); err != nil {
			return 0, 1, fmt.Sprintf("Failed to scan share: %v", err)
		}
		sharesList = append(sharesList, s)
	}

	if len(sharesList) == 0 {
		return 0, 0, "No shares with sync enabled"
	}

	// Get all enabled peers
	peersQuery := `SELECT id, name, address, port, password, sync_timeout_hours FROM peers WHERE enabled = 1`
	peerRows, err := db.Query(peersQuery)
	if err != nil {
		return 0, 1, fmt.Sprintf("Failed to query peers: %v", err)
	}
	defer peerRows.Close()

	type PeerInfo struct {
		ID           int
		Name         string
		Address      string
		Port         int
		Password     *[]byte
		TimeoutHours int
	}

	var peersList []PeerInfo
	for peerRows.Next() {
		var p PeerInfo
		if err := peerRows.Scan(&p.ID, &p.Name, &p.Address, &p.Port, &p.Password, &p.TimeoutHours); err != nil {
			return 0, 1, fmt.Sprintf("Failed to scan peer: %v", err)
		}
		peersList = append(peersList, p)
	}

	if len(peersList) == 0 {
		return 0, 0, "No enabled peers"
	}

	// Get server name for manifest identification
	serverName, err := GetServerName(db)
	if err != nil {
		return 0, 1, fmt.Sprintf("Failed to get server name: %v", err)
	}

	// Get master key for password decryption
	var masterKey string
	err = db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		return 0, 1, fmt.Sprintf("Failed to get master key: %v", err)
	}

	// Sync each share to each peer
	successCount := 0
	errorCount := 0
	var lastError string

	for _, share := range sharesList {
		for _, peer := range peersList {
			// Decrypt peer password
			peerPassword := ""
			if peer.Password != nil && len(*peer.Password) > 0 {
				peerPassword, err = peers.DecryptPeerPassword(peer.Password, masterKey)
				if err != nil {
					errorCount++
					lastError = fmt.Sprintf("Failed to decrypt password for peer %s: %v", peer.Name, err)
					continue
				}
			}

			req := &SyncRequest{
				ShareID:          share.ID,
				PeerID:           peer.ID,
				UserID:           share.UserID,
				SharePath:        share.Path,
				PeerAddress:      peer.Address,
				PeerPort:         peer.Port,
				PeerPassword:     peerPassword,
				SourceServer:     serverName,
				PeerTimeoutHours: peer.TimeoutHours,
			}

			if err := SyncShareIncremental(db, req); err != nil {
				errorCount++
				lastError = fmt.Sprintf("Share %s to %s: %v", share.Name, peer.Name, err)
			} else {
				successCount++
			}
		}
	}

	return successCount, errorCount, lastError
}

// SyncPeer synchronizes all enabled shares to a specific peer
// Returns: successCount, errorCount, lastError
func SyncPeer(db *sql.DB, peerID int, peerName, peerAddress string, peerPort int, peerPassword *[]byte, peerTimeoutHours int) (int, int, string) {
	// Get all shares with sync enabled
	sharesQuery := `SELECT id, user_id, name, path FROM shares WHERE sync_enabled = 1`
	shareRows, err := db.Query(sharesQuery)
	if err != nil {
		return 0, 1, fmt.Sprintf("Failed to query shares: %v", err)
	}
	defer shareRows.Close()

	type ShareInfo struct {
		ID     int
		UserID int
		Name   string
		Path   string
	}

	var sharesList []ShareInfo
	for shareRows.Next() {
		var s ShareInfo
		if err := shareRows.Scan(&s.ID, &s.UserID, &s.Name, &s.Path); err != nil {
			return 0, 1, fmt.Sprintf("Failed to scan share: %v", err)
		}
		sharesList = append(sharesList, s)
	}

	if len(sharesList) == 0 {
		return 0, 0, "No shares with sync enabled"
	}

	// Sync each share to this peer
	successCount := 0
	errorCount := 0
	var lastError string

	// Get server name for manifest identification
	serverName, err := GetServerName(db)
	if err != nil {
		return 0, 1, fmt.Sprintf("Failed to get server name: %v", err)
	}

	// Get master key for password decryption
	var masterKey string
	err = db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		return 0, 1, fmt.Sprintf("Failed to get master key: %v", err)
	}

	// Decrypt peer password
	password := ""
	if peerPassword != nil && len(*peerPassword) > 0 {
		password, err = peers.DecryptPeerPassword(peerPassword, masterKey)
		if err != nil {
			return 0, 1, fmt.Sprintf("Failed to decrypt password for peer %s: %v", peerName, err)
		}
	}

	for _, share := range sharesList {
		req := &SyncRequest{
			ShareID:          share.ID,
			PeerID:           peerID,
			UserID:           share.UserID,
			SharePath:        share.Path,
			PeerAddress:      peerAddress,
			PeerPort:         peerPort,
			PeerPassword:     password,
			SourceServer:     serverName,
			PeerTimeoutHours: peerTimeoutHours,
		}

		if err := SyncShareIncremental(db, req); err != nil {
			errorCount++
			lastError = fmt.Sprintf("Share %s to %s: %v", share.Name, peerName, err)
		} else {
			successCount++
		}
	}

	return successCount, errorCount, lastError
}
