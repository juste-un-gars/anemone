// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package trash

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// TrashItem represents a file in the trash
type TrashItem struct {
	Name         string    // Original filename
	RelativePath string    // Relative path within share
	TrashedPath  string    // Full path in .trash directory
	Size         int64     // File size in bytes
	ModTime      time.Time // Last modification time
	TrashedAt    time.Time // When it was deleted
}

// ListTrashItems scans a share's .trash directory and returns all items
func ListTrashItems(sharePath, username string) ([]*TrashItem, error) {
	trashPath := filepath.Join(sharePath, ".trash", username)

	// Check if trash directory exists
	if _, err := os.Stat(trashPath); os.IsNotExist(err) {
		return []*TrashItem{}, nil // Empty trash
	}

	var items []*TrashItem

	err := filepath.Walk(trashPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories (we only list files)
		if info.IsDir() {
			return nil
		}

		// Get relative path from trash root
		relPath, err := filepath.Rel(trashPath, path)
		if err != nil {
			return err
		}

		item := &TrashItem{
			Name:         info.Name(),
			RelativePath: relPath,
			TrashedPath:  path,
			Size:         info.Size(),
			ModTime:      info.ModTime(),
			TrashedAt:    info.ModTime(), // Samba recycle touches the file
		}

		items = append(items, item)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan trash: %w", err)
	}

	return items, nil
}

// RestoreItem restores a file from trash to its original location
func RestoreItem(sharePath, username, relPath string) error {
	trashPath := filepath.Join(sharePath, ".trash", username, relPath)
	originalPath := filepath.Join(sharePath, relPath)

	// Check if file exists in trash
	if _, err := os.Stat(trashPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found in trash")
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(originalPath)
	if err := os.MkdirAll(parentDir, 0775); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Check if a file already exists at destination
	if _, err := os.Stat(originalPath); err == nil {
		// File exists, rename with timestamp
		timestamp := time.Now().Format("20060102-150405")
		base := filepath.Base(originalPath)
		ext := filepath.Ext(base)
		nameWithoutExt := base[:len(base)-len(ext)]
		originalPath = filepath.Join(parentDir, fmt.Sprintf("%s.restored-%s%s", nameWithoutExt, timestamp, ext))
	}

	// Move file from trash to original location
	if err := os.Rename(trashPath, originalPath); err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	// Clean up empty directories in trash
	cleanupEmptyDirs(filepath.Dir(trashPath), filepath.Join(sharePath, ".trash", username))

	return nil
}

// DeleteItem permanently deletes a file from trash
func DeleteItem(sharePath, username, relPath string) error {
	trashPath := filepath.Join(sharePath, ".trash", username, relPath)

	// Check if file exists
	if _, err := os.Stat(trashPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found in trash")
	}

	// Delete the file
	if err := os.Remove(trashPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Clean up empty directories
	cleanupEmptyDirs(filepath.Dir(trashPath), filepath.Join(sharePath, ".trash", username))

	return nil
}

// EmptyTrash deletes all files in the trash for a user
func EmptyTrash(sharePath, username string) error {
	trashPath := filepath.Join(sharePath, ".trash", username)

	// Check if trash exists
	if _, err := os.Stat(trashPath); os.IsNotExist(err) {
		return nil // Already empty
	}

	// Remove the entire trash directory
	if err := os.RemoveAll(trashPath); err != nil {
		return fmt.Errorf("failed to empty trash: %w", err)
	}

	return nil
}

// cleanupEmptyDirs removes empty directories up to but not including the base directory
func cleanupEmptyDirs(dir, base string) {
	for dir != base && dir != "." && dir != "/" {
		// Try to remove the directory (will fail if not empty)
		if err := os.Remove(dir); err != nil {
			break // Directory not empty or error, stop
		}
		dir = filepath.Dir(dir)
	}
}

// GetTrashSize returns the total size of trash for a user
func GetTrashSize(sharePath, username string) (int64, error) {
	trashPath := filepath.Join(sharePath, ".trash", username)

	var totalSize int64

	err := filepath.Walk(trashPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return 0, fmt.Errorf("failed to calculate trash size: %w", err)
	}

	return totalSize, nil
}

// CopyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Preserve permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}
