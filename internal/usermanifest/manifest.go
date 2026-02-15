// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package usermanifest provides functionality to generate and maintain
// manifest files in user shares for efficient synchronization with AnemoneSync.
//
// The manifest files are stored in .anemone/manifest.json within each user share
// and contain metadata about all files in the share, allowing sync clients to
// quickly detect changes without scanning the entire share via SMB.
package usermanifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"github.com/juste-un-gars/anemone/internal/logger"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UserManifest represents a manifest for a user share, compatible with AnemoneSync.
// This manifest is stored as plain JSON (not encrypted) in .anemone/manifest.json
// within each user share directory.
type UserManifest struct {
	// Version of the manifest format (currently 1)
	Version int `json:"version"`

	// GeneratedAt is the UTC timestamp when this manifest was generated
	GeneratedAt time.Time `json:"generated_at"`

	// ShareName is the name of the share (e.g., "data_alice", "backup_alice")
	ShareName string `json:"share_name"`

	// ShareType is either "data" or "backup"
	ShareType string `json:"share_type"`

	// Username is the owner of the share
	Username string `json:"username"`

	// FileCount is the total number of files in the manifest
	FileCount int `json:"file_count"`

	// TotalSize is the total size of all files in bytes
	TotalSize int64 `json:"total_size"`

	// Files contains metadata for each file in the share
	Files []UserFileEntry `json:"files"`
}

// UserFileEntry represents metadata for a single file in the manifest.
// The format is designed to be compatible with AnemoneSync requirements.
type UserFileEntry struct {
	// Path is the relative path from the share root, using forward slashes
	Path string `json:"path"`

	// Size is the file size in bytes
	Size int64 `json:"size"`

	// Mtime is the Unix timestamp of the last modification time
	Mtime int64 `json:"mtime"`

	// Hash is the file content hash in format "sha256:hexdigest"
	Hash string `json:"hash"`
}

// ManifestDir is the directory name where manifests are stored within shares
const ManifestDir = ".anemone"

// ManifestFileName is the name of the manifest file
const ManifestFileName = "manifest.json"

// BuildUserManifest scans a share directory and creates a manifest.
// It reuses checksums from an existing manifest when file size and mtime match.
//
// Parameters:
//   - sharePath: absolute path to the share directory
//   - shareName: name of the share (e.g., "data_alice")
//   - shareType: either "data" or "backup"
//   - username: the owner of the share
//
// Returns the built manifest or an error if the scan fails.
func BuildUserManifest(sharePath, shareName, shareType, username string) (*UserManifest, error) {
	manifest := &UserManifest{
		Version:     1,
		GeneratedAt: time.Now().UTC(),
		ShareName:   shareName,
		ShareType:   shareType,
		Username:    username,
		Files:       []UserFileEntry{},
	}

	// Try to load existing manifest for checksum reuse
	existingManifest := loadCachedManifest(sharePath)
	existingFiles := make(map[string]UserFileEntry)
	if existingManifest != nil {
		for _, f := range existingManifest.Files {
			existingFiles[f.Path] = f
		}
	}

	checksumCalculated := 0
	checksumReused := 0

	err := filepath.Walk(sharePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log but continue on permission errors
			if os.IsPermission(err) {
				logger.Info("Permission denied", "path", path)
				return nil
			}
			return err
		}

		// Skip the share root itself
		if path == sharePath {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(sharePath, path)
		if err != nil {
			return err
		}

		// Skip hidden files and directories (starting with .)
		baseName := filepath.Base(path)
		if strings.HasPrefix(baseName, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories (we only track files)
		if info.IsDir() {
			return nil
		}

		// Skip non-regular files (symlinks, pipes, etc.)
		if !info.Mode().IsRegular() {
			return nil
		}

		// Use forward slashes for consistency (cross-platform)
		relPath = filepath.ToSlash(relPath)

		// Try to reuse checksum from existing manifest
		var hash string
		if existing, ok := existingFiles[relPath]; ok {
			// Check if file hasn't changed (same size and mtime)
			if existing.Size == info.Size() && existing.Mtime == info.ModTime().Unix() {
				hash = existing.Hash
				checksumReused++
			}
		}

		// Calculate checksum if not reused
		if hash == "" {
			var calcErr error
			hash, calcErr = calculateChecksum(path)
			if calcErr != nil {
				logger.Info("Failed to calculate checksum for", "rel_path", relPath, "calc_err", calcErr)
				return nil // Skip this file but continue
			}
			checksumCalculated++
		}

		manifest.Files = append(manifest.Files, UserFileEntry{
			Path:  relPath,
			Size:  info.Size(),
			Mtime: info.ModTime().Unix(),
			Hash:  hash,
		})

		manifest.FileCount++
		manifest.TotalSize += info.Size()

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	if checksumCalculated > 0 || checksumReused > 0 {
		logger.Info("files ( checksums calculated, reused from cache)", "share_name", shareName, "file_count", manifest.FileCount, "checksum_calculated", checksumCalculated, "checksum_reused", checksumReused)
	}

	return manifest, nil
}

// WriteManifest writes a manifest to the share directory.
// It uses atomic write (temp file + rename) to prevent partial reads.
//
// The manifest is written to: {sharePath}/.anemone/manifest.json
func WriteManifest(manifest *UserManifest, sharePath string) error {
	anemoneDir := filepath.Join(sharePath, ManifestDir)

	// Create .anemone directory if it doesn't exist
	if err := os.MkdirAll(anemoneDir, 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}

	// Marshal manifest to JSON with indentation for readability
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Write to temp file first (atomic write pattern)
	tempFile := filepath.Join(anemoneDir, ManifestFileName+".tmp")
	finalFile := filepath.Join(anemoneDir, ManifestFileName)

	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp manifest: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, finalFile); err != nil {
		// Clean up temp file on error
		os.Remove(tempFile)
		return fmt.Errorf("failed to finalize manifest: %w", err)
	}

	return nil
}

// loadCachedManifest loads an existing manifest from a share directory.
// Returns nil if the manifest doesn't exist or can't be parsed.
func loadCachedManifest(sharePath string) *UserManifest {
	manifestPath := filepath.Join(sharePath, ManifestDir, ManifestFileName)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil // No existing manifest
	}

	var manifest UserManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		logger.Info("Failed to parse existing manifest", "error", err)
		return nil
	}

	return &manifest
}

// calculateChecksum calculates the SHA-256 checksum of a file.
// Returns the checksum in format "sha256:hexdigest".
func calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	checksum := hex.EncodeToString(hash.Sum(nil))
	return "sha256:" + checksum, nil
}

// FormatSize formats a byte size into a human-readable string.
func FormatSize(bytes int64) string {
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
