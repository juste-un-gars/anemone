// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// This file contains incremental sync logic for P2P file synchronization.

package sync

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/juste-un-gars/anemone/internal/crypto"
	"github.com/juste-un-gars/anemone/internal/logger"
)

// uploadManifestToRemote uploads an encrypted manifest to the remote peer
// This is a helper function to allow progressive manifest saves during sync
func uploadManifestToRemote(ctx context.Context, client *http.Client, req *SyncRequest, manifest *SyncManifest, shareName string, encryptionKey string) error {
	// Marshal and encrypt manifest
	manifestJSON, err := MarshalManifest(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	var encryptedManifest bytes.Buffer
	if err := crypto.EncryptStream(bytes.NewReader(manifestJSON), &encryptedManifest, encryptionKey); err != nil {
		return fmt.Errorf("failed to encrypt manifest: %w", err)
	}

	// Upload encrypted manifest
	manifestURL := fmt.Sprintf("https://%s:%d/api/sync/manifest?source_server=%s&user_id=%d&share_name=%s",
		req.PeerAddress, req.PeerPort, url.QueryEscape(req.SourceServer), req.UserID, url.QueryEscape(shareName))

	manifestPutReq, err := http.NewRequestWithContext(ctx, http.MethodPut, manifestURL, &encryptedManifest)
	if err != nil {
		return fmt.Errorf("failed to create manifest upload request: %w", err)
	}
	manifestPutReq.Header.Set("Content-Type", "application/octet-stream")

	// Add authentication headers if password is provided
	if req.PeerPassword != "" {
		manifestPutReq.Header.Set("X-Sync-Password", req.PeerPassword)
		manifestPutReq.Header.Set("X-Source-Server", req.SourceServer)
	}

	resp, err := client.Do(manifestPutReq)
	if err != nil {
		return fmt.Errorf("failed to upload manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("manifest upload returned status %d", resp.StatusCode)
	}

	return nil
}

// SyncShareIncremental performs incremental file-by-file sync with encryption
// Uses manifest-based approach to only sync changed files
func SyncShareIncremental(db *sql.DB, req *SyncRequest) error {
	// Check if sync is already running for this peer
	hasRunning, err := HasRunningSyncForPeer(db, req.PeerID)
	if err != nil {
		return fmt.Errorf("failed to check running sync: %w", err)
	}
	if hasRunning {
		return fmt.Errorf("sync already in progress for peer ID %d", req.PeerID)
	}

	// Create context with configurable timeout (or no timeout if disabled)
	var ctx context.Context
	var cancel context.CancelFunc
	if req.PeerTimeoutHours > 0 {
		// Use configured timeout
		timeoutDuration := time.Duration(req.PeerTimeoutHours) * time.Hour
		ctx, cancel = context.WithTimeout(context.Background(), timeoutDuration)
		logger.Info("Sync timeout configured", "hours", req.PeerTimeoutHours)
	} else {
		// No timeout (0 = disabled)
		ctx, cancel = context.WithCancel(context.Background())
		logger.Info("⏱️  Sync timeout: disabled")
	}
	defer cancel()

	// Create sync log entry
	logID, err := CreateSyncLog(db, req.UserID, req.PeerID)
	if err != nil {
		return fmt.Errorf("failed to create sync log: %w", err)
	}

	// Track actual upload progress (for stats even if error occurs)
	var uploadedCount int = 0
	var totalBytes int64 = 0

	// Check for context cancellation/timeout
	select {
	case <-ctx.Done():
		errMsg := fmt.Sprintf("Sync timeout: %v", ctx.Err())
		UpdateSyncLog(db, logID, "error", uploadedCount, totalBytes, errMsg)
		return fmt.Errorf(errMsg)
	default:
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
	peerURL := fmt.Sprintf("https://%s:%d/api/sync/manifest?source_server=%s&user_id=%d&share_name=%s",
		req.PeerAddress, req.PeerPort, url.QueryEscape(req.SourceServer), req.UserID, url.QueryEscape(shareName))

	// Create HTTP client with optimized connection pooling for many small files
	// Keep-alive is enabled by default, but we optimize the pool settings
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			// Enable TLS session resumption for faster subsequent handshakes
			ClientSessionCache: tls.NewLRUClientSessionCache(32),
		},
		// Connection pool optimization for sequential uploads
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     120 * time.Second,
		// Disable compression (files are already encrypted, compression won't help)
		DisableCompression: true,
		// Force HTTP/1.1 keep-alive
		ForceAttemptHTTP2:     false,
		MaxConnsPerHost:       10,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	client := &http.Client{
		Transport: tr,
		// No global timeout - each request manages its own via context
	}

	// Try to fetch remote manifest
	var remoteManifest *SyncManifest
	manifestReq, err := http.NewRequestWithContext(ctx, http.MethodGet, peerURL, nil)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create manifest request: %v", err)
		UpdateSyncLog(db, logID, "error", 0, 0, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Add authentication headers if password is provided
	if req.PeerPassword != "" {
		manifestReq.Header.Set("X-Sync-Password", req.PeerPassword)
		manifestReq.Header.Set("X-Source-Server", req.SourceServer)
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
	logger.Info("Sync started", "user_id", req.UserID, "peer_id", req.PeerID)
	logger.Info("Local manifest loaded", "file_count", len(localManifest.Files))
	if remoteManifest != nil {
		logger.Info("Remote manifest loaded", "file_count", len(remoteManifest.Files))
	} else {
		logger.Info("   Remote manifest is nil (first sync)")
	}

	// Debug log for delta
	logger.Info("Sync delta computed", "to_add", len(delta.ToAdd), "to_update", len(delta.ToUpdate), "to_delete", len(delta.ToDelete))
	if len(delta.ToDelete) > 0 {
		logger.Info("Files to delete", "to_delete", delta.ToDelete)
	}

	// Create progress manifest - starts as copy of remote (or new if remote is nil)
	// This manifest will be updated incrementally and saved in batches to enable resumable sync
	progressManifest := &SyncManifest{
		Version:      1,
		LastSync:     time.Now(),
		UserID:       req.UserID,
		ShareName:    shareName,
		SourceServer: req.SourceServer,
		Files:        make(map[string]FileMetadata),
	}
	// Copy existing remote files if available
	if remoteManifest != nil && remoteManifest.Files != nil {
		for k, v := range remoteManifest.Files {
			progressManifest.Files[k] = v
		}
	}

	// Setup defer to save progress manifest on error/timeout (allows resumable sync)
	var syncErr error
	defer func() {
		if syncErr != nil && uploadedCount > 0 {
			// Save progress manifest to enable resumable sync
			logger.Info("Sync error occurred after uploading files - saving progress manifest for resumable sync", "uploaded_count", uploadedCount)
			if manifestErr := uploadManifestToRemote(ctx, client, req, progressManifest, shareName, encryptionKey); manifestErr != nil {
				logger.Info("Failed to save progress manifest", "manifest_err", manifestErr)
			} else {
				logger.Info("✅ Progress manifest saved successfully - next sync will resume from here")
			}
		}
	}()

	// Calculate total files to sync
	totalFiles := len(delta.ToAdd) + len(delta.ToUpdate)

	// Upload new and modified files
	filesToUpload := append(delta.ToAdd, delta.ToUpdate...)
	lastLoggedCount := 0

	if totalFiles > 0 {
		logger.Info("Starting upload", "total_files", totalFiles)
	}

	for _, relativePath := range filesToUpload {
		fileMeta := localManifest.Files[relativePath]
		sourcePath := filepath.Join(req.SharePath, relativePath)

		// Open file for streaming (don't load entire file in RAM)
		file, err := os.Open(sourcePath)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to open file %s: %v", relativePath, err)
			UpdateSyncLog(db, logID, "error", uploadedCount, totalBytes, errMsg)
			syncErr = fmt.Errorf(errMsg)
			return syncErr
		}

		// Stream encrypt and upload file (memory-efficient)
		err = streamEncryptAndUpload(ctx, client, file, req, shareName, fileMeta.EncryptedPath, encryptionKey, req.UserID)
		file.Close()

		if err != nil {
			errMsg := fmt.Sprintf("Failed to upload file %s: %v", relativePath, err)
			UpdateSyncLog(db, logID, "error", uploadedCount, totalBytes, errMsg)
			syncErr = fmt.Errorf(errMsg)
			return syncErr
		}

		totalBytes += fileMeta.Size
		uploadedCount++

		// Update progress manifest with successfully uploaded file
		progressManifest.Files[relativePath] = fileMeta
		progressManifest.LastSync = time.Now()

		// Save progress manifest every 500 files (checkpoint for resumable sync)
		if uploadedCount%500 == 0 {
			logger.Info("Checkpoint: saving progress manifest after files...", "uploaded_count", uploadedCount)
			if manifestErr := uploadManifestToRemote(ctx, client, req, progressManifest, shareName, encryptionKey); manifestErr != nil {
				logger.Info("Warning: failed to save progress manifest checkpoint", "manifest_err", manifestErr)
			} else {
				logger.Info("✅ Progress manifest checkpoint saved successfully")
			}
		}

		// Log progress every 100 files
		if uploadedCount-lastLoggedCount >= 100 {
			percentage := (uploadedCount * 100) / totalFiles
			logger.Info("Upload progress: / files (%%)", "uploaded_count", uploadedCount, "total_files", totalFiles, "percentage", percentage)
			lastLoggedCount = uploadedCount
		}
	}

	// Delete obsolete files on peer
	for _, relativePath := range delta.ToDelete {
		remoteMeta := remoteManifest.Files[relativePath]
		deleteURL := fmt.Sprintf("https://%s:%d/api/sync/file?source_server=%s&user_id=%d&share_name=%s&path=%s",
			req.PeerAddress, req.PeerPort, url.QueryEscape(req.SourceServer), req.UserID, url.QueryEscape(shareName), url.QueryEscape(remoteMeta.EncryptedPath))

		deleteReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to create delete request for %s: %v", relativePath, err)
			UpdateSyncLog(db, logID, "error", uploadedCount, totalBytes, errMsg)
			syncErr = fmt.Errorf(errMsg)
			return syncErr
		}

		// Add authentication headers if password is provided
		if req.PeerPassword != "" {
			deleteReq.Header.Set("X-Sync-Password", req.PeerPassword)
			deleteReq.Header.Set("X-Source-Server", req.SourceServer)
		}

		resp, err := client.Do(deleteReq)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to delete file %s: %v", relativePath, err)
			UpdateSyncLog(db, logID, "error", uploadedCount, totalBytes, errMsg)
			syncErr = fmt.Errorf(errMsg)
			return syncErr
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errMsg := fmt.Sprintf("Failed to delete file %s: status %d", relativePath, resp.StatusCode)
			UpdateSyncLog(db, logID, "error", uploadedCount, totalBytes, errMsg)
			syncErr = fmt.Errorf(errMsg)
			return syncErr
		}

		// Remove deleted file from progress manifest
		delete(progressManifest.Files, relativePath)

		logger.Info("Deleted obsolete file on peer", "relative_path", relativePath, "encrypted_path", remoteMeta.EncryptedPath)
	}

	// Save final progress manifest (reflects actual state on peer)
	logger.Info("💾 Saving final manifest...")
	if err := uploadManifestToRemote(ctx, client, req, progressManifest, shareName, encryptionKey); err != nil {
		errMsg := fmt.Sprintf("Failed to upload final manifest: %v", err)
		UpdateSyncLog(db, logID, "error", uploadedCount, totalBytes, errMsg)
		syncErr = fmt.Errorf(errMsg)
		return syncErr
	}
	logger.Info("✅ Final manifest saved successfully")

	// Cleanup orphaned files on peer (files that exist physically but not in manifest)
	if err := cleanupOrphanedFiles(ctx, client, req, localManifest, shareName); err != nil {
		// Log error but don't fail the sync - cleanup is best-effort
		logger.Info("Warning: Failed to cleanup orphaned files", "error", err)
	}

	// Upload source server info (unencrypted metadata for display purposes)
	sourceInfo := map[string]string{
		"source_server": req.SourceServer,
		"synced_at":     time.Now().Format(time.RFC3339),
	}
	sourceInfoJSON, _ := json.Marshal(sourceInfo)

	sourceInfoURL := fmt.Sprintf("https://%s:%d/api/sync/source-info?source_server=%s&user_id=%d&share_name=%s",
		req.PeerAddress, req.PeerPort, url.QueryEscape(req.SourceServer), req.UserID, url.QueryEscape(shareName))

	sourceInfoReq, err := http.NewRequestWithContext(ctx, http.MethodPut, sourceInfoURL, bytes.NewReader(sourceInfoJSON))
	if err == nil {
		sourceInfoReq.Header.Set("Content-Type", "application/json")
		if req.PeerPassword != "" {
			sourceInfoReq.Header.Set("X-Sync-Password", req.PeerPassword)
			sourceInfoReq.Header.Set("X-Source-Server", req.SourceServer)
		}
		// Send source info (ignore errors - it's just metadata)
		client.Do(sourceInfoReq)
	}

	// Log final upload stats
	if totalFiles > 0 {
		gbSynced := float64(totalBytes) / 1024.0 / 1024.0 / 1024.0
		logger.Info("Upload complete: files, GB synced", "total_files", totalFiles, "gb_synced", gbSynced)
	} else {
		logger.Info("✅ Sync complete: No changes detected")
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

// cleanupOrphanedFiles removes files on peer that don't exist in the local manifest
// This handles orphaned files that were left behind (e.g., from trash deletion)
func cleanupOrphanedFiles(ctx context.Context, client *http.Client, req *SyncRequest, localManifest *SyncManifest, shareName string) error {
	// Fetch list of physical files from peer
	listURL := fmt.Sprintf("https://%s:%d/api/sync/list-physical-files?source_server=%s&user_id=%d&share_name=%s",
		req.PeerAddress, req.PeerPort, url.QueryEscape(req.SourceServer), req.UserID, url.QueryEscape(shareName))

	listReq, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create list request: %w", err)
	}

	// Add authentication headers if password is provided
	if req.PeerPassword != "" {
		listReq.Header.Set("X-Sync-Password", req.PeerPassword)
		listReq.Header.Set("X-Source-Server", req.SourceServer)
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
		logger.Info("Found orphaned file(s) to clean up on peer", "orphaned_files", len(orphanedFiles))

		for _, orphanedFile := range orphanedFiles {
			deleteURL := fmt.Sprintf("https://%s:%d/api/sync/file?source_server=%s&user_id=%d&share_name=%s&path=%s",
				req.PeerAddress, req.PeerPort, url.QueryEscape(req.SourceServer), req.UserID, url.QueryEscape(shareName), url.QueryEscape(orphanedFile))

			deleteReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
			if err != nil {
				logger.Info("Failed to create delete request for orphaned file", "orphaned_file", orphanedFile, "error", err)
				continue
			}

			if req.PeerPassword != "" {
				deleteReq.Header.Set("X-Sync-Password", req.PeerPassword)
				deleteReq.Header.Set("X-Source-Server", req.SourceServer)
			}

			resp, err := client.Do(deleteReq)
			if err != nil {
				logger.Info("Failed to delete orphaned file", "orphaned_file", orphanedFile, "error", err)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				logger.Info("Failed to delete orphaned file", "orphaned_file", orphanedFile, "status_code", resp.StatusCode)
				continue
			}

			logger.Info("Deleted orphaned file", "orphaned_file", orphanedFile)
		}
	}

	return nil
}

// streamEncryptAndUpload encrypts and uploads a file using streaming to avoid loading entire file in RAM
// This prevents OOM (Out Of Memory) issues when syncing large files
func streamEncryptAndUpload(ctx context.Context, client *http.Client, file *os.File, req *SyncRequest, shareName, encryptedPath, encryptionKey string, userID int) error {
	// Create a pipe for streaming the complete multipart request
	pipeReader, pipeWriter := io.Pipe()

	// Create multipart writer
	mw := multipart.NewWriter(pipeWriter)
	contentType := mw.FormDataContentType()

	// Channel to capture errors from goroutine
	errChan := make(chan error, 1)

	// Goroutine to build multipart form with streamed encrypted data
	go func() {
		defer pipeWriter.Close()
		defer mw.Close()

		// Add metadata fields
		if err := mw.WriteField("user_id", fmt.Sprintf("%d", userID)); err != nil {
			errChan <- fmt.Errorf("failed to write user_id field: %w", err)
			return
		}
		if err := mw.WriteField("share_name", shareName); err != nil {
			errChan <- fmt.Errorf("failed to write share_name field: %w", err)
			return
		}
		if err := mw.WriteField("relative_path", encryptedPath); err != nil {
			errChan <- fmt.Errorf("failed to write relative_path field: %w", err)
			return
		}

		// Add file part
		part, err := mw.CreateFormFile("file", filepath.Base(encryptedPath))
		if err != nil {
			errChan <- fmt.Errorf("failed to create form file part: %w", err)
			return
		}

		// Encrypt and stream file directly into multipart (memory-efficient)
		if err := crypto.EncryptStream(file, part, encryptionKey); err != nil {
			errChan <- fmt.Errorf("encryption failed: %w", err)
			return
		}

		errChan <- nil
	}()

	// Upload file
	uploadURL := fmt.Sprintf("https://%s:%d/api/sync/file?source_server=%s", req.PeerAddress, req.PeerPort, url.QueryEscape(req.SourceServer))

	uploadReq, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, pipeReader)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	// Set correct content type with boundary
	uploadReq.Header.Set("Content-Type", contentType)

	// Add authentication headers if password is provided
	if req.PeerPassword != "" {
		uploadReq.Header.Set("X-Sync-Password", req.PeerPassword)
		uploadReq.Header.Set("X-Source-Server", req.SourceServer)
	}

	// Send request
	resp, err := client.Do(uploadReq)
	if err != nil {
		return fmt.Errorf("failed to send upload request: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors from goroutine
	if goroutineErr := <-errChan; goroutineErr != nil {
		return goroutineErr
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
