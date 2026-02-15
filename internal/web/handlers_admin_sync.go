// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"database/sql"
	"fmt"
	"github.com/juste-un-gars/anemone/internal/logger"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/incoming"
	"github.com/juste-un-gars/anemone/internal/sync"
	"github.com/juste-un-gars/anemone/internal/syncconfig"
)

func (s *Server) handleAdminSync(w http.ResponseWriter, r *http.Request) {
	// Redirect GET to consolidated backups page
	if r.Method == http.MethodGet {
		http.Redirect(w, r, "/admin/backups", http.StatusSeeOther)
		return
	}

	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Get sync configuration
	config, err := syncconfig.Get(s.db)
	if err != nil {
		logger.Info("Error getting sync config", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get recent syncs (last 20)
	type RecentSync struct {
		Username    string
		PeerName    string
		StartedAt   time.Time
		CompletedAt *time.Time
		Status      string
		FilesSynced int
		BytesSynced int64
		Speed       string // Calculated transfer speed (e.g., "25.3 MB/s")
	}

	query := `
		SELECT u.username, p.name, sl.started_at, sl.completed_at, sl.status, sl.files_synced, sl.bytes_synced
		FROM sync_log sl
		JOIN users u ON sl.user_id = u.id
		JOIN peers p ON sl.peer_id = p.id
		ORDER BY sl.started_at DESC
		LIMIT 20
	`

	rows, err := s.db.Query(query)
	if err != nil {
		logger.Info("Error getting recent syncs", "error", err)
		// Continue with empty list
	}

	var recentSyncs []RecentSync
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var rs RecentSync
			var startedAtStr, completedAtStr sql.NullString
			if err := rows.Scan(&rs.Username, &rs.PeerName, &startedAtStr, &completedAtStr, &rs.Status, &rs.FilesSynced, &rs.BytesSynced); err != nil {
				logger.Info("Error scanning sync log", "error", err)
				continue
			}
			// Parse SQLite datetime strings (try multiple formats)
			if startedAtStr.Valid {
				rs.StartedAt = parseSQLiteDateTimeSync(startedAtStr.String)
			}
			if completedAtStr.Valid {
				t := parseSQLiteDateTimeSync(completedAtStr.String)
				if !t.IsZero() {
					rs.CompletedAt = &t
				}
			}
			// Calculate transfer speed if sync completed and has data
			if rs.CompletedAt != nil && rs.BytesSynced > 0 {
				duration := rs.CompletedAt.Sub(rs.StartedAt)
				if duration.Seconds() > 0 {
					speedBps := float64(rs.BytesSynced) / duration.Seconds()
					if speedBps >= 1024*1024 {
						rs.Speed = fmt.Sprintf("%.1f MB/s", speedBps/1024/1024)
					} else if speedBps >= 1024 {
						rs.Speed = fmt.Sprintf("%.1f KB/s", speedBps/1024)
					} else {
						rs.Speed = fmt.Sprintf("%.0f B/s", speedBps)
					}
				}
			}
			recentSyncs = append(recentSyncs, rs)
		}
	}

	// Get success/error messages from query params
	successMsg := r.URL.Query().Get("success")
	errorMsg := r.URL.Query().Get("error")

	data := struct {
		Lang        string
		Title       string
		Session     *auth.Session
		Config      *syncconfig.SyncConfig
		RecentSyncs []RecentSync
		Success     string
		Error       string
	}{
		Lang:        lang,
		Title:       "Synchronisation Automatique",
		Session:     session,
		Config:      config,
		RecentSyncs: recentSyncs,
		Success:     successMsg,
		Error:       errorMsg,
	}

	if err := s.templates.ExecuteTemplate(w, "admin_sync.html", data); err != nil {
		logger.Info("Error rendering admin_sync template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleAdminSyncConfig saves the sync configuration
func (s *Server) handleAdminSyncConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/sync?error=Invalid+form+data", http.StatusSeeOther)
		return
	}

	enabled := r.FormValue("enabled") == "on"
	interval := r.FormValue("interval")
	fixedHourStr := r.FormValue("fixed_hour")

	// Validate interval
	validIntervals := map[string]bool{
		"30min": true,
		"1h":    true,
		"2h":    true,
		"6h":    true,
		"fixed": true,
	}
	if !validIntervals[interval] {
		http.Redirect(w, r, "/admin/sync?error=Invalid+interval", http.StatusSeeOther)
		return
	}

	// Parse fixed_hour if interval is "fixed"
	fixedHour := 23
	if interval == "fixed" {
		var err error
		fixedHour, err = strconv.Atoi(fixedHourStr)
		if err != nil || fixedHour < 0 || fixedHour > 23 {
			http.Redirect(w, r, "/admin/sync?error=Invalid+fixed+hour+(must+be+0-23)", http.StatusSeeOther)
			return
		}
	}

	// Update configuration
	config := &syncconfig.SyncConfig{
		Enabled:   enabled,
		Interval:  interval,
		FixedHour: fixedHour,
	}

	if err := syncconfig.Update(s.db, config); err != nil {
		logger.Info("Error updating sync config", "error", err)
		http.Redirect(w, r, "/admin/sync?error=Failed+to+update+configuration", http.StatusSeeOther)
		return
	}

	logger.Info("Admin updated sync config: enabled=, interval=, fixed_hour", "username", session.Username, "enabled", enabled, "interval", interval, "fixed_hour", fixedHour)

	http.Redirect(w, r, "/admin/sync?success=Configuration+enregistrée+avec+succès", http.StatusSeeOther)
}

// handleAdminSyncForce forces immediate synchronization of all users
func (s *Server) handleAdminSyncForce(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	logger.Info("Admin triggered forced synchronization of all users", "username", session.Username)

	// Run SyncAllUsers
	successCount, errorCount, lastError := sync.SyncAllUsers(s.db)

	// Update last_sync timestamp
	if err := syncconfig.UpdateLastSync(s.db); err != nil {
		logger.Info("Warning: Failed to update last_sync", "error", err)
	}

	// Redirect with result message
	if errorCount > 0 {
		errorMsg := fmt.Sprintf("Synchronisation partielle : %d réussis, %d échecs. Dernière erreur: %s",
			successCount, errorCount, lastError)
		http.Redirect(w, r, "/admin/peers?error="+errorMsg, http.StatusSeeOther)
	} else if successCount == 0 {
		http.Redirect(w, r, "/admin/peers?error=Aucune+synchronisation+effectuée+(pas+de+partages+activés+ou+pas+de+pairs)", http.StatusSeeOther)
	} else {
		successMsg := fmt.Sprintf("Synchronisation réussie : %d synchronisations effectuées", successCount)
		http.Redirect(w, r, "/admin/peers?success="+successMsg, http.StatusSeeOther)
	}
}

// handleAdminIncoming displays incoming backups from remote peers
func (s *Server) handleAdminIncoming(w http.ResponseWriter, r *http.Request) {
	// Redirect GET to consolidated backups page
	if r.Method == http.MethodGet {
		http.Redirect(w, r, "/admin/backups", http.StatusSeeOther)
		return
	}

	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Scan incoming backups directory
	backups, err := incoming.ScanIncomingBackups(s.db, s.cfg.IncomingDir)
	if err != nil {
		logger.Info("Error scanning incoming backups", "error", err)
		http.Error(w, "Failed to scan incoming backups", http.StatusInternalServerError)
		return
	}

	// Calculate total statistics
	var totalFiles int
	var totalSize int64
	for _, backup := range backups {
		totalFiles += backup.FileCount
		totalSize += backup.TotalSize
	}

	data := struct {
		Lang       string
		Session    *auth.Session
		Backups    []*incoming.IncomingBackup
		TotalFiles int
		TotalSize  string
		Error      string
		Success    string
	}{
		Lang:       s.cfg.Language,
		Session:    session,
		Backups:    backups,
		TotalFiles: totalFiles,
		TotalSize:  incoming.FormatBytes(totalSize),
		Error:      r.URL.Query().Get("error"),
		Success:    r.URL.Query().Get("success"),
	}

	if err := s.templates.ExecuteTemplate(w, "admin_incoming.html", data); err != nil {
		logger.Info("Template error", "error", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

// handleAdminIncomingDelete deletes an incoming backup
func (s *Server) handleAdminIncomingDelete(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/incoming?error=Invalid+form+data", http.StatusSeeOther)
		return
	}

	// Get backup path from form
	backupPath := r.FormValue("path")
	if backupPath == "" {
		http.Redirect(w, r, "/admin/incoming?error=Missing+backup+path", http.StatusSeeOther)
		return
	}

	// Security check: ensure path is within incoming directory
	incomingDir := s.cfg.IncomingDir
	absIncomingDir, err := filepath.Abs(incomingDir)
	if err != nil {
		http.Redirect(w, r, "/admin/incoming?error=Invalid+incoming+directory", http.StatusSeeOther)
		return
	}
	absBackupPath, err := filepath.Abs(backupPath)
	if err != nil {
		http.Redirect(w, r, "/admin/incoming?error=Invalid+backup+path", http.StatusSeeOther)
		return
	}
	// Use filepath.Rel to properly check if path is within directory
	relPath, err := filepath.Rel(absIncomingDir, absBackupPath)
	if err != nil || strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
		logger.Info("Security: Attempted to delete path outside incoming directory", "backup_path", backupPath)
		http.Redirect(w, r, "/admin/incoming?error=Invalid+backup+path", http.StatusSeeOther)
		return
	}

	// Delete the backup
	if err := incoming.DeleteIncomingBackup(backupPath, s.cfg.IncomingDir); err != nil {
		logger.Info("Error deleting backup", "backup_path", backupPath, "error", err)
		http.Redirect(w, r, "/admin/incoming?error=Failed+to+delete+backup", http.StatusSeeOther)
		return
	}

	logger.Info("Admin deleted incoming backup", "username", session.Username, "backup_path", backupPath)
	http.Redirect(w, r, "/admin/incoming?success=Backup+deleted+successfully", http.StatusSeeOther)
}

// parseSQLiteDateTimeSync parses a datetime string from SQLite, trying multiple formats.
// SQLite can return datetimes in various formats depending on how they were stored.
func parseSQLiteDateTimeSync(s string) time.Time {
	// List of formats to try (most common first)
	formats := []string{
		"2006-01-02 15:04:05",           // Standard SQLite format
		"2006-01-02T15:04:05",           // ISO 8601 with T
		"2006-01-02 15:04:05.000",       // With milliseconds
		"2006-01-02T15:04:05.000",       // ISO with T and milliseconds
		"2006-01-02 15:04:05Z",          // With Z timezone
		"2006-01-02T15:04:05Z",          // ISO with T and Z
		"2006-01-02 15:04:05-07:00",     // With timezone offset
		"2006-01-02T15:04:05-07:00",     // ISO with timezone offset
		time.RFC3339,                    // Full RFC3339
		time.RFC3339Nano,                // RFC3339 with nanoseconds
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}

	// Log error if no format worked
	logger.Info("Warning: Could not parse datetime string", "s", s)
	return time.Time{}
}
