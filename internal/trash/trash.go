// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package trash manages deleted files with configurable retention and recovery support.
package trash

import (
	"database/sql"
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

// ListTrashItems scans a share's .trash directory and returns all deleted files
// With keeptree=yes, Samba preserves directory structure, so we walk recursively
// to find the actual deleted files and return them with their relative paths
func ListTrashItems(sharePath, username string) ([]*TrashItem, error) {
	trashPath := filepath.Join(sharePath, ".trash", username)

	// Check if trash directory exists
	if _, err := os.Stat(trashPath); os.IsNotExist(err) {
		return []*TrashItem{}, nil // Empty trash
	}

	var items []*TrashItem

	// Walk recursively to find all files (not directories)
	err := filepath.Walk(trashPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Skip the root trash directory itself
		if path == trashPath {
			return nil
		}

		// Skip directories - we only want files
		// The directories are just structure from keeptree=yes
		if info.IsDir() {
			return nil
		}

		// Calculate relative path from trash root
		relPath, err := filepath.Rel(trashPath, path)
		if err != nil {
			return nil // Skip if we can't get relative path
		}

		item := &TrashItem{
			Name:         info.Name(),
			RelativePath: relPath,
			TrashedPath:  path,
			Size:         info.Size(),
			ModTime:      info.ModTime(),
			TrashedAt:    info.ModTime(), // Samba recycle touches the file
			IsDir:        false,
		}

		items = append(items, item)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan trash directory: %w", err)
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

// CleanupAllUserTrash runs trash cleanup for all users and their shares
// This should be called periodically (e.g., daily) to enforce retention policy
func CleanupAllUserTrash(db interface{}, retentionDays int) (int, error) {
	if retentionDays == 0 {
		// Retention disabled (infinite retention)
		return 0, nil
	}

	// Type assert to *sql.DB
	database, ok := db.(interface {
		Query(query string, args ...interface{}) (*sql.Rows, error)
	})
	if !ok {
		return 0, fmt.Errorf("invalid database type")
	}

	// Get all users and their shares
	rows, err := database.Query("SELECT u.username, s.path FROM users u JOIN shares s ON u.id = s.user_id")
	if err != nil {
		return 0, fmt.Errorf("failed to query users and shares: %w", err)
	}
	defer rows.Close()

	totalDeleted := 0

	for rows.Next() {
		var username, sharePath string
		if err := rows.Scan(&username, &sharePath); err != nil {
			fmt.Printf("Warning: failed to scan user/share: %v\n", err)
			continue
		}

		// Cleanup trash for this user's share
		deleted, err := CleanupOldTrashItems(sharePath, username, retentionDays)
		if err != nil {
			fmt.Printf("Warning: failed to cleanup trash for %s in %s: %v\n", username, sharePath, err)
			continue
		}

		if deleted > 0 {
			fmt.Printf("Cleaned up %d old item(s) from %s's trash in share %s\n", deleted, username, sharePath)
			totalDeleted += deleted
		}
	}

	if totalDeleted > 0 {
		fmt.Printf("Total: cleaned up %d old item(s) from trash across all users\n", totalDeleted)
	}

	return totalDeleted, nil
}

// CleanupOldTrashItems deletes trash items older than the specified number of days
// Returns number of items deleted and any error encountered
func CleanupOldTrashItems(sharePath, username string, retentionDays int) (int, error) {
	trashPath := filepath.Join(sharePath, ".trash", username)

	// Check if trash directory exists
	if _, err := os.Stat(trashPath); os.IsNotExist(err) {
		return 0, nil // No trash directory, nothing to clean
	}

	now := time.Now()
	cutoffTime := now.AddDate(0, 0, -retentionDays)
	deletedCount := 0

	// Read directory entries (only first level)
	entries, err := os.ReadDir(trashPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read trash directory: %w", err)
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip entries we can't read
		}

		// Check if item is older than retention period
		if info.ModTime().Before(cutoffTime) {
			itemPath := filepath.Join(trashPath, entry.Name())

			// Delete the item (using sudo for permission, -rf to handle directories)
			cmd := exec.Command("sudo", "rm", "-rf", itemPath)
			if err := cmd.Run(); err != nil {
				// Log error but continue with other items
				fmt.Printf("Warning: failed to delete old trash item %s: %v\n", entry.Name(), err)
				continue
			}

			deletedCount++
		}
	}

	// Clean up empty directories
	if deletedCount > 0 {
		cleanupEmptyDirs(trashPath, filepath.Join(sharePath, ".trash"))
	}

	return deletedCount, nil
}
