// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package restore provides file and share restoration from local and remote backups.
package restore

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/juste-un-gars/anemone/internal/crypto"
	"github.com/juste-un-gars/anemone/internal/sync"
)

// BackupInfo represents information about a backup
type BackupInfo struct {
	UserID       int       `json:"user_id"`
	ShareName    string    `json:"share_name"`
	Path         string    `json:"path"`
	LastModified time.Time `json:"last_modified"`
	FileCount    int       `json:"file_count"`
	TotalSize    int64     `json:"total_size"`
}

// FileNode represents a file or directory in the tree structure
type FileNode struct {
	Name        string                `json:"name"`
	Path        string                `json:"path"`
	IsDir       bool                  `json:"is_dir"`
	Size        int64                 `json:"size,omitempty"`
	ModTime     time.Time             `json:"mod_time,omitempty"`
	Children    map[string]*FileNode  `json:"children,omitempty"`
}

// ListUserBackups lists all available backups for a given user
func ListUserBackups(db *sql.DB, userID int, backupsDir string) ([]*BackupInfo, error) {
	// Get username for validation
	var username string
	err := db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Scan backups directory for directories matching {user_id}_*
	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*BackupInfo{}, nil // No backups yet
		}
		return nil, fmt.Errorf("failed to read backups directory: %w", err)
	}

	var backups []*BackupInfo
	prefix := fmt.Sprintf("%d_", userID)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if directory matches user's backups
		if !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}

		// Extract share name from directory name
		shareName := strings.TrimPrefix(entry.Name(), prefix)
		backupPath := filepath.Join(backupsDir, entry.Name())

		// Get backup info
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Count files and calculate total size
		fileCount, totalSize := scanBackupDir(backupPath)

		backups = append(backups, &BackupInfo{
			UserID:       userID,
			ShareName:    shareName,
			Path:         backupPath,
			LastModified: info.ModTime(),
			FileCount:    fileCount,
			TotalSize:    totalSize,
		})
	}

	// Sort by share name
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ShareName < backups[j].ShareName
	})

	return backups, nil
}

// scanBackupDir scans a backup directory and returns file count and total size
func scanBackupDir(path string) (int, int64) {
	var fileCount int
	var totalSize int64

	filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		fileCount++
		totalSize += info.Size()
		return nil
	})

	return fileCount, totalSize
}

// GetBackupManifest reads and decrypts a backup manifest
func GetBackupManifest(backupPath, userEncryptionKey string) (*sync.SyncManifest, error) {
	manifestPath := filepath.Join(backupPath, ".anemone-manifest.json.enc")

	// Check if manifest exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("manifest not found")
	}

	// Read encrypted manifest
	encryptedData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	// Decrypt manifest
	var decryptedBuf bytes.Buffer
	err = crypto.DecryptStream(bytes.NewReader(encryptedData), &decryptedBuf, userEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt manifest: %w", err)
	}

	// Parse JSON
	var manifest sync.SyncManifest
	if err := json.Unmarshal(decryptedBuf.Bytes(), &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// BuildFileTree creates a hierarchical tree structure from a flat manifest
func BuildFileTree(manifest *sync.SyncManifest) *FileNode {
	root := &FileNode{
		Name:     "/",
		Path:     "/",
		IsDir:    true,
		Children: make(map[string]*FileNode),
	}

	for filePath, metadata := range manifest.Files {
		parts := strings.Split(filePath, "/")
		currentNode := root

		// Navigate/create directory structure
		for i := 0; i < len(parts)-1; i++ {
			dirName := parts[i]
			if currentNode.Children == nil {
				currentNode.Children = make(map[string]*FileNode)
			}

			if _, exists := currentNode.Children[dirName]; !exists {
				currentNode.Children[dirName] = &FileNode{
					Name:     dirName,
					Path:     strings.Join(parts[:i+1], "/"),
					IsDir:    true,
					Children: make(map[string]*FileNode),
				}
			}
			currentNode = currentNode.Children[dirName]
		}

		// Add file node
		fileName := parts[len(parts)-1]
		if currentNode.Children == nil {
			currentNode.Children = make(map[string]*FileNode)
		}
		currentNode.Children[fileName] = &FileNode{
			Name:    fileName,
			Path:    filePath,
			IsDir:   false,
			Size:    metadata.Size,
			ModTime: metadata.ModTime,
		}
	}

	return root
}

// RestoreFile decrypts a file from a backup and writes it to a writer
func RestoreFile(backupPath, relativePath, userEncryptionKey string, writer io.Writer) error {
	// Build encrypted file path
	encryptedPath := filepath.Join(backupPath, relativePath+".enc")

	// Check if file exists
	if _, err := os.Stat(encryptedPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found in backup: %s", relativePath)
	}

	// Open encrypted file
	encryptedFile, err := os.Open(encryptedPath)
	if err != nil {
		return fmt.Errorf("failed to open encrypted file: %w", err)
	}
	defer encryptedFile.Close()

	// Decrypt and stream to writer
	err = crypto.DecryptStream(encryptedFile, writer, userEncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt file: %w", err)
	}

	return nil
}

// GetFileFromManifest retrieves file metadata from manifest
func GetFileFromManifest(manifest *sync.SyncManifest, relativePath string) (*sync.FileMetadata, error) {
	// Normalize path (use forward slashes)
	relativePath = filepath.ToSlash(relativePath)

	metadata, exists := manifest.Files[relativePath]
	if !exists {
		return nil, fmt.Errorf("file not found in manifest: %s", relativePath)
	}

	return &metadata, nil
}

// ListFilesInDirectory lists all files in a specific directory from the manifest
func ListFilesInDirectory(manifest *sync.SyncManifest, dirPath string) ([]string, error) {
	// Normalize directory path
	dirPath = filepath.ToSlash(dirPath)
	if dirPath == "" {
		dirPath = "."
	}
	if !strings.HasSuffix(dirPath, "/") && dirPath != "." {
		dirPath += "/"
	}

	var files []string
	for filePath := range manifest.Files {
		// Check if file is in this directory (not in subdirectories)
		if dirPath == "." {
			// Root directory - only files without '/'
			if !strings.Contains(filePath, "/") {
				files = append(files, filePath)
			}
		} else {
			// Check if file starts with directory path
			if strings.HasPrefix(filePath, dirPath) {
				relativePath := strings.TrimPrefix(filePath, dirPath)
				// Only include files directly in this directory (not in subdirectories)
				if !strings.Contains(relativePath, "/") {
					files = append(files, filePath)
				}
			}
		}
	}

	sort.Strings(files)
	return files, nil
}
