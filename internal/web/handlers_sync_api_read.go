// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// This file contains sync API handlers for read/list/download operations.

package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/juste-un-gars/anemone/internal/logger"
)

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
		logger.Info("Error listing files in", "backup_dir", backupDir, "error", err)
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

	logger.Info("Listed physical files for user , share", "files", len(files), "user_id", userID, "share_name", shareName)
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
	// Structure: {IncomingDir}/{source_server}/{user_id}_{share_name}/
	backupsDir := s.cfg.IncomingDir

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
		logger.Info("Error reading backups directory", "error", err)
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
	backupPath := filepath.Join(s.cfg.IncomingDir, sourceServer, backupDir)
	manifestPath := filepath.Join(backupPath, ".anemone-manifest.json.enc")

	// Check if manifest exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		http.Error(w, "Manifest not found", http.StatusNotFound)
		return
	}

	// Read encrypted manifest
	encryptedData, err := os.ReadFile(manifestPath)
	if err != nil {
		logger.Info("Error reading encrypted manifest", "error", err)
		http.Error(w, "Failed to read manifest", http.StatusInternalServerError)
		return
	}

	// Return encrypted manifest as-is
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\".anemone-manifest.json.enc\"")
	w.WriteHeader(http.StatusOK)
	w.Write(encryptedData)

	logger.Info("Sent encrypted manifest for user share", "user_id", userID, "share_name", shareName)
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
	backupPath := filepath.Join(s.cfg.IncomingDir, sourceServer, backupDir)

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
		logger.Info("Security: Attempted path traversal", "file_path", filePath, "rel_path", relPath)
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
		logger.Info("Error accessing file", "error", err)
		http.Error(w, "Failed to access file", http.StatusInternalServerError)
		return
	}

	// Open encrypted file
	encryptedFile, err := os.Open(encryptedFilePath)
	if err != nil {
		logger.Info("Error opening encrypted file", "error", err)
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

	logger.Info("Sent encrypted file for user share", "file_path", filePath, "user_id", userID, "share_name", shareName)
}
