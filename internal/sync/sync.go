// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package sync

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/juste-un-gars/anemone/internal/crypto"
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
	ShareID      int
	PeerID       int
	UserID       int
	SharePath    string
	PeerAddress  string
	PeerPort     int
	PeerPassword string // Optional password for peer authentication
	SourceServer string // Name of the source server (for manifest identification)
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

// SyncShare synchronizes a share to a peer using HTTPS with encryption
// Creates tar.gz archive, encrypts it with user's key, and sends to peer
func SyncShare(db *sql.DB, req *SyncRequest) error {
	// Create sync log entry
	logID, err := CreateSyncLog(db, req.UserID, req.PeerID)
	if err != nil {
		return fmt.Errorf("failed to create sync log: %w", err)
	}

	// Get user's encryption key
	encryptionKey, err := GetUserEncryptionKey(db, req.UserID)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get encryption key: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Create tar.gz archive of the share directory
	var tarBuf bytes.Buffer
	fileCount, totalSize, err := createTarGz(&tarBuf, req.SharePath)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create archive: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Encrypt the archive
	var encryptedBuf bytes.Buffer
	if err := crypto.EncryptStream(&tarBuf, &encryptedBuf, encryptionKey); err != nil {
		errMsg := fmt.Sprintf("Failed to encrypt archive: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Send encrypted archive to peer via HTTP POST
	peerURL := fmt.Sprintf("https://%s:%d/api/sync/receive", req.PeerAddress, req.PeerPort)

	// Create multipart form with share info
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add metadata fields
	writer.WriteField("share_id", fmt.Sprintf("%d", req.ShareID))
	writer.WriteField("user_id", fmt.Sprintf("%d", req.UserID))
	writer.WriteField("encrypted", "true") // Flag to indicate encryption
	// Extract share name from path (last directory)
	shareName := filepath.Base(filepath.Dir(req.SharePath))
	writer.WriteField("share_name", shareName)

	// Add encrypted archive file
	part, err := writer.CreateFormFile("archive", "share.tar.gz.enc")
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create form file: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}
	io.Copy(part, &encryptedBuf)
	writer.Close()

	// Create HTTP client that accepts self-signed certs
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   5 * time.Minute, // 5 min timeout for large transfers
	}

	// Send POST request
	resp, err := client.Post(peerURL, writer.FormDataContentType(), &requestBody)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to send to peer: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("Peer returned error %d: %s", resp.StatusCode, string(body))
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Update log with success
	err = UpdateSyncLog(db, logID, "success", fileCount, totalSize, "")
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

// SyncShareIncremental performs incremental file-by-file sync with encryption
// Uses manifest-based approach to only sync changed files
func SyncShareIncremental(db *sql.DB, req *SyncRequest) error {
	// Create sync log entry
	logID, err := CreateSyncLog(db, req.UserID, req.PeerID)
	if err != nil {
		return fmt.Errorf("failed to create sync log: %w", err)
	}

	// Get user's encryption key
	encryptionKey, err := GetUserEncryptionKey(db, req.UserID)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get encryption key: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Extract share name from path
	shareName := filepath.Base(filepath.Dir(req.SharePath))

	// Build local manifest
	localManifest, err := BuildManifest(req.SharePath, req.UserID, shareName, req.SourceServer)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to build local manifest: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Fetch remote manifest from peer
	peerURL := fmt.Sprintf("https://%s:%d/api/sync/manifest?user_id=%d&share_name=%s",
		req.PeerAddress, req.PeerPort, req.UserID, shareName)

	// Create HTTP client that accepts self-signed certs
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   5 * time.Minute,
	}

	// Try to fetch remote manifest
	var remoteManifest *SyncManifest
	manifestReq, err := http.NewRequest(http.MethodGet, peerURL, nil)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create manifest request: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Add authentication header if password is provided
	if req.PeerPassword != "" {
		manifestReq.Header.Set("X-Sync-Password", req.PeerPassword)
	}

	resp, err := client.Do(manifestReq)
	if err == nil && resp.StatusCode == http.StatusOK {
		// Manifest exists - decrypt it
		defer resp.Body.Close()
		encryptedData, err := io.ReadAll(resp.Body)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to read remote manifest: %v", err)
			UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
			return fmt.Errorf(errMsg)
		}

		// Decrypt manifest
		var decryptedBuf bytes.Buffer
		if err := crypto.DecryptStream(bytes.NewReader(encryptedData), &decryptedBuf, encryptionKey); err != nil {
			errMsg := fmt.Sprintf("Failed to decrypt manifest: %v", err)
			UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
			return fmt.Errorf(errMsg)
		}

		// Unmarshal manifest
		remoteManifest, err = UnmarshalManifest(decryptedBuf.Bytes())
		if err != nil {
			errMsg := fmt.Sprintf("Failed to parse remote manifest: %v", err)
			UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
			return fmt.Errorf(errMsg)
		}
	} else if resp != nil && resp.StatusCode == http.StatusNotFound {
		// No remote manifest yet (first sync) - that's OK
		remoteManifest = nil
	} else {
		// Other error
		errMsg := fmt.Sprintf("Failed to fetch remote manifest: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Compare manifests to get delta
	delta, err := CompareManifests(localManifest, remoteManifest)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to compare manifests: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Debug log for manifests
	log.Printf("📊 Sync user %d to peer %d:", req.UserID, req.PeerID)
	log.Printf("   Local manifest has %d files", len(localManifest.Files))
	if remoteManifest != nil {
		log.Printf("   Remote manifest has %d files", len(remoteManifest.Files))
	} else {
		log.Printf("   Remote manifest is nil (first sync)")
	}

	// Debug log for delta
	log.Printf("   Delta: %d to add, %d to update, %d to delete",
		len(delta.ToAdd), len(delta.ToUpdate), len(delta.ToDelete))
	if len(delta.ToDelete) > 0 {
		log.Printf("🗑️  Files to delete: %v", delta.ToDelete)
	}

	var totalBytes int64 = 0
	totalFiles := len(delta.ToAdd) + len(delta.ToUpdate)

	// Upload new and modified files
	filesToUpload := append(delta.ToAdd, delta.ToUpdate...)
	for _, relativePath := range filesToUpload {
		fileMeta := localManifest.Files[relativePath]
		sourcePath := filepath.Join(req.SharePath, relativePath)

		// Read file
		fileData, err := os.ReadFile(sourcePath)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to read file %s: %v", relativePath, err)
			UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
			return fmt.Errorf(errMsg)
		}

		// Encrypt file
		var encryptedBuf bytes.Buffer
		if err := crypto.EncryptStream(bytes.NewReader(fileData), &encryptedBuf, encryptionKey); err != nil {
			errMsg := fmt.Sprintf("Failed to encrypt file %s: %v", relativePath, err)
			UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
			return fmt.Errorf(errMsg)
		}

		// Upload encrypted file
		uploadURL := fmt.Sprintf("https://%s:%d/api/sync/file?source_server=%s", req.PeerAddress, req.PeerPort, req.SourceServer)

		var requestBody bytes.Buffer
		writer := multipart.NewWriter(&requestBody)

		// Add metadata
		writer.WriteField("user_id", fmt.Sprintf("%d", req.UserID))
		writer.WriteField("share_name", shareName)
		writer.WriteField("relative_path", fileMeta.EncryptedPath) // Use .enc path

		// Add file
		part, err := writer.CreateFormFile("file", filepath.Base(fileMeta.EncryptedPath))
		if err != nil {
			errMsg := fmt.Sprintf("Failed to create form file for %s: %v", relativePath, err)
			UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
			return fmt.Errorf(errMsg)
		}
		io.Copy(part, &encryptedBuf)
		writer.Close()

		// Send POST request
		uploadReq, err := http.NewRequest(http.MethodPost, uploadURL, &requestBody)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to create upload request for %s: %v", relativePath, err)
			UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
			return fmt.Errorf(errMsg)
		}
		uploadReq.Header.Set("Content-Type", writer.FormDataContentType())

		// Add authentication header if password is provided
		if req.PeerPassword != "" {
			uploadReq.Header.Set("X-Sync-Password", req.PeerPassword)
		}

		resp, err := client.Do(uploadReq)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to upload file %s: %v", relativePath, err)
			UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
			return fmt.Errorf(errMsg)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errMsg := fmt.Sprintf("Failed to upload file %s: status %d", relativePath, resp.StatusCode)
			UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
			return fmt.Errorf(errMsg)
		}

		totalBytes += fileMeta.Size
	}

	// Delete obsolete files on peer
	for _, relativePath := range delta.ToDelete {
		remoteMeta := remoteManifest.Files[relativePath]
		deleteURL := fmt.Sprintf("https://%s:%d/api/sync/file?source_server=%s&user_id=%d&share_name=%s&path=%s",
			req.PeerAddress, req.PeerPort, req.SourceServer, req.UserID, shareName, remoteMeta.EncryptedPath)

		deleteReq, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to create delete request for %s: %v", relativePath, err)
			UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
			return fmt.Errorf(errMsg)
		}

		// Add authentication header if password is provided
		if req.PeerPassword != "" {
			deleteReq.Header.Set("X-Sync-Password", req.PeerPassword)
		}

		resp, err := client.Do(deleteReq)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to delete file %s: %v", relativePath, err)
			UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
			return fmt.Errorf(errMsg)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errMsg := fmt.Sprintf("Failed to delete file %s: status %d", relativePath, resp.StatusCode)
			UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
			return fmt.Errorf(errMsg)
		}

		log.Printf("✅ Deleted obsolete file on peer: %s (encrypted: %s)", relativePath, remoteMeta.EncryptedPath)
	}

	// Marshal and encrypt updated manifest
	manifestJSON, err := MarshalManifest(localManifest)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to marshal manifest: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}

	var encryptedManifest bytes.Buffer
	if err := crypto.EncryptStream(bytes.NewReader(manifestJSON), &encryptedManifest, encryptionKey); err != nil {
		errMsg := fmt.Sprintf("Failed to encrypt manifest: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Upload encrypted manifest
	manifestURL := fmt.Sprintf("https://%s:%d/api/sync/manifest?source_server=%s&user_id=%d&share_name=%s",
		req.PeerAddress, req.PeerPort, req.SourceServer, req.UserID, shareName)

	manifestPutReq, err := http.NewRequest(http.MethodPut, manifestURL, &encryptedManifest)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create manifest upload request: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}
	manifestPutReq.Header.Set("Content-Type", "application/octet-stream")

	// Add authentication header if password is provided
	if req.PeerPassword != "" {
		manifestPutReq.Header.Set("X-Sync-Password", req.PeerPassword)
	}

	resp, err = client.Do(manifestPutReq)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to upload manifest: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Failed to upload manifest: status %d", resp.StatusCode)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Cleanup orphaned files on peer (files that exist physically but not in manifest)
	if err := cleanupOrphanedFiles(client, req, localManifest, shareName); err != nil {
		// Log error but don't fail the sync - cleanup is best-effort
		log.Printf("⚠️  Warning: Failed to cleanup orphaned files: %v", err)
	}

	// Upload source server info (unencrypted metadata for display purposes)
	sourceInfo := map[string]string{
		"source_server": req.SourceServer,
		"synced_at":     time.Now().Format(time.RFC3339),
	}
	sourceInfoJSON, _ := json.Marshal(sourceInfo)

	sourceInfoURL := fmt.Sprintf("https://%s:%d/api/sync/source-info?source_server=%s&user_id=%d&share_name=%s",
		req.PeerAddress, req.PeerPort, req.SourceServer, req.UserID, shareName)

	sourceInfoReq, err := http.NewRequest(http.MethodPut, sourceInfoURL, bytes.NewReader(sourceInfoJSON))
	if err == nil {
		sourceInfoReq.Header.Set("Content-Type", "application/json")
		if req.PeerPassword != "" {
			sourceInfoReq.Header.Set("X-Sync-Password", req.PeerPassword)
		}
		// Send source info (ignore errors - it's just metadata)
		client.Do(sourceInfoReq)
	}

	// Update log with success
	err = UpdateSyncLog(db, logID, "success", totalFiles, totalBytes, "")
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
	peersQuery := `SELECT id, name, address, port, password FROM peers WHERE enabled = 1`
	peerRows, err := db.Query(peersQuery)
	if err != nil {
		return 0, 1, fmt.Sprintf("Failed to query peers: %v", err)
	}
	defer peerRows.Close()

	type PeerInfo struct {
		ID       int
		Name     string
		Address  string
		Port     int
		Password *string
	}

	var peersList []PeerInfo
	for peerRows.Next() {
		var p PeerInfo
		if err := peerRows.Scan(&p.ID, &p.Name, &p.Address, &p.Port, &p.Password); err != nil {
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

	// Sync each share to each peer
	successCount := 0
	errorCount := 0
	var lastError string

	for _, share := range sharesList {
		for _, peer := range peersList {
			// Get peer password (empty string if NULL)
			peerPassword := ""
			if peer.Password != nil {
				peerPassword = *peer.Password
			}

			req := &SyncRequest{
				ShareID:      share.ID,
				PeerID:       peer.ID,
				UserID:       share.UserID,
				SharePath:    share.Path,
				PeerAddress:  peer.Address,
				PeerPort:     peer.Port,
				PeerPassword: peerPassword,
				SourceServer: serverName,
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

// createTarGz creates a tar.gz archive of a directory
// Returns: fileCount, totalSize (bytes), error
func createTarGz(buf *bytes.Buffer, sourceDir string) (int, int64, error) {
	gzipWriter := gzip.NewWriter(buf)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	fileCount := 0
	var totalSize int64

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// Update header name to be relative to source dir
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// If not a regular file, skip content
		if !info.Mode().IsRegular() {
			return nil
		}

		// Write file content
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		written, err := io.Copy(tarWriter, file)
		if err != nil {
			return err
		}

		fileCount++
		totalSize += written

		return nil
	})

	return fileCount, totalSize, err
}

// ExtractTarGz extracts a tar.gz archive to a destination directory
func ExtractTarGz(reader io.Reader, destDir string) error {
	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create gzip reader
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	// Extract each file
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct full path
		targetPath := filepath.Join(destDir, header.Name)

		// Check for path traversal attacks
		if !filepath.HasPrefix(targetPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", header.Name)
		}

		// Handle different file types
		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}

		case tar.TypeReg:
			// Create parent directories if needed
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Create file
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			// Copy file content
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
			outFile.Close()

			// Set file permissions
			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to set permissions on %s: %w", targetPath, err)
			}

		default:
			// Skip other types (symlinks, etc.) for now
			continue
		}
	}

	return nil
}

// SyncPeer synchronizes all enabled shares to a specific peer
// Returns: successCount, errorCount, lastError
func SyncPeer(db *sql.DB, peerID int, peerName, peerAddress string, peerPort int, peerPassword *string) (int, int, string) {
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

	// Get peer password (empty string if NULL)
	password := ""
	if peerPassword != nil {
		password = *peerPassword
	}

	for _, share := range sharesList {
		req := &SyncRequest{
			ShareID:      share.ID,
			PeerID:       peerID,
			UserID:       share.UserID,
			SharePath:    share.Path,
			PeerAddress:  peerAddress,
			PeerPort:     peerPort,
			PeerPassword: password,
			SourceServer: serverName,
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

// cleanupOrphanedFiles removes files on peer that don't exist in the local manifest
// This handles orphaned files that were left behind (e.g., from trash deletion)
func cleanupOrphanedFiles(client *http.Client, req *SyncRequest, localManifest *SyncManifest, shareName string) error {
	// Fetch list of physical files from peer
	listURL := fmt.Sprintf("https://%s:%d/api/sync/list-physical-files?source_server=%s&user_id=%d&share_name=%s",
		req.PeerAddress, req.PeerPort, req.SourceServer, req.UserID, shareName)

	listReq, err := http.NewRequest(http.MethodGet, listURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create list request: %w", err)
	}

	// Add authentication header if password is provided
	if req.PeerPassword != "" {
		listReq.Header.Set("X-Sync-Password", req.PeerPassword)
	}

	resp, err := client.Do(listReq)
	if err != nil {
		return fmt.Errorf("failed to fetch physical files list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch physical files list: status %d", resp.StatusCode)
	}

	// Parse response
	var filesList struct {
		Files []string `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&filesList); err != nil {
		return fmt.Errorf("failed to parse physical files list: %w", err)
	}

	// Build set of expected files from local manifest
	expectedFiles := make(map[string]bool)
	for _, meta := range localManifest.Files {
		expectedFiles[meta.EncryptedPath] = true
	}

	// Find orphaned files (physical files not in manifest)
	var orphanedFiles []string
	for _, physicalFile := range filesList.Files {
		if !expectedFiles[physicalFile] {
			orphanedFiles = append(orphanedFiles, physicalFile)
		}
	}

	// Delete orphaned files
	if len(orphanedFiles) > 0 {
		log.Printf("🧹 Found %d orphaned file(s) to clean up on peer", len(orphanedFiles))

		for _, orphanedFile := range orphanedFiles {
			deleteURL := fmt.Sprintf("https://%s:%d/api/sync/file?source_server=%s&user_id=%d&share_name=%s&path=%s",
				req.PeerAddress, req.PeerPort, req.SourceServer, req.UserID, shareName, orphanedFile)

			deleteReq, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
			if err != nil {
				log.Printf("⚠️  Failed to create delete request for orphaned file %s: %v", orphanedFile, err)
				continue
			}

			if req.PeerPassword != "" {
				deleteReq.Header.Set("X-Sync-Password", req.PeerPassword)
			}

			resp, err := client.Do(deleteReq)
			if err != nil {
				log.Printf("⚠️  Failed to delete orphaned file %s: %v", orphanedFile, err)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Printf("⚠️  Failed to delete orphaned file %s: status %d", orphanedFile, resp.StatusCode)
				continue
			}

			log.Printf("✅ Deleted orphaned file: %s", orphanedFile)
		}
	}

	return nil
}
