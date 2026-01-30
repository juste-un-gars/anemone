// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package usbbackup

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"github.com/juste-un-gars/anemone/internal/logger"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/juste-un-gars/anemone/internal/crypto"
	"github.com/juste-un-gars/anemone/internal/shares"
)

// FileMetadata represents metadata for a single file
type FileMetadata struct {
	Size          int64     `json:"size"`
	ModTime       time.Time `json:"mtime"`
	Checksum      string    `json:"checksum"`
	EncryptedName string    `json:"encrypted_name"`
}

// BackupManifest represents the manifest of backed up files
type BackupManifest struct {
	Version      int                     `json:"version"`
	LastSync     time.Time               `json:"last_sync"`
	UserID       int                     `json:"user_id"`
	ShareName    string                  `json:"share_name"`
	SourceServer string                  `json:"source_server"`
	Files        map[string]FileMetadata `json:"files"`
}

// SyncResult contains the result of a sync operation
type SyncResult struct {
	FilesAdded   int
	FilesUpdated int
	FilesDeleted int
	BytesSynced  int64
	Errors       []string
}

// ConfigBackupInfo contains paths for config backup
type ConfigBackupInfo struct {
	DataDir  string // Base data directory (e.g., /srv/anemone)
	DBPath   string // Path to database file
	CertsDir string // Path to certificates directory
	SMBConf  string // Path to smb.conf
}

// SyncConfig backs up only the configuration (DB, certs, smb.conf)
// This is a lightweight backup that fits on any USB drive
func SyncConfig(db *sql.DB, backup *USBBackup, configInfo *ConfigBackupInfo, masterKey string, serverName string) (*SyncResult, error) {
	if !backup.IsMounted() {
		return nil, fmt.Errorf("backup drive not mounted: %s", backup.MountPath)
	}

	if err := backup.EnsureBackupDir(); err != nil {
		return nil, err
	}

	// Update status to running
	UpdateSyncStatus(db, backup.ID, "running", "", 0, 0)

	result := &SyncResult{}

	// Config backup directory
	configDir := filepath.Join(backup.GetFullBackupPath(), "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		UpdateSyncStatus(db, backup.ID, "error", err.Error(), 0, 0)
		return nil, fmt.Errorf("failed to create config backup directory: %w", err)
	}

	// 1. Backup database (encrypted)
	if configInfo.DBPath != "" {
		dbDest := filepath.Join(configDir, "anemone.db.enc")
		bytesCopied, err := copyFileEncrypted(configInfo.DBPath, dbDest, masterKey)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("database: %v", err))
			logger.Info("⚠️  USB backup: failed to backup database: %v", err)
		} else {
			result.FilesAdded++
			result.BytesSynced += bytesCopied
			logger.Info("✅ USB backup: database backed up (%s)", FormatBytes(bytesCopied))
		}
	}

	// 2. Backup certificates directory (encrypted)
	if configInfo.CertsDir != "" {
		certsDestDir := filepath.Join(configDir, "certs")
		if err := os.MkdirAll(certsDestDir, 0755); err == nil {
			err := filepath.Walk(configInfo.CertsDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				relPath, _ := filepath.Rel(configInfo.CertsDir, path)
				destPath := filepath.Join(certsDestDir, relPath+".enc")

				// Ensure parent directory exists
				os.MkdirAll(filepath.Dir(destPath), 0755)

				bytesCopied, copyErr := copyFileEncrypted(path, destPath, masterKey)
				if copyErr != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("cert %s: %v", relPath, copyErr))
				} else {
					result.FilesAdded++
					result.BytesSynced += bytesCopied
				}
				return nil
			})
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("certs dir: %v", err))
			} else {
				logger.Info("✅ USB backup: certificates backed up")
			}
		}
	}

	// 3. Backup smb.conf (encrypted)
	if configInfo.SMBConf != "" {
		if _, err := os.Stat(configInfo.SMBConf); err == nil {
			smbDest := filepath.Join(configDir, "smb.conf.enc")
			bytesCopied, err := copyFileEncrypted(configInfo.SMBConf, smbDest, masterKey)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("smb.conf: %v", err))
				logger.Info("⚠️  USB backup: failed to backup smb.conf: %v", err)
			} else {
				result.FilesAdded++
				result.BytesSynced += bytesCopied
				logger.Info("✅ USB backup: smb.conf backed up")
			}
		}
	}

	// Save config manifest
	configManifest := map[string]interface{}{
		"version":       1,
		"backup_type":   "config",
		"source_server": serverName,
		"timestamp":     time.Now().Format(time.RFC3339),
		"files_count":   result.FilesAdded,
		"bytes_total":   result.BytesSynced,
	}
	manifestData, _ := json.MarshalIndent(configManifest, "", "  ")
	manifestPath := filepath.Join(configDir, ".anemone-config-manifest.json")
	os.WriteFile(manifestPath, manifestData, 0600)

	// Update final status
	if len(result.Errors) > 0 {
		errMsg := strings.Join(result.Errors, "; ")
		UpdateSyncStatus(db, backup.ID, "error", errMsg, result.FilesAdded, result.BytesSynced)
	} else {
		UpdateSyncStatus(db, backup.ID, "success", "", result.FilesAdded, result.BytesSynced)
	}

	logger.Info("📦 USB config backup completed: %d files, %s", result.FilesAdded, FormatBytes(result.BytesSynced))
	return result, nil
}

// SyncAllShares backs up selected shares to the USB drive
// Respects backup.SelectedShares - if empty, backs up all shares with sync_enabled
func SyncAllShares(db *sql.DB, backup *USBBackup, masterKey string, serverName string) (*SyncResult, error) {
	if !backup.IsMounted() {
		return nil, fmt.Errorf("backup drive not mounted: %s", backup.MountPath)
	}

	if err := backup.EnsureBackupDir(); err != nil {
		return nil, err
	}

	// Update status to running
	UpdateSyncStatus(db, backup.ID, "running", "", 0, 0)

	// Get all shares
	allShares, err := shares.GetAll(db)
	if err != nil {
		UpdateSyncStatus(db, backup.ID, "error", err.Error(), 0, 0)
		return nil, fmt.Errorf("failed to get shares: %w", err)
	}

	result := &SyncResult{}
	selectedIDs := backup.GetSelectedShareIDs()
	sharesBackedUp := 0

	for _, share := range allShares {
		// Check if share should be backed up
		// If selectedIDs is empty, backup all shares with sync_enabled
		// If selectedIDs is not empty, only backup selected shares (ignore sync_enabled)
		shouldBackup := false
		if len(selectedIDs) == 0 {
			// No selection = all shares with sync_enabled
			shouldBackup = share.SyncEnabled
		} else {
			// Specific selection = only selected shares
			shouldBackup = backup.IsShareSelected(share.ID)
		}

		if !shouldBackup {
			continue
		}

		logger.Info("📂 USB backup: syncing share %s (ID: %d)", share.Name, share.ID)

		shareResult, err := syncShare(db, backup, share, masterKey, serverName)
		if err != nil {
			errMsg := fmt.Sprintf("share %s: %v", share.Name, err)
			result.Errors = append(result.Errors, errMsg)
			logger.Info("⚠️  USB backup error for share %s: %v", share.Name, err)
			continue
		}

		result.FilesAdded += shareResult.FilesAdded
		result.FilesUpdated += shareResult.FilesUpdated
		result.FilesDeleted += shareResult.FilesDeleted
		result.BytesSynced += shareResult.BytesSynced
		sharesBackedUp++
	}

	// Update final status
	totalFiles := result.FilesAdded + result.FilesUpdated
	if len(result.Errors) > 0 {
		errMsg := strings.Join(result.Errors, "; ")
		UpdateSyncStatus(db, backup.ID, "error", errMsg, totalFiles, result.BytesSynced)
	} else {
		UpdateSyncStatus(db, backup.ID, "success", "", totalFiles, result.BytesSynced)
	}

	logger.Info("📦 USB backup completed: %d shares, %d files, %s",
		sharesBackedUp, totalFiles, FormatBytes(result.BytesSynced))

	return result, nil
}

// syncShare backs up a single share to the USB drive
func syncShare(db *sql.DB, backup *USBBackup, share *shares.Share, masterKey string, serverName string) (*SyncResult, error) {
	result := &SyncResult{}

	// Destination directory: {backup_path}/{user_id}_{share_name}/
	destDir := filepath.Join(backup.GetFullBackupPath(), fmt.Sprintf("%d_%s", share.UserID, share.Name))
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Build local manifest
	localManifest, err := buildManifest(share.Path, share.UserID, share.Name, serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to build manifest: %w", err)
	}

	// Load remote manifest from USB
	remoteManifest, err := loadManifest(destDir)
	if err != nil {
		logger.Info("📦 No existing manifest on USB for %s, full backup needed", share.Name)
		remoteManifest = &BackupManifest{Files: make(map[string]FileMetadata)}
	}

	// Calculate delta
	toAdd, toUpdate, toDelete := compareManifests(localManifest, remoteManifest)

	logger.Info("📊 Share %s: %d to add, %d to update, %d to delete",
		share.Name, len(toAdd), len(toUpdate), len(toDelete))

	// Copy new and updated files (encrypted)
	for _, relPath := range append(toAdd, toUpdate...) {
		srcPath := filepath.Join(share.Path, relPath)

		// Generate encrypted filename
		encName := generateEncryptedName(relPath)
		destPath := filepath.Join(destDir, encName)

		bytesCopied, err := copyFileEncrypted(srcPath, destPath, masterKey)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", relPath, err))
			continue
		}

		result.BytesSynced += bytesCopied
		if contains(toAdd, relPath) {
			result.FilesAdded++
		} else {
			result.FilesUpdated++
		}
	}

	// Delete obsolete files
	for _, relPath := range toDelete {
		meta, ok := remoteManifest.Files[relPath]
		if !ok {
			continue
		}
		destPath := filepath.Join(destDir, meta.EncryptedName)
		if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
			result.Errors = append(result.Errors, fmt.Sprintf("delete %s: %v", relPath, err))
			continue
		}
		result.FilesDeleted++
	}

	// Update manifest with new file info
	for relPath, meta := range localManifest.Files {
		remoteManifest.Files[relPath] = meta
	}
	for _, relPath := range toDelete {
		delete(remoteManifest.Files, relPath)
	}
	remoteManifest.LastSync = time.Now()
	remoteManifest.SourceServer = serverName

	// Save updated manifest (encrypted)
	if err := saveManifest(remoteManifest, destDir, masterKey); err != nil {
		return nil, fmt.Errorf("failed to save manifest: %w", err)
	}

	return result, nil
}

// buildManifest scans a directory and creates a manifest
func buildManifest(sourceDir string, userID int, shareName string, serverName string) (*BackupManifest, error) {
	manifest := &BackupManifest{
		Version:      1,
		LastSync:     time.Now(),
		UserID:       userID,
		ShareName:    shareName,
		SourceServer: serverName,
		Files:        make(map[string]FileMetadata),
	}

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories, hidden files, and .trash
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return nil
		}

		// Calculate checksum
		checksum, err := calculateChecksum(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		manifest.Files[relPath] = FileMetadata{
			Size:          info.Size(),
			ModTime:       info.ModTime(),
			Checksum:      checksum,
			EncryptedName: generateEncryptedName(relPath),
		}

		return nil
	})

	return manifest, err
}

// loadManifest loads manifest from USB backup directory
func loadManifest(destDir string) (*BackupManifest, error) {
	manifestPath := filepath.Join(destDir, ".anemone-manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest BackupManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// saveManifest saves manifest to USB backup directory
func saveManifest(manifest *BackupManifest, destDir string, masterKey string) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	manifestPath := filepath.Join(destDir, ".anemone-manifest.json")
	return os.WriteFile(manifestPath, data, 0600)
}

// compareManifests compares local and remote manifests
func compareManifests(local, remote *BackupManifest) (toAdd, toUpdate, toDelete []string) {
	// Files to add or update
	for relPath, localMeta := range local.Files {
		remoteMeta, exists := remote.Files[relPath]
		if !exists {
			toAdd = append(toAdd, relPath)
		} else if localMeta.Checksum != remoteMeta.Checksum {
			toUpdate = append(toUpdate, relPath)
		}
	}

	// Files to delete
	for relPath := range remote.Files {
		if _, exists := local.Files[relPath]; !exists {
			toDelete = append(toDelete, relPath)
		}
	}

	return
}

// copyFileEncrypted copies a file with encryption using streaming
func copyFileEncrypted(src, dest string, masterKey string) (int64, error) {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return 0, fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	destFile, err := os.Create(dest)
	if err != nil {
		return 0, fmt.Errorf("failed to create destination: %w", err)
	}
	defer destFile.Close()

	// Encrypt using streaming
	if err := crypto.EncryptStream(srcFile, destFile, masterKey); err != nil {
		os.Remove(dest) // Clean up on error
		return 0, fmt.Errorf("failed to encrypt: %w", err)
	}

	// Get size of encrypted file
	info, err := destFile.Stat()
	if err != nil {
		return 0, nil // File was written, just can't get size
	}

	return info.Size(), nil
}

// calculateChecksum calculates SHA256 checksum of a file
func calculateChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// generateEncryptedName generates a safe encrypted filename
func generateEncryptedName(relPath string) string {
	// Use SHA256 of path as filename to avoid path traversal issues
	hash := sha256.Sum256([]byte(relPath))
	return hex.EncodeToString(hash[:16]) + ".enc"
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// EstimateBackupSize calculates the estimated size of a backup
// Returns total bytes for config and data separately
func EstimateBackupSize(db *sql.DB, backup *USBBackup, configInfo *ConfigBackupInfo) (configBytes int64, dataBytes int64, err error) {
	// Estimate config size
	if configInfo != nil {
		// Database
		if info, err := os.Stat(configInfo.DBPath); err == nil {
			configBytes += info.Size()
		}

		// Certificates
		if configInfo.CertsDir != "" {
			filepath.Walk(configInfo.CertsDir, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() {
					configBytes += info.Size()
				}
				return nil
			})
		}

		// SMB config
		if info, err := os.Stat(configInfo.SMBConf); err == nil {
			configBytes += info.Size()
		}
	}

	// Estimate data size (only if not config-only backup)
	if backup.BackupType != BackupTypeConfig {
		allShares, err := shares.GetAll(db)
		if err != nil {
			return configBytes, 0, err
		}

		selectedIDs := backup.GetSelectedShareIDs()

		for _, share := range allShares {
			// Check if share should be backed up
			shouldBackup := false
			if len(selectedIDs) == 0 {
				shouldBackup = share.SyncEnabled
			} else {
				shouldBackup = backup.IsShareSelected(share.ID)
			}

			if !shouldBackup {
				continue
			}

			// Calculate share size
			shareSize, _ := CalculateDirSize(share.Path)
			dataBytes += shareSize
		}
	}

	return configBytes, dataBytes, nil
}

// CalculateDirSize calculates total size of files in a directory
func CalculateDirSize(path string) (int64, error) {
	var totalSize int64

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}
		if info.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip hidden files
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		totalSize += info.Size()
		return nil
	})

	return totalSize, err
}
