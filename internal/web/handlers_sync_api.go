// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/juste-un-gars/anemone/internal/crypto"
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
		log.Printf("Error parsing multipart form: %v", err)
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
		log.Printf("Error getting user shares: %v", err)
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
		log.Printf("No matching share found for user %d, share %s", userID, shareName)
		http.Error(w, "Share not found", http.StatusNotFound)
		return
	}

	// Get archive file
	file, _, err := r.FormFile("archive")
	if err != nil {
		log.Printf("Error getting archive file: %v", err)
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
			log.Printf("Error getting encryption key: %v", err)
			http.Error(w, "Failed to get encryption key", http.StatusInternalServerError)
			return
		}

		// Decrypt archive
		var decryptedBuf bytes.Buffer
		if err := crypto.DecryptStream(file, &decryptedBuf, encryptionKey); err != nil {
			log.Printf("Error decrypting archive: %v", err)
			http.Error(w, fmt.Sprintf("Failed to decrypt archive: %v", err), http.StatusInternalServerError)
			return
		}
		reader = &decryptedBuf
	}

	// Extract archive to local share path
	if err := sync.ExtractTarGz(reader, targetShare.Path); err != nil {
		log.Printf("Error extracting archive: %v", err)
		http.Error(w, fmt.Sprintf("Failed to extract archive: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully received and extracted sync to: %s (user %d, share %s)", targetShare.Path, userID, shareName)

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
		log.Printf("Error reading manifest file: %v", err)
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
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Failed to read manifest data", http.StatusBadRequest)
		return
	}

	// Build backup directory path with source server separation
	backupDirName := fmt.Sprintf("%d_%s", userID, shareName)
	backupDir := filepath.Join(s.cfg.IncomingDir, sourceServer, backupDirName)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		log.Printf("Error creating backup directory: %v", err)
		http.Error(w, "Failed to create backup directory", http.StatusInternalServerError)
		return
	}

	// Write encrypted manifest
	manifestPath := filepath.Join(backupDir, ".anemone-manifest.json.enc")
	if err := os.WriteFile(manifestPath, encryptedData, 0644); err != nil {
		log.Printf("Error writing manifest file: %v", err)
		http.Error(w, "Failed to write manifest", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully updated manifest for user %d, share %s", userID, shareName)

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
		log.Printf("Error reading request body: %v", err)
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
		log.Printf("Error creating backup directory: %v", err)
		http.Error(w, "Failed to create backup directory", http.StatusInternalServerError)
		return
	}

	// Write source info file (unencrypted metadata)
	sourceInfoPath := filepath.Join(backupDir, ".source-info.json")
	if err := os.WriteFile(sourceInfoPath, sourceInfoData, 0644); err != nil {
		log.Printf("Error writing source info file: %v", err)
		http.Error(w, "Failed to write source info", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully updated source info for user %d, share %s", userID, shareName)

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
		log.Printf("Error parsing multipart form: %v", err)
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
		log.Printf("Error getting file: %v", err)
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
		log.Printf("Error creating directory: %v", err)
		http.Error(w, "Failed to create directory", http.StatusInternalServerError)
		return
	}

	// Write file to disk
	outFile, err := os.Create(targetPath)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, file); err != nil {
		log.Printf("Error writing file: %v", err)
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully uploaded file: %s (user %d, share %s)", relativePath, userID, shareName)

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
			log.Printf("File already deleted: %s", relativePath)
		} else {
			log.Printf("Error deleting file: %v", err)
			http.Error(w, "Failed to delete file", http.StatusInternalServerError)
			return
		}
	}

	// Clean up empty parent directories
	cleanEmptyParentDirs(filepath.Dir(targetPath), backupDir)

	log.Printf("Successfully deleted file: %s (user %d, share %s)", relativePath, userID, shareName)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "message": "File deleted"}`)
}

// handleAPISyncListPhysicalFiles lists all physical .enc files in a backup directory
// GET /api/sync/list-physical-files?source_server=X&user_id=5&share_name=backup
func (s *Server) handleAPISyncListPhysicalFiles(w http.ResponseWriter, r *http.Request) {
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

	// Build backup directory path with source server separation
	backupDirName := fmt.Sprintf("%d_%s", userID, shareName)
	backupDir := filepath.Join(s.cfg.IncomingDir, sourceServer, backupDirName)

	// Check if backup directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		// No backup directory yet - return empty list
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"files":[]}`)
		return
	}

	// List all .enc files recursively
	var files []string
	err = filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip manifest and source info files
		if strings.HasPrefix(filepath.Base(path), ".anemone-") ||
		   strings.HasPrefix(filepath.Base(path), ".source-") {
			return nil
		}

		// Only include .enc files
		if strings.HasSuffix(path, ".enc") {
			// Get relative path from backup directory
			relPath, err := filepath.Rel(backupDir, path)
			if err != nil {
				return err
			}
			// Use forward slashes for consistency
			relPath = filepath.ToSlash(relPath)
			files = append(files, relPath)
		}

		return nil
	})

	if err != nil {
		log.Printf("Error listing files in %s: %v", backupDir, err)
		http.Error(w, "Failed to list files", http.StatusInternalServerError)
		return
	}

	// Return list of files as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Marshal file list to JSON
	filesJSON, err := json.Marshal(map[string][]string{"files": files})
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Write(filesJSON)

	log.Printf("Listed %d physical files for user %d, share %s", len(files), userID, shareName)
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
		log.Printf("Error globbing backup directories: %v", err)
		http.Error(w, "Failed to find backup directories", http.StatusInternalServerError)
		return
	}

	// Delete all matching directories
	deletedCount := 0
	for _, backupDir := range matches {
		// Delete the entire backup directory (using sudo for permission)
		cmd := exec.Command("sudo", "rm", "-rf", backupDir)
		if err := cmd.Run(); err != nil {
			log.Printf("Warning: failed to delete backup directory %s: %v", backupDir, err)
			continue
		}
		deletedCount++
		log.Printf("Deleted backup directory: %s", backupDir)
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "message": "User backup deleted", "deleted_directories": %d}`, deletedCount)

	log.Printf("Deleted %d backup director(ies) for user %d from source %s", deletedCount, userID, sourceServer)
}


// handleAPISyncListUserBackups lists available backups for a given user on this peer
// GET /api/sync/list-user-backups?user_id=X
// This endpoint is called by the origin server to discover backups stored on this peer
func (s *Server) handleAPISyncListUserBackups(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "Missing user_id parameter", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Scan incoming backups directory for this user
	// Structure: /backups/incoming/{source_server}/{user_id}_{share_name}/
	backupsDir := filepath.Join(s.cfg.DataDir, "backups", "incoming")

	type BackupInfo struct {
		SourceServer string    `json:"source_server"`
		ShareName    string    `json:"share_name"`
		FileCount    int       `json:"file_count"`
		TotalSize    int64     `json:"total_size"`
		LastModified time.Time `json:"last_modified"`
	}

	var backups []BackupInfo
	prefix := fmt.Sprintf("%d_", userID)

	// First level: read source server directories
	serverEntries, err := os.ReadDir(backupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			// No backups directory yet
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
			return
		}
		log.Printf("Error reading backups directory: %v", err)
		http.Error(w, "Failed to read backups directory", http.StatusInternalServerError)
		return
	}

	// Iterate over each source server directory
	for _, serverEntry := range serverEntries {
		if !serverEntry.IsDir() {
			continue
		}

		serverDir := filepath.Join(backupsDir, serverEntry.Name())

		// Second level: read backup directories for this source server
		backupEntries, err := os.ReadDir(serverDir)
		if err != nil {
			continue // Skip if we can't read this server's directory
		}

		for _, entry := range backupEntries {
			if !entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
				continue
			}

			// Extract username from directory name: {user_id}_{username} -> backup_{username}
			username := strings.TrimPrefix(entry.Name(), prefix)
			shareName := "backup_" + username
			backupPath := filepath.Join(serverDir, entry.Name())

			// Get modification time
			info, err := entry.Info()
			if err != nil {
				continue
			}

			// Count files and size
			var fileCount int
			var totalSize int64
			filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				fileCount++
				totalSize += info.Size()
				return nil
			})

			backups = append(backups, BackupInfo{
				SourceServer: serverEntry.Name(),
				ShareName:    shareName,
				FileCount:    fileCount,
				TotalSize:    totalSize,
				LastModified: info.ModTime(),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backups)
}

// handleAPISyncDownloadEncryptedManifest downloads the encrypted manifest without decrypting it
// GET /api/sync/download-encrypted-manifest?user_id=X&share_name=Y&source_server=Z
// Returns the .anemone-manifest.json.enc file as-is (encrypted)
func (s *Server) handleAPISyncDownloadEncryptedManifest(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	shareName := r.URL.Query().Get("share_name")
	sourceServer := r.URL.Query().Get("source_server")

	if userIDStr == "" || shareName == "" || sourceServer == "" {
		http.Error(w, "Missing user_id, share_name, or source_server", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Build backup path
	// Convert share name to directory name (e.g., "backup_test" -> "test")
	// Convention: incoming/{source_server}/{user_id}_{username}/ but API uses backup_{username}
	username := shareName
	if strings.HasPrefix(shareName, "backup_") {
		username = strings.TrimPrefix(shareName, "backup_")
	} else if strings.HasPrefix(shareName, "data_") {
		username = strings.TrimPrefix(shareName, "data_")
	}
	backupDir := fmt.Sprintf("%d_%s", userID, username)
	backupPath := filepath.Join(s.cfg.DataDir, "backups", "incoming", sourceServer, backupDir)
	manifestPath := filepath.Join(backupPath, ".anemone-manifest.json.enc")

	// Check if manifest exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		http.Error(w, "Manifest not found", http.StatusNotFound)
		return
	}

	// Read encrypted manifest
	encryptedData, err := os.ReadFile(manifestPath)
	if err != nil {
		log.Printf("Error reading encrypted manifest: %v", err)
		http.Error(w, "Failed to read manifest", http.StatusInternalServerError)
		return
	}

	// Return encrypted manifest as-is
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\".anemone-manifest.json.enc\"")
	w.WriteHeader(http.StatusOK)
	w.Write(encryptedData)

	log.Printf("Sent encrypted manifest for user %d share %s", userID, shareName)
}

// handleAPISyncDownloadEncryptedFile downloads an encrypted file without decrypting it
// GET /api/sync/download-encrypted-file?user_id=X&share_name=Y&path=Z&source_server=W
// Returns the encrypted file as-is (with .enc extension)
func (s *Server) handleAPISyncDownloadEncryptedFile(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	shareName := r.URL.Query().Get("share_name")
	filePath := r.URL.Query().Get("path")
	sourceServer := r.URL.Query().Get("source_server")

	if userIDStr == "" || shareName == "" || filePath == "" || sourceServer == "" {
		http.Error(w, "Missing user_id, share_name, path, or source_server", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Build backup path
	// Convert share name to directory name (e.g., "backup_test" -> "test")
	// Convention: incoming/{source_server}/{user_id}_{username}/ but API uses backup_{username}
	username := shareName
	if strings.HasPrefix(shareName, "backup_") {
		username = strings.TrimPrefix(shareName, "backup_")
	} else if strings.HasPrefix(shareName, "data_") {
		username = strings.TrimPrefix(shareName, "data_")
	}
	backupDir := fmt.Sprintf("%d_%s", userID, username)
	backupPath := filepath.Join(s.cfg.DataDir, "backups", "incoming", sourceServer, backupDir)

	// Build encrypted file path
	encryptedFilePath := filepath.Join(backupPath, filePath+".enc")

	// Security check: ensure path is within backup directory
	absBackupPath, err := filepath.Abs(backupPath)
	if err != nil {
		http.Error(w, "Invalid backup path", http.StatusBadRequest)
		return
	}
	absFilePath, err := filepath.Abs(encryptedFilePath)
	if err != nil {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}
	// Use filepath.Rel to properly check if path is within directory
	// This prevents path traversal attacks like /srv/anemone/backups_evil/../etc/passwd
	relPath, err := filepath.Rel(absBackupPath, absFilePath)
	if err != nil || strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
		log.Printf("Security: Attempted path traversal: %s (relative: %s)", filePath, relPath)
		http.Error(w, "Invalid file path", http.StatusForbidden)
		return
	}

	// Check if file exists
	fileInfo, err := os.Stat(encryptedFilePath)
	if os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error accessing file: %v", err)
		http.Error(w, "Failed to access file", http.StatusInternalServerError)
		return
	}

	// Open encrypted file
	encryptedFile, err := os.Open(encryptedFilePath)
	if err != nil {
		log.Printf("Error opening encrypted file: %v", err)
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		return
	}
	defer encryptedFile.Close()

	// Return encrypted file as-is
	fileName := filepath.Base(filePath) + ".enc"
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	w.WriteHeader(http.StatusOK)

	// Stream the encrypted file
	io.Copy(w, encryptedFile)

	log.Printf("Sent encrypted file %s for user %d share %s", filePath, userID, shareName)
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
			log.Printf("Could not remove empty directory %s: %v", dir, err)
			return
		}

		log.Printf("Removed empty directory: %s", dir)

		// Move up to parent directory
		dir = filepath.Dir(dir)
	}
}
