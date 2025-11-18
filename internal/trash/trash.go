// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package trash

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// TrashItem represents a file or directory in the trash
type TrashItem struct {
	Name         string    // Original filename
	RelativePath string    // Relative path within share
	TrashedPath  string    // Full path in .trash directory
	Size         int64     // File size in bytes (0 for directories)
	ModTime      time.Time // Last modification time
	TrashedAt    time.Time // When it was deleted
	IsDir        bool      // True if this is a directory
}

// ListTrashItems scans a share's .trash directory and returns all items (files and directories)
// Only returns top-level items to avoid showing files inside deleted directories twice
func ListTrashItems(sharePath, username string) ([]*TrashItem, error) {
	trashPath := filepath.Join(sharePath, ".trash", username)

	// Check if trash directory exists
	if _, err := os.Stat(trashPath); os.IsNotExist(err) {
		return []*TrashItem{}, nil // Empty trash
	}

	var items []*TrashItem

	// Read directory entries (only first level)
	entries, err := os.ReadDir(trashPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read trash directory: %w", err)
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip entries we can't read
		}

		path := filepath.Join(trashPath, entry.Name())

		// Calculate size (for directories, walk and sum all files)
		size := info.Size()
		if entry.IsDir() {
			size = calculateDirSize(path)
		}

		item := &TrashItem{
			Name:         entry.Name(),
			RelativePath: entry.Name(),
			TrashedPath:  path,
			Size:         size,
			ModTime:      info.ModTime(),
			TrashedAt:    info.ModTime(), // Samba recycle touches the file
			IsDir:        entry.IsDir(),
		}

		items = append(items, item)
	}

	return items, nil
}

// calculateDirSize recursively calculates the total size of a directory
func calculateDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

// RestoreItem restores a file or directory from trash to its original location
func RestoreItem(sharePath, username, relPath string) error {
	trashPath := filepath.Join(sharePath, ".trash", username, relPath)
	originalPath := filepath.Join(sharePath, relPath)

	// Check if item exists in trash
	if _, err := os.Stat(trashPath); os.IsNotExist(err) {
		return fmt.Errorf("item not found in trash")
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

	// Move file from trash to original location (using sudo for permission)
	cmd := exec.Command("sudo", "mv", trashPath, originalPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	// Fix permissions after restore to ensure files are readable
	// u+rwX = user can read/write/execute(dirs only)
	// go+rX = group and others can read/execute(dirs only)
	// This gives 755 for directories and 644 for files
	cmdChmod := exec.Command("sudo", "chmod", "-R", "u+rwX,go+rX", originalPath)
	if err := cmdChmod.Run(); err != nil {
		return fmt.Errorf("failed to fix permissions: %w", err)
	}

	// Clean up empty directories in trash
	cleanupEmptyDirs(filepath.Dir(trashPath), filepath.Join(sharePath, ".trash", username))

	return nil
}

// DeleteItem permanently deletes a file or directory from trash
func DeleteItem(sharePath, username, relPath string) error {
	trashPath := filepath.Join(sharePath, ".trash", username, relPath)

	// Check if item exists
	if _, err := os.Stat(trashPath); os.IsNotExist(err) {
		return fmt.Errorf("item not found in trash")
	}

	// Delete the item (using sudo for permission, -rf to handle directories)
	cmd := exec.Command("sudo", "rm", "-rf", trashPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
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

	// Remove the entire trash directory (using sudo for permission)
	cmd := exec.Command("sudo", "rm", "-rf", trashPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to empty trash: %w", err)
	}

	// Recreate the trash directory for future use
	if err := os.MkdirAll(trashPath, 0755); err != nil {
		return fmt.Errorf("failed to recreate trash directory: %w", err)
	}

	return nil
}

// cleanupEmptyDirs removes empty directories up to but not including the base directory
func cleanupEmptyDirs(dir, base string) {
	for dir != base && dir != "." && dir != "/" {
		// Try to remove the directory (will fail if not empty) - using sudo for permission
		cmd := exec.Command("sudo", "rmdir", dir)
		if err := cmd.Run(); err != nil {
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
