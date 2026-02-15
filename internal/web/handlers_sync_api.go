// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/juste-un-gars/anemone/internal/crypto"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/shares"
	"github.com/juste-un-gars/anemone/internal/sync"
)

// handleAPISyncReceive receives and extracts a share archive from a peer
func (s *Server) handleAPISyncReceive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 10GB)
	if err := r.ParseMultipartForm(10 << 30); err != nil {
		logger.Info("Error parsing multipart form", "error", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get metadata from form
	userIDStr := r.FormValue("user_id")
	shareName := r.FormValue("share_name")

	if userIDStr == "" || shareName == "" {
		http.Error(w, "Missing user_id or share_name", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Find matching share in local database
	userShares, err := shares.GetByUser(s.db, userID)
	if err != nil {
		logger.Info("Error getting user shares", "error", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var targetShare *shares.Share
	for _, share := range userShares {
		if share.Name == shareName || share.Name == "backup_"+shareName || shareName == "backup" {
			targetShare = share
			break
		}
	}

	if targetShare == nil {
		logger.Info("No matching share found for user , share", "user_id", userID, "share_name", shareName)
		http.Error(w, "Share not found", http.StatusNotFound)
		return
	}

	// Get archive file
	file, _, err := r.FormFile("archive")
	if err != nil {
		logger.Info("Error getting archive file", "error", err)
		http.Error(w, "Missing archive file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check if archive is encrypted
	encrypted := r.FormValue("encrypted") == "true"

	var reader io.Reader = file
	if encrypted {
		// Get user's encryption key
		encryptionKey, err := sync.GetUserEncryptionKey(s.db, userID)
		if err != nil {
			logger.Info("Error getting encryption key", "error", err)
			http.Error(w, "Failed to get encryption key", http.StatusInternalServerError)
			return
		}

		// Decrypt archive
		var decryptedBuf bytes.Buffer
		if err := crypto.DecryptStream(file, &decryptedBuf, encryptionKey); err != nil {
			logger.Info("Error decrypting archive", "error", err)
			http.Error(w, fmt.Sprintf("Failed to decrypt archive: %v", err), http.StatusInternalServerError)
			return
		}
		reader = &decryptedBuf
	}

	// Extract archive to local share path
	if err := sync.ExtractTarGz(reader, targetShare.Path); err != nil {
		logger.Info("Error extracting archive", "error", err)
		http.Error(w, fmt.Sprintf("Failed to extract archive: %v", err), http.StatusInternalServerError)
		return
	}

	logger.Info("Successfully received and extracted sync to: (user , share )", "path", targetShare.Path, "user_id", userID, "share_name", shareName)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "message": "Sync received and extracted"}`)
}

// handleAPISyncManifest handles GET and PUT requests for sync manifests
func (s *Server) handleAPISyncManifest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleAPISyncManifestGet(w, r)
	case http.MethodPut:
		s.handleAPISyncManifestPut(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAPISyncManifestGet returns the manifest for a given share
// GET /api/sync/manifest?user_id=5&share_name=backup
func (s *Server) handleAPISyncManifestGet(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	sourceServer := r.URL.Query().Get("source_server")
	if sourceServer == "" {
		sourceServer = "unknown"
	}
	userIDStr := r.URL.Query().Get("user_id")
	shareName := r.URL.Query().Get("share_name")

	if userIDStr == "" || shareName == "" {
		http.Error(w, "Missing user_id or share_name", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Build backup directory path with source server separation
	// Format: {IncomingDir}/{source_server}/{user_id}_{share_name}/
	backupDirName := fmt.Sprintf("%d_%s", userID, shareName)
	backupDir := filepath.Join(s.cfg.IncomingDir, sourceServer, backupDirName)
	manifestPath := filepath.Join(backupDir, ".anemone-manifest.json.enc")

	// Check if manifest file exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		// No manifest yet (first sync) - return 404
		http.Error(w, "No manifest found (first sync)", http.StatusNotFound)
		return
	}

	// Read encrypted manifest
	encryptedData, err := os.ReadFile(manifestPath)
	if err != nil {
		logger.Info("Error reading manifest file", "error", err)
		http.Error(w, "Failed to read manifest", http.StatusInternalServerError)
		return
	}

	// Return encrypted manifest
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\".anemone-manifest.json.enc\"")
	w.WriteHeader(http.StatusOK)
	w.Write(encryptedData)
}

// handleAPISyncManifestPut updates the manifest for a given share
// PUT /api/sync/manifest?user_id=5&share_name=backup
// Body: encrypted manifest data
func (s *Server) handleAPISyncManifestPut(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	sourceServer := r.URL.Query().Get("source_server")
	if sourceServer == "" {
		sourceServer = "unknown"
	}
	userIDStr := r.URL.Query().Get("user_id")
	shareName := r.URL.Query().Get("share_name")

	if userIDStr == "" || shareName == "" {
		http.Error(w, "Missing user_id or share_name", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Security check: prevent path traversal
	if isPathTraversal(sourceServer) {
		http.Error(w, "Invalid source_server (path traversal detected)", http.StatusBadRequest)
		return
	}

	// Read encrypted manifest from request body
	encryptedData, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Info("Error reading request body", "error", err)
		http.Error(w, "Failed to read manifest data", http.StatusBadRequest)
		return
	}

	// Build backup directory path with source server separation
	backupDirName := fmt.Sprintf("%d_%s", userID, shareName)
	backupDir := filepath.Join(s.cfg.IncomingDir, sourceServer, backupDirName)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		logger.Info("Error creating backup directory", "error", err)
		http.Error(w, "Failed to create backup directory", http.StatusInternalServerError)
		return
	}

	// Write encrypted manifest
	manifestPath := filepath.Join(backupDir, ".anemone-manifest.json.enc")
	if err := os.WriteFile(manifestPath, encryptedData, 0644); err != nil {
		logger.Info("Error writing manifest file", "error", err)
		http.Error(w, "Failed to write manifest", http.StatusInternalServerError)
		return
	}

	logger.Info("Successfully updated manifest for user , share", "user_id", userID, "share_name", shareName)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "message": "Manifest updated"}`)
}

// handleAPISyncSourceInfo handles PUT request to store source server information
func (s *Server) handleAPISyncSourceInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	userIDStr := r.URL.Query().Get("user_id")
	shareName := r.URL.Query().Get("share_name")

	if userIDStr == "" || shareName == "" {
		http.Error(w, "Missing user_id or share_name", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Read JSON data from request body
	sourceInfoData, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Info("Error reading request body", "error", err)
		http.Error(w, "Failed to read source info data", http.StatusBadRequest)
		return
	}

	// Extract source_server from query parameter
	sourceServer := r.URL.Query().Get("source_server")
	if sourceServer == "" {
		sourceServer = "unknown"
	}

	// Build backup directory path with source server separation
	backupDirName := fmt.Sprintf("%d_%s", userID, shareName)
	backupDir := filepath.Join(s.cfg.IncomingDir, sourceServer, backupDirName)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		logger.Info("Error creating backup directory", "error", err)
		http.Error(w, "Failed to create backup directory", http.StatusInternalServerError)
		return
	}

	// Write source info file (unencrypted metadata)
	sourceInfoPath := filepath.Join(backupDir, ".source-info.json")
	if err := os.WriteFile(sourceInfoPath, sourceInfoData, 0644); err != nil {
		logger.Info("Error writing source info file", "error", err)
		http.Error(w, "Failed to write source info", http.StatusInternalServerError)
		return
	}

	logger.Info("Successfully updated source info for user , share", "user_id", userID, "share_name", shareName)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "message": "Source info updated"}`)
}

// handleAPISyncFile handles POST and DELETE requests for individual files
func (s *Server) handleAPISyncFile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleAPISyncFileUpload(w, r)
	case http.MethodDelete:
		s.handleAPISyncFileDelete(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAPISyncFileUpload handles uploading a single encrypted file
// POST /api/sync/file?source_server=X
// Multipart form with: user_id, share_name, relative_path, file
func (s *Server) handleAPISyncFileUpload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 10GB)
	if err := r.ParseMultipartForm(10 << 30); err != nil {
		logger.Info("Error parsing multipart form", "error", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get source_server from query params
	sourceServer := r.URL.Query().Get("source_server")
	if sourceServer == "" {
		sourceServer = "unknown"
	}

	// Get metadata from form
	userIDStr := r.FormValue("user_id")
	shareName := r.FormValue("share_name")
	relativePath := r.FormValue("relative_path")

	if userIDStr == "" || shareName == "" || relativePath == "" {
		http.Error(w, "Missing user_id, share_name, or relative_path", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Security check: prevent path traversal
	if isPathTraversal(relativePath) || isPathTraversal(sourceServer) {
		http.Error(w, "Invalid relative_path (path traversal detected)", http.StatusBadRequest)
		return
	}

	// Get file from multipart form
	file, _, err := r.FormFile("file")
	if err != nil {
		logger.Info("Error getting file", "error", err)
		http.Error(w, "Missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Build backup directory path with source server separation
	backupDirName := fmt.Sprintf("%d_%s", userID, shareName)
	backupDir := filepath.Join(s.cfg.IncomingDir, sourceServer, backupDirName)
	targetPath := filepath.Join(backupDir, relativePath)

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		logger.Info("Error creating directory", "error", err)
		http.Error(w, "Failed to create directory", http.StatusInternalServerError)
		return
	}

	// Write file to disk
	outFile, err := os.Create(targetPath)
	if err != nil {
		logger.Info("Error creating file", "error", err)
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, file); err != nil {
		logger.Info("Error writing file", "error", err)
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	logger.Info("Successfully uploaded file: (user , share )", "relative_path", relativePath, "user_id", userID, "share_name", shareName)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "message": "File uploaded"}`)
}

// handleAPISyncFileDelete handles deleting a single file from backup
// DELETE /api/sync/file?user_id=5&share_name=backup&path=documents/report.pdf.enc
func (s *Server) handleAPISyncFileDelete(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	sourceServer := r.URL.Query().Get("source_server")
	if sourceServer == "" {
		sourceServer = "unknown"
	}
	userIDStr := r.URL.Query().Get("user_id")
	shareName := r.URL.Query().Get("share_name")
	relativePath := r.URL.Query().Get("path")

	if userIDStr == "" || shareName == "" || relativePath == "" {
		http.Error(w, "Missing user_id, share_name, or path", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Security check: prevent path traversal
	if isPathTraversal(relativePath) || isPathTraversal(sourceServer) {
		http.Error(w, "Invalid path (path traversal detected)", http.StatusBadRequest)
		return
	}

	// Build backup directory path with source server separation
	backupDirName := fmt.Sprintf("%d_%s", userID, shareName)
	backupDir := filepath.Join(s.cfg.IncomingDir, sourceServer, backupDirName)
	targetPath := filepath.Join(backupDir, relativePath)

	// Delete file
	if err := os.Remove(targetPath); err != nil {
		if os.IsNotExist(err) {
			// File already doesn't exist - that's OK
			logger.Info("File already deleted", "relative_path", relativePath)
		} else {
			logger.Info("Error deleting file", "error", err)
			http.Error(w, "Failed to delete file", http.StatusInternalServerError)
			return
		}
	}

	// Clean up empty parent directories
	cleanEmptyParentDirs(filepath.Dir(targetPath), backupDir)

	logger.Info("Successfully deleted file: (user , share )", "relative_path", relativePath, "user_id", userID, "share_name", shareName)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "message": "File deleted"}`)
}


// handleAPISyncDeleteUserBackup deletes all backup data for a user on this peer
// DELETE /api/sync/delete-user-backup?source_server=X&user_id=5
func (s *Server) handleAPISyncDeleteUserBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	sourceServer := r.URL.Query().Get("source_server")
	if sourceServer == "" {
		sourceServer = "unknown"
	}
	userIDStr := r.URL.Query().Get("user_id")

	if userIDStr == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Security check: prevent path traversal
	if isPathTraversal(sourceServer) {
		http.Error(w, "Invalid source_server (path traversal detected)", http.StatusBadRequest)
		return
	}

	// Build backup directory path for this user from this source server
	// Format: {IncomingDir}/{source_server}/{user_id}_*
	incomingDir := filepath.Join(s.cfg.IncomingDir, sourceServer)

	// Check if incoming directory for this source server exists
	if _, err := os.Stat(incomingDir); os.IsNotExist(err) {
		// No backups from this server - that's OK
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"success": true, "message": "No backups found for this user", "deleted_directories": 0}`)
		return
	}

	// Find all backup directories for this user (user_id_*)
	pattern := fmt.Sprintf("%d_*", userID)
	matches, err := filepath.Glob(filepath.Join(incomingDir, pattern))
	if err != nil {
		logger.Info("Error globbing backup directories", "error", err)
		http.Error(w, "Failed to find backup directories", http.StatusInternalServerError)
		return
	}

	// Delete all matching directories
	deletedCount := 0
	for _, backupDir := range matches {
		// Delete the entire backup directory (using sudo for permission)
		cmd := exec.Command("sudo", "rm", "-rf", backupDir)
		if err := cmd.Run(); err != nil {
			logger.Info("Warning: failed to delete backup directory", "backup_dir", backupDir, "error", err)
			continue
		}
		deletedCount++
		logger.Info("Deleted backup directory", "backup_dir", backupDir)
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "message": "User backup deleted", "deleted_directories": %d}`, deletedCount)

	logger.Info("Deleted backup director(ies) for user from source", "deleted_count", deletedCount, "user_id", userID, "source_server", sourceServer)
}



// cleanEmptyParentDirs removes empty parent directories up to (but not including) the stopDir.
// This is called after deleting a file to clean up any empty directories left behind.
func cleanEmptyParentDirs(dir, stopDir string) {
	// Normalize paths for comparison
	stopDir = filepath.Clean(stopDir)

	for {
		dir = filepath.Clean(dir)

		// Stop if we've reached or passed the stop directory
		if dir == stopDir || len(dir) <= len(stopDir) {
			return
		}

		// Check if directory is empty
		entries, err := os.ReadDir(dir)
		if err != nil {
			// Can't read directory, stop
			return
		}

		if len(entries) > 0 {
			// Directory is not empty, stop
			return
		}

		// Try to remove empty directory
		if err := os.Remove(dir); err != nil {
			// Failed to remove (permissions, etc.), stop
			logger.Info("Could not remove empty directory", "dir", dir, "error", err)
			return
		}

		logger.Info("Removed empty directory", "dir", dir)

		// Move up to parent directory
		dir = filepath.Dir(dir)
	}
}
