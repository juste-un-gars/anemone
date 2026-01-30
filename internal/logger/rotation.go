// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RotatingWriter handles daily log rotation with retention policies.
//
// Features:
// - Creates a new log file each day (prefix-YYYY-MM-DD.log)
// - Automatically rotates at midnight
// - Cleans up old files based on age (RetentionDays) or total size (MaxSizeMB)
type RotatingWriter struct {
	dir           string
	prefix        string
	retentionDays int
	maxSizeMB     int

	mu          sync.Mutex
	currentFile *os.File
	currentDate string
}

// NewRotatingWriter creates a new rotating log writer.
//
// Parameters:
//   - dir: directory for log files
//   - prefix: filename prefix (e.g., "anemone" -> "anemone-2026-01-30.log")
//   - retentionDays: delete logs older than this (0 = no age limit)
//   - maxSizeMB: delete oldest logs when total size exceeds this (0 = no size limit)
func NewRotatingWriter(dir, prefix string, retentionDays, maxSizeMB int) (*RotatingWriter, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	rw := &RotatingWriter{
		dir:           dir,
		prefix:        prefix,
		retentionDays: retentionDays,
		maxSizeMB:     maxSizeMB,
	}

	// Open initial log file
	if err := rw.rotate(); err != nil {
		return nil, err
	}

	// Run initial cleanup
	go rw.cleanup()

	return rw, nil
}

// Write implements io.Writer. Thread-safe.
func (rw *RotatingWriter) Write(p []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// Check if we need to rotate (new day)
	today := time.Now().Format("2006-01-02")
	if today != rw.currentDate {
		if err := rw.rotate(); err != nil {
			return 0, err
		}
	}

	return rw.currentFile.Write(p)
}

// Close closes the current log file.
func (rw *RotatingWriter) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.currentFile != nil {
		return rw.currentFile.Close()
	}
	return nil
}

// rotate switches to a new log file for today.
// Must be called with rw.mu held.
func (rw *RotatingWriter) rotate() error {
	// Close previous file if open
	if rw.currentFile != nil {
		rw.currentFile.Close()
	}

	// Create new file for today
	today := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s-%s.log", rw.prefix, today)
	path := filepath.Join(rw.dir, filename)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	rw.currentFile = f
	rw.currentDate = today

	// Schedule cleanup in background
	go rw.cleanup()

	return nil
}

// cleanup removes old log files based on retention policies.
func (rw *RotatingWriter) cleanup() {
	files, err := rw.listLogFiles()
	if err != nil {
		return
	}

	if len(files) == 0 {
		return
	}

	// Sort by date (oldest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].date.Before(files[j].date)
	})

	now := time.Now()
	var toDelete []string

	// Mark files for deletion by age
	if rw.retentionDays > 0 {
		cutoff := now.AddDate(0, 0, -rw.retentionDays)
		for _, f := range files {
			if f.date.Before(cutoff) {
				toDelete = append(toDelete, f.path)
			}
		}
	}

	// Mark files for deletion by total size
	if rw.maxSizeMB > 0 {
		var totalSize int64
		for _, f := range files {
			totalSize += f.size
		}

		maxBytes := int64(rw.maxSizeMB) * 1024 * 1024
		for i := 0; i < len(files)-1 && totalSize > maxBytes; i++ {
			// Don't delete today's file
			if files[i].date.Format("2006-01-02") == now.Format("2006-01-02") {
				continue
			}
			// Only add if not already marked for deletion
			if !contains(toDelete, files[i].path) {
				toDelete = append(toDelete, files[i].path)
			}
			totalSize -= files[i].size
		}
	}

	// Delete marked files
	for _, path := range toDelete {
		os.Remove(path)
	}
}

// logFileInfo holds information about a log file.
type logFileInfo struct {
	path string
	date time.Time
	size int64
}

// listLogFiles returns all log files matching our pattern.
func (rw *RotatingWriter) listLogFiles() ([]logFileInfo, error) {
	entries, err := os.ReadDir(rw.dir)
	if err != nil {
		return nil, err
	}

	var files []logFileInfo
	prefix := rw.prefix + "-"
	suffix := ".log"

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
			continue
		}

		// Extract date from filename
		dateStr := strings.TrimPrefix(name, prefix)
		dateStr = strings.TrimSuffix(dateStr, suffix)

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue // Skip files with invalid date format
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, logFileInfo{
			path: filepath.Join(rw.dir, name),
			date: date,
			size: info.Size(),
		})
	}

	return files, nil
}

// ListLogFiles returns information about available log files.
// This is exported for the admin UI.
func ListLogFiles(dir, prefix string) ([]LogFileEntry, error) {
	rw := &RotatingWriter{dir: dir, prefix: prefix}
	files, err := rw.listLogFiles()
	if err != nil {
		return nil, err
	}

	// Sort by date (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].date.After(files[j].date)
	})

	var entries []LogFileEntry
	for _, f := range files {
		entries = append(entries, LogFileEntry{
			Name: filepath.Base(f.path),
			Path: f.path,
			Date: f.date,
			Size: f.size,
		})
	}

	return entries, nil
}

// LogFileEntry represents a log file for the admin UI.
type LogFileEntry struct {
	Name string
	Path string
	Date time.Time
	Size int64
}

// FormatSize returns a human-readable file size.
func (e LogFileEntry) FormatSize() string {
	const (
		KB = 1024
		MB = KB * 1024
	)

	switch {
	case e.Size >= MB:
		return fmt.Sprintf("%.1f MB", float64(e.Size)/MB)
	case e.Size >= KB:
		return fmt.Sprintf("%.1f KB", float64(e.Size)/KB)
	default:
		return fmt.Sprintf("%d B", e.Size)
	}
}

// contains checks if a slice contains a string.
func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
