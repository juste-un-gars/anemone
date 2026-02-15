// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package usermanifest

import (
	"database/sql"
	"github.com/juste-un-gars/anemone/internal/logger"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/juste-un-gars/anemone/internal/shares"
)

// Watcher monitors share directories for changes and regenerates manifests.
type Watcher struct {
	db       *sql.DB
	watcher  *fsnotify.Watcher
	mu       sync.Mutex
	debounce map[string]*time.Timer // sharePath -> debounce timer
	stopCh   chan struct{}
}

// debounceDelay is the time to wait after a change before regenerating the manifest.
// This prevents regenerating multiple times during bulk file operations.
const debounceDelay = 3 * time.Second

// NewWatcher creates a new file system watcher for manifest updates.
func NewWatcher(db *sql.DB) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		db:       db,
		watcher:  fsWatcher,
		debounce: make(map[string]*time.Timer),
		stopCh:   make(chan struct{}),
	}, nil
}

// Start begins watching all share directories.
func (w *Watcher) Start() error {
	// Get all shares from database
	allShares, err := shares.GetAll(w.db)
	if err != nil {
		return err
	}

	watchCount := 0
	for _, share := range allShares {
		count, err := w.addWatchRecursive(share.Path)
		if err != nil {
			logger.Info("Failed to watch", "path", share.Path, "error", err)
			continue
		}
		watchCount += count
	}

	logger.Info("Manifest watcher started: directories monitored", "watch_count", watchCount)

	// Start event processing goroutine
	go w.processEvents()

	return nil
}

// Stop stops the watcher and releases resources.
func (w *Watcher) Stop() error {
	close(w.stopCh)

	w.mu.Lock()
	for _, timer := range w.debounce {
		timer.Stop()
	}
	w.mu.Unlock()

	return w.watcher.Close()
}

// addWatchRecursive adds watches to a directory and all its subdirectories.
// Returns the number of watches added.
func (w *Watcher) addWatchRecursive(root string) (int, error) {
	count := 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil // Skip permission errors
			}
			return err
		}

		if !info.IsDir() {
			return nil
		}

		// Skip hidden directories (like .anemone, .trash)
		baseName := filepath.Base(path)
		if strings.HasPrefix(baseName, ".") && path != root {
			return filepath.SkipDir
		}

		if err := w.watcher.Add(path); err != nil {
			logger.Info("Cannot watch", "path", path, "error", err)
			return nil // Continue with other directories
		}

		count++
		return nil
	})

	return count, err
}

// processEvents handles file system events.
func (w *Watcher) processEvents() {
	for {
		select {
		case <-w.stopCh:
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			logger.Info("Watcher error", "error", err)
		}
	}
}

// handleEvent processes a single file system event.
func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Skip events on hidden files/directories
	baseName := filepath.Base(event.Name)
	if strings.HasPrefix(baseName, ".") {
		return
	}

	// Find which share this event belongs to
	sharePath := w.findSharePath(event.Name)
	if sharePath == "" {
		return
	}

	// If a new directory was created, add a watch on it
	if event.Op&fsnotify.Create != 0 {
		info, err := os.Stat(event.Name)
		if err == nil && info.IsDir() {
			if err := w.watcher.Add(event.Name); err == nil {
				logger.Info("Added watch", "name", event.Name)
			}
		}
	}

	// Debounce manifest regeneration
	w.scheduleRegeneration(sharePath)
}

// findSharePath finds the root share path for a given file path.
func (w *Watcher) findSharePath(filePath string) string {
	allShares, err := shares.GetAll(w.db)
	if err != nil {
		return ""
	}

	for _, share := range allShares {
		// Check if filePath is under this share
		if strings.HasPrefix(filePath, share.Path+string(os.PathSeparator)) || filePath == share.Path {
			return share.Path
		}
	}

	return ""
}

// scheduleRegeneration schedules a manifest regeneration with debouncing.
func (w *Watcher) scheduleRegeneration(sharePath string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Cancel existing timer if any
	if timer, exists := w.debounce[sharePath]; exists {
		timer.Stop()
	}

	// Schedule new regeneration
	w.debounce[sharePath] = time.AfterFunc(debounceDelay, func() {
		w.regenerateManifest(sharePath)
	})
}

// regenerateManifest regenerates the manifest for a specific share.
func (w *Watcher) regenerateManifest(sharePath string) {
	// Find share info from database
	allShares, err := shares.GetAll(w.db)
	if err != nil {
		logger.Info("Failed to get shares", "error", err)
		return
	}

	var targetShare *shares.Share
	for _, share := range allShares {
		if share.Path == sharePath {
			targetShare = share
			break
		}
	}

	if targetShare == nil {
		logger.Info("Share not found for path", "share_path", sharePath)
		return
	}

	shareType := determineShareType(targetShare.Name)
	username, err := getUsername(w.db, targetShare.UserID)
	if err != nil {
		logger.Info("Failed to get username", "error", err)
		return
	}

	startTime := time.Now()

	manifest, err := BuildUserManifest(sharePath, targetShare.Name, shareType, username)
	if err != nil {
		logger.Info("Failed to build manifest for", "name", targetShare.Name, "error", err)
		return
	}

	if err := WriteManifest(manifest, sharePath); err != nil {
		logger.Info("Failed to write manifest for", "name", targetShare.Name, "error", err)
		return
	}

	elapsed := time.Since(startTime)
	logger.Info("Manifest updated: ( files, ) in", "name", targetShare.Name, "file_count", manifest.FileCount, "total_size", FormatSize(manifest.TotalSize), "round", elapsed.Round(time.Millisecond))
}

// AddShareWatch adds watches for a newly created share.
func (w *Watcher) AddShareWatch(sharePath string) error {
	count, err := w.addWatchRecursive(sharePath)
	if err != nil {
		return err
	}
	logger.Info("Added watches for new share", "count", count, "share_path", sharePath)
	return nil
}

// RemoveShareWatch removes watches for a deleted share.
func (w *Watcher) RemoveShareWatch(sharePath string) error {
	// fsnotify automatically removes watches when directories are deleted
	// but we clean up our debounce map
	w.mu.Lock()
	if timer, exists := w.debounce[sharePath]; exists {
		timer.Stop()
		delete(w.debounce, sharePath)
	}
	w.mu.Unlock()

	logger.Info("Removed watches for share", "share_path", sharePath)
	return nil
}
