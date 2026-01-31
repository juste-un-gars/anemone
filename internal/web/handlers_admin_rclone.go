// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/incoming"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/rclone"
)

// handleAdminRclone displays the rclone backup management page
func (s *Server) handleAdminRclone(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	lang := s.getLang(r)

	// Get all rclone backup configurations
	backups, err := rclone.GetAll(s.db)
	if err != nil {
		logger.Info("Error getting rclone backups: %v", err)
		backups = []*rclone.RcloneBackup{}
	}

	// Check if rclone is installed
	rcloneInstalled := rclone.IsRcloneInstalled()
	var rcloneVersion string
	if rcloneInstalled {
		rcloneVersion, _ = rclone.GetRcloneVersion()
	}

	// Build backup status info
	type BackupWithStatus struct {
		*rclone.RcloneBackup
		LastSyncAgo string
		StatusClass string
	}

	var backupsWithStatus []BackupWithStatus
	for _, b := range backups {
		bws := BackupWithStatus{
			RcloneBackup: b,
		}

		// Format last sync time
		if b.LastSync != nil {
			bws.LastSyncAgo = incoming.FormatTimeAgo(*b.LastSync, lang)
		}

		// Status class for styling
		switch b.LastStatus {
		case "success":
			bws.StatusClass = "text-green-600"
		case "error":
			bws.StatusClass = "text-red-600"
		case "running":
			bws.StatusClass = "text-blue-600"
		default:
			bws.StatusClass = "text-gray-500"
		}

		backupsWithStatus = append(backupsWithStatus, bws)
	}

	data := map[string]interface{}{
		"Session":          session,
		"Title":            i18n.T(lang, "rclone.title"),
		"Lang":             lang,
		"Backups":          backupsWithStatus,
		"RcloneInstalled":  rcloneInstalled,
		"RcloneVersion":    rcloneVersion,
		"FormatBytes":      rclone.FormatBytes,
		"Success":          r.URL.Query().Get("success") != "",
		"Syncing":          r.URL.Query().Get("syncing") != "",
		"TestSuccess":      r.URL.Query().Get("test_success") != "",
		"TestError":        r.URL.Query().Get("test_error"),
		"Error":            r.URL.Query().Get("error"),
	}

	if err := s.templates.ExecuteTemplate(w, "admin_rclone.html", data); err != nil {
		logger.Info("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleAdminRcloneAdd handles adding a new rclone backup configuration
func (s *Server) handleAdminRcloneAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := s.getLang(r)

	name := strings.TrimSpace(r.FormValue("name"))
	sftpHost := strings.TrimSpace(r.FormValue("sftp_host"))
	sftpPortStr := strings.TrimSpace(r.FormValue("sftp_port"))
	sftpUser := strings.TrimSpace(r.FormValue("sftp_user"))
	sftpKeyPath := strings.TrimSpace(r.FormValue("sftp_key_path"))
	remotePath := strings.TrimSpace(r.FormValue("remote_path"))
	enabled := r.FormValue("enabled") == "on"

	if name == "" || sftpHost == "" || sftpUser == "" || remotePath == "" {
		http.Redirect(w, r, "/admin/rclone?error="+i18n.T(lang, "missing_fields"), http.StatusSeeOther)
		return
	}

	sftpPort := 22
	if sftpPortStr != "" {
		if p, err := strconv.Atoi(sftpPortStr); err == nil && p > 0 && p < 65536 {
			sftpPort = p
		}
	}

	backup := &rclone.RcloneBackup{
		Name:        name,
		SFTPHost:    sftpHost,
		SFTPPort:    sftpPort,
		SFTPUser:    sftpUser,
		SFTPKeyPath: sftpKeyPath,
		RemotePath:  remotePath,
		Enabled:     enabled,
	}

	if err := rclone.Create(s.db, backup); err != nil {
		logger.Info("Error creating rclone backup: %v", err)
		http.Redirect(w, r, "/admin/rclone?error="+i18n.T(lang, "error_creating"), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/rclone?success=1", http.StatusSeeOther)
}

// handleAdminRcloneActions handles edit, delete, sync, test actions for rclone backups
func (s *Server) handleAdminRcloneActions(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL: /admin/rclone/{id}/{action}
	path := strings.TrimPrefix(r.URL.Path, "/admin/rclone/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch action {
	case "delete":
		s.handleRcloneDelete(w, r, id)
	case "sync":
		s.handleRcloneSync(w, r, id)
	case "test":
		s.handleRcloneTest(w, r, id)
	case "edit":
		s.handleRcloneEdit(w, r, id)
	default:
		// Show edit form
		s.handleRcloneEditForm(w, r, id)
	}
}

// handleRcloneDelete deletes a rclone backup configuration
func (s *Server) handleRcloneDelete(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := rclone.Delete(s.db, id); err != nil {
		logger.Info("Error deleting rclone backup: %v", err)
		http.Error(w, "Error deleting backup", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/rclone?deleted=1", http.StatusSeeOther)
}

// handleRcloneSync triggers a manual sync for a rclone backup
func (s *Server) handleRcloneSync(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := s.getLang(r)

	backup, err := rclone.GetByID(s.db, id)
	if err != nil {
		logger.Info("Error getting rclone backup: %v", err)
		http.Redirect(w, r, "/admin/rclone?error="+i18n.T(lang, "backup_not_found"), http.StatusSeeOther)
		return
	}

	if !rclone.IsRcloneInstalled() {
		http.Redirect(w, r, "/admin/rclone?error="+i18n.T(lang, "rclone.not_installed"), http.StatusSeeOther)
		return
	}

	// Run sync in background
	go func() {
		dataDir := s.cfg.DataDir
		result, syncErr := rclone.Sync(s.db, backup, dataDir)

		if syncErr != nil {
			logger.Info("Rclone backup sync error: %v", syncErr)
		} else if result != nil {
			logger.Info("Rclone backup sync completed: %d files, %s",
				result.FilesTransferred, rclone.FormatBytes(result.BytesTransferred))
		}
	}()

	http.Redirect(w, r, "/admin/rclone?syncing=1", http.StatusSeeOther)
}

// handleRcloneTest tests the SFTP connection for a rclone backup
func (s *Server) handleRcloneTest(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := s.getLang(r)

	backup, err := rclone.GetByID(s.db, id)
	if err != nil {
		logger.Info("Error getting rclone backup: %v", err)
		http.Redirect(w, r, "/admin/rclone?error="+i18n.T(lang, "backup_not_found"), http.StatusSeeOther)
		return
	}

	if !rclone.IsRcloneInstalled() {
		http.Redirect(w, r, "/admin/rclone?error="+i18n.T(lang, "rclone.not_installed"), http.StatusSeeOther)
		return
	}

	err = rclone.TestConnection(backup)
	if err != nil {
		logger.Info("Rclone connection test failed: %v", err)
		http.Redirect(w, r, "/admin/rclone?test_error="+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/rclone?test_success=1", http.StatusSeeOther)
}

// handleRcloneEditForm shows the edit form for a rclone backup
func (s *Server) handleRcloneEditForm(w http.ResponseWriter, r *http.Request, id int) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	lang := s.getLang(r)

	backup, err := rclone.GetByID(s.db, id)
	if err != nil {
		http.Redirect(w, r, "/admin/rclone?error=not_found", http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{
		"Session": session,
		"Title":   i18n.T(lang, "rclone.edit"),
		"Lang":    lang,
		"Backup":  backup,
	}

	if err := s.templates.ExecuteTemplate(w, "admin_rclone_edit.html", data); err != nil {
		logger.Info("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleRcloneEdit processes the edit form submission
func (s *Server) handleRcloneEdit(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := s.getLang(r)

	backup, err := rclone.GetByID(s.db, id)
	if err != nil {
		http.Redirect(w, r, "/admin/rclone?error=not_found", http.StatusSeeOther)
		return
	}

	backup.Name = strings.TrimSpace(r.FormValue("name"))
	backup.SFTPHost = strings.TrimSpace(r.FormValue("sftp_host"))
	backup.SFTPUser = strings.TrimSpace(r.FormValue("sftp_user"))
	backup.SFTPKeyPath = strings.TrimSpace(r.FormValue("sftp_key_path"))
	backup.RemotePath = strings.TrimSpace(r.FormValue("remote_path"))
	backup.Enabled = r.FormValue("enabled") == "on"

	// Parse port
	if portStr := r.FormValue("sftp_port"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 && p < 65536 {
			backup.SFTPPort = p
		}
	}

	// Schedule fields
	backup.SyncEnabled = r.FormValue("sync_enabled") == "on"
	backup.SyncFrequency = strings.TrimSpace(r.FormValue("sync_frequency"))
	backup.SyncTime = strings.TrimSpace(r.FormValue("sync_time"))

	// Parse day of week (0-6)
	if dowStr := r.FormValue("sync_day_of_week"); dowStr != "" {
		if dow, err := strconv.Atoi(dowStr); err == nil {
			backup.SyncDayOfWeek = &dow
		}
	}

	// Parse day of month (1-31)
	if domStr := r.FormValue("sync_day_of_month"); domStr != "" {
		if dom, err := strconv.Atoi(domStr); err == nil {
			backup.SyncDayOfMonth = &dom
		}
	}

	// Parse interval minutes
	if intervalStr := r.FormValue("sync_interval_minutes"); intervalStr != "" {
		if interval, err := strconv.Atoi(intervalStr); err == nil {
			backup.SyncIntervalMinutes = interval
		}
	}

	// Defaults for schedule
	if backup.SyncFrequency == "" {
		backup.SyncFrequency = "daily"
	}
	if backup.SyncTime == "" {
		backup.SyncTime = "02:00"
	}
	if backup.SyncIntervalMinutes == 0 {
		backup.SyncIntervalMinutes = 60
	}

	if backup.Name == "" || backup.SFTPHost == "" || backup.SFTPUser == "" || backup.RemotePath == "" {
		http.Redirect(w, r, "/admin/rclone/"+strconv.Itoa(id)+"?error="+i18n.T(lang, "missing_fields"), http.StatusSeeOther)
		return
	}

	if err := rclone.Update(s.db, backup); err != nil {
		logger.Info("Error updating rclone backup: %v", err)
		http.Redirect(w, r, "/admin/rclone/"+strconv.Itoa(id)+"?error="+i18n.T(lang, "error_updating"), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/rclone?updated=1", http.StatusSeeOther)
}
