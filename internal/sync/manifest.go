// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package sync

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

// FileMetadata represents metadata for a single file
type FileMetadata struct {
	Size          int64     `json:"size"`
	ModTime       time.Time `json:"mtime"`
	Checksum      string    `json:"checksum"`
	EncryptedPath string    `json:"encrypted_path"`
}

// SyncManifest represents the complete manifest of synced files
type SyncManifest struct {
	Version      int                     `json:"version"`
	LastSync     time.Time               `json:"last_sync"`
	UserID       int                     `json:"user_id"`
	ShareName    string                  `json:"share_name"`
	SourceServer string                  `json:"source_server"` // Name of the server that created this backup
	Files        map[string]FileMetadata `json:"files"`
}

// SyncDelta represents changes between local and remote manifests
type SyncDelta struct {
	ToAdd    []string // Files to add (new files)
	ToUpdate []string // Files to update (modified)
	ToDelete []string // Files to delete on remote
}

// LoadLocalManifestCache loads a cached local manifest from disk
func LoadLocalManifestCache(cacheFile string) (*SyncManifest, error) {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No cache exists yet (first sync)
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}
	return UnmarshalManifest(data)
}

// SaveLocalManifestCache saves a local manifest to disk cache
func SaveLocalManifestCache(manifest *SyncManifest, cacheFile string) error {
	data, err := MarshalManifest(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(cacheFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// BuildManifest scans a directory recursively and creates a manifest
// Excludes hidden files/directories and the .trash directory
// Uses cached manifest to avoid recalculating checksums for unchanged files
func BuildManifest(sourceDir string, userID int, shareName string, sourceServer string) (*SyncManifest, error) {
	// Try to load cached manifest
	cacheFile := filepath.Join(sourceDir, ".anemone-local-manifest.json")
	cachedManifest, err := LoadLocalManifestCache(cacheFile)
	if err != nil {
		logger.Info("⚠️  Warning: failed to load manifest cache: %v (will do full scan)", err)
		cachedManifest = nil
	}

	if cachedManifest != nil {
		logger.Info("📦 Loaded cached manifest with %d files", len(cachedManifest.Files))
	}

	manifest := &SyncManifest{
		Version:      1,
		LastSync:     time.Now(),
		UserID:       userID,
		ShareName:    shareName,
		SourceServer: sourceServer,
		Files:        make(map[string]FileMetadata),
	}

	fileCount := 0
	lastLoggedCount := 0
	checksumCalculated := 0
	checksumReused := 0
	logger.Info("🔍 Building manifest for share '%s' (user %d)...", shareName, userID)

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the source directory itself
		if path == sourceDir {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Skip hidden files/directories (starting with .)
		if strings.HasPrefix(filepath.Base(path), ".") {
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

		// Use forward slashes for consistency (even on Windows)
		relPath = filepath.ToSlash(relPath)

		// Try to reuse checksum from cache if file hasn't changed
		var checksum string
		if cachedManifest != nil {
			if cachedMeta, exists := cachedManifest.Files[relPath]; exists {
				// Check if file unchanged (same size and mtime)
				if cachedMeta.Size == info.Size() && cachedMeta.ModTime.Equal(info.ModTime()) {
					// File unchanged - reuse cached checksum
					checksum = cachedMeta.Checksum
					checksumReused++
				}
			}
		}

		// If no cached checksum available, calculate it
		if checksum == "" {
			var err error
			checksum, err = CalculateChecksum(path)
			if err != nil {
				return fmt.Errorf("failed to calculate checksum for %s: %w", relPath, err)
			}
			checksumCalculated++
		}

		// Add to manifest
		manifest.Files[relPath] = FileMetadata{
			Size:          info.Size(),
			ModTime:       info.ModTime(),
			Checksum:      checksum,
			EncryptedPath: relPath + ".enc",
		}

		fileCount++

		// Log progress every 1000 files
		if fileCount-lastLoggedCount >= 1000 {
			logger.Info("   📊 Manifest progress: %d files scanned (%d checksums calculated, %d reused from cache)...",
				fileCount, checksumCalculated, checksumReused)
			lastLoggedCount = fileCount
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	logger.Info("✅ Manifest built: %d files indexed (%d checksums calculated, %d reused from cache)",
		fileCount, checksumCalculated, checksumReused)

	// Save manifest cache for next sync
	if err := SaveLocalManifestCache(manifest, cacheFile); err != nil {
		logger.Info("⚠️  Warning: failed to save manifest cache: %v", err)
		// Don't fail the sync, just log the warning
	} else {
		logger.Info("💾 Manifest cache saved to %s", cacheFile)
	}

	return manifest, nil
}

// CompareManifests compares local and remote manifests and returns delta
// If remote is nil, all local files are considered new
func CompareManifests(local, remote *SyncManifest) (*SyncDelta, error) {
	delta := &SyncDelta{
		ToAdd:    []string{},
		ToUpdate: []string{},
		ToDelete: []string{},
	}

	// If no remote manifest, everything is new
	if remote == nil || remote.Files == nil {
		for relPath := range local.Files {
			delta.ToAdd = append(delta.ToAdd, relPath)
		}
		return delta, nil
	}

	// Check local files against remote
	for relPath, localMeta := range local.Files {
		remoteMeta, exists := remote.Files[relPath]

		if !exists {
			// File doesn't exist on remote -> add
			delta.ToAdd = append(delta.ToAdd, relPath)
		} else {
			// File exists, check if modified
			if localMeta.Checksum != remoteMeta.Checksum ||
				localMeta.Size != remoteMeta.Size ||
				!localMeta.ModTime.Equal(remoteMeta.ModTime) {
				// File modified -> update
				delta.ToUpdate = append(delta.ToUpdate, relPath)
			}
			// else: file unchanged, skip
		}
	}

	// Check remote files not in local -> delete
	for relPath := range remote.Files {
		if _, exists := local.Files[relPath]; !exists {
			delta.ToDelete = append(delta.ToDelete, relPath)
		}
	}

	return delta, nil
}

// CalculateChecksum calculates SHA-256 checksum of a file
func CalculateChecksum(filePath string) (string, error) {
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

// MarshalManifest serializes a manifest to JSON
func MarshalManifest(manifest *SyncManifest) ([]byte, error) {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}
	return data, nil
}

// UnmarshalManifest deserializes a manifest from JSON
func UnmarshalManifest(data []byte) (*SyncManifest, error) {
	var manifest SyncManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}
	return &manifest, nil
}

// GetManifestStats returns statistics about the manifest
func GetManifestStats(manifest *SyncManifest) (fileCount int, totalSize int64) {
	fileCount = len(manifest.Files)
	for _, meta := range manifest.Files {
		totalSize += meta.Size
	}
	return
}

// PrintDelta prints a human-readable summary of a sync delta
func PrintDelta(delta *SyncDelta) string {
	return fmt.Sprintf("Delta: %d to add, %d to update, %d to delete",
		len(delta.ToAdd), len(delta.ToUpdate), len(delta.ToDelete))
}
