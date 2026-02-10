// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/incoming"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/rclone"
)

// validRemoteName matches safe rclone remote names (alphanumeric, dash, underscore, dot).
var validRemoteName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// handleAdminRclone displays the rclone backup management page
func (s *Server) handleAdminRclone(w http.ResponseWriter, r *http.Request) {
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

	// Get SSH key info
	sshKeyInfo, _ := rclone.GetSSHKeyInfo(s.cfg.DataDir)

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
		"SSHKeyInfo":       sshKeyInfo,
		"FormatBytes":      rclone.FormatBytes,
		"Success":          r.URL.Query().Get("success") != "",
		"Syncing":          r.URL.Query().Get("syncing") != "",
		"TestSuccess":      r.URL.Query().Get("test_success") != "",
		"TestError":        r.URL.Query().Get("test_error"),
		"Error":            r.URL.Query().Get("error"),
		"KeyGenerated":     r.URL.Query().Get("key_generated") != "",
	}

	if err := s.templates.ExecuteTemplate(w, "admin_rclone.html", data); err != nil {
		logger.Info("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleAdminRcloneAdd handles GET (show form) and POST (create) for rclone backup configurations
func (s *Server) handleAdminRcloneAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.handleRcloneAddForm(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := s.getLang(r)
	providerType := strings.TrimSpace(r.FormValue("provider_type"))
	if providerType == "" {
		providerType = rclone.ProviderSFTP
	}

	name := strings.TrimSpace(r.FormValue("name"))
	remotePath := strings.TrimSpace(r.FormValue("remote_path"))
	enabled := r.FormValue("enabled") == "on"

	if name == "" || remotePath == "" {
		http.Redirect(w, r, "/admin/rclone/add?error="+i18n.T(lang, "missing_fields"), http.StatusSeeOther)
		return
	}

	backup := &rclone.RcloneBackup{
		Name:         name,
		RemotePath:   remotePath,
		Enabled:      enabled,
		ProviderType: providerType,
	}

	switch providerType {
	case rclone.ProviderSFTP:
		backup.SFTPHost = strings.TrimSpace(r.FormValue("sftp_host"))
		backup.SFTPUser = strings.TrimSpace(r.FormValue("sftp_user"))
		backup.SFTPKeyPath = strings.TrimSpace(r.FormValue("sftp_key_path"))
		if backup.SFTPHost == "" || backup.SFTPUser == "" {
			http.Redirect(w, r, "/admin/rclone/add?error="+i18n.T(lang, "missing_fields"), http.StatusSeeOther)
			return
		}
		sftpPort := 22
		if portStr := r.FormValue("sftp_port"); portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil && p > 0 && p < 65536 {
				sftpPort = p
			}
		}
		backup.SFTPPort = sftpPort

	case rclone.ProviderS3:
		cfg := map[string]string{
			"endpoint":          strings.TrimSpace(r.FormValue("s3_endpoint")),
			"region":            strings.TrimSpace(r.FormValue("s3_region")),
			"access_key_id":     strings.TrimSpace(r.FormValue("s3_access_key_id")),
			"secret_access_key": strings.TrimSpace(r.FormValue("s3_secret_access_key")),
			"s3_provider":       strings.TrimSpace(r.FormValue("s3_provider")),
		}
		if cfg["access_key_id"] == "" {
			http.Redirect(w, r, "/admin/rclone/add?error="+i18n.T(lang, "missing_fields"), http.StatusSeeOther)
			return
		}
		backup.ProviderConfig = cfg

	case rclone.ProviderWebDAV:
		pass := strings.TrimSpace(r.FormValue("webdav_pass"))
		if pass != "" {
			obscured, err := rclone.ObscurePassword(pass)
			if err == nil {
				pass = obscured
			}
		}
		cfg := map[string]string{
			"url":    strings.TrimSpace(r.FormValue("webdav_url")),
			"vendor": strings.TrimSpace(r.FormValue("webdav_vendor")),
			"user":   strings.TrimSpace(r.FormValue("webdav_user")),
			"pass":   pass,
		}
		if cfg["url"] == "" {
			http.Redirect(w, r, "/admin/rclone/add?error="+i18n.T(lang, "missing_fields"), http.StatusSeeOther)
			return
		}
		backup.ProviderConfig = cfg

	case rclone.ProviderRemote:
		remoteName := strings.TrimSpace(r.FormValue("remote_name"))
		if remoteName == "" {
			http.Redirect(w, r, "/admin/rclone/add?error="+i18n.T(lang, "missing_fields"), http.StatusSeeOther)
			return
		}
		// Strip trailing colon if user included it
		remoteName = strings.TrimSuffix(remoteName, ":")
		if !validRemoteName.MatchString(remoteName) {
			http.Redirect(w, r, "/admin/rclone/add?error="+i18n.T(lang, "rclone.remote.invalid_name"), http.StatusSeeOther)
			return
		}
		backup.ProviderConfig = map[string]string{"remote_name": remoteName}
	}

	// Encryption (optional, all providers)
	if cryptPass := strings.TrimSpace(r.FormValue("crypt_password")); cryptPass != "" && r.FormValue("crypt_enabled") == "on" {
		if backup.ProviderConfig == nil {
			backup.ProviderConfig = map[string]string{}
		}
		obscured, err := rclone.ObscurePassword(cryptPass)
		if err == nil {
			backup.ProviderConfig["crypt_password"] = obscured
		}
	}

	if err := rclone.Create(s.db, backup); err != nil {
		logger.Info("Error creating rclone backup: %v", err)
		http.Redirect(w, r, "/admin/rclone/add?error="+i18n.T(lang, "error_creating"), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/backups?tab=cloud&success=1", http.StatusSeeOther)
}

// handleRcloneAddForm shows the add form for a new rclone backup
func (s *Server) handleRcloneAddForm(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	lang := s.getLang(r)

	// Get available named remotes
	remotes, _ := rclone.ListRemotes()

	// Get SSH key info for SFTP default
	sshKeyInfo, _ := rclone.GetSSHKeyInfo(s.cfg.DataDir)

	data := struct {
		V2TemplateData
		Remotes    []rclone.RemoteInfo
		SSHKeyPath string
		Error      string
	}{
		V2TemplateData: V2TemplateData{
			Lang:       lang,
			Title:      i18n.T(lang, "rclone.add"),
			ActivePage: "backups",
			Session:    session,
		},
		Remotes: remotes,
		Error:   r.URL.Query().Get("error"),
	}

	if sshKeyInfo != nil && sshKeyInfo.Exists {
		data.SSHKeyPath = sshKeyInfo.RelativePath
	}

	tmpl := s.loadV2Page("v2_rclone_add.html", s.funcMap)
	if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
		logger.Info("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleAdminRcloneListRemotes returns available named remotes as JSON
func (s *Server) handleAdminRcloneListRemotes(w http.ResponseWriter, r *http.Request) {
	remotes, err := rclone.ListRemotes()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(remotes)
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

	err = rclone.TestConnection(backup, s.cfg.DataDir)
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

	remotes, _ := rclone.ListRemotes()

	data := struct {
		V2TemplateData
		Backup  *rclone.RcloneBackup
		Remotes []rclone.RemoteInfo
	}{
		V2TemplateData: V2TemplateData{
			Lang:       lang,
			Title:      i18n.T(lang, "rclone.edit"),
			ActivePage: "backups",
			Session:    session,
		},
		Backup:  backup,
		Remotes: remotes,
	}

	tmpl := s.loadV2Page("v2_rclone_edit.html", s.funcMap)
	if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
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

	// Common fields
	backup.Name = strings.TrimSpace(r.FormValue("name"))
	backup.RemotePath = strings.TrimSpace(r.FormValue("remote_path"))
	backup.Enabled = r.FormValue("enabled") == "on"

	// Provider-specific fields (provider type is read-only, kept from DB)
	switch backup.ProviderType {
	case rclone.ProviderSFTP:
		backup.SFTPHost = strings.TrimSpace(r.FormValue("sftp_host"))
		backup.SFTPUser = strings.TrimSpace(r.FormValue("sftp_user"))
		backup.SFTPKeyPath = strings.TrimSpace(r.FormValue("sftp_key_path"))
		if portStr := r.FormValue("sftp_port"); portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil && p > 0 && p < 65536 {
				backup.SFTPPort = p
			}
		}
		if backup.SFTPHost == "" || backup.SFTPUser == "" {
			http.Redirect(w, r, "/admin/rclone/"+strconv.Itoa(id)+"?error="+i18n.T(lang, "missing_fields"), http.StatusSeeOther)
			return
		}

	case rclone.ProviderS3:
		cfg := backup.ProviderConfig
		if cfg == nil {
			cfg = map[string]string{}
		}
		cfg["endpoint"] = strings.TrimSpace(r.FormValue("s3_endpoint"))
		cfg["region"] = strings.TrimSpace(r.FormValue("s3_region"))
		cfg["access_key_id"] = strings.TrimSpace(r.FormValue("s3_access_key_id"))
		cfg["s3_provider"] = strings.TrimSpace(r.FormValue("s3_provider"))
		// Only update secret if provided (non-empty)
		if sk := strings.TrimSpace(r.FormValue("s3_secret_access_key")); sk != "" {
			cfg["secret_access_key"] = sk
		}
		backup.ProviderConfig = cfg

	case rclone.ProviderWebDAV:
		cfg := backup.ProviderConfig
		if cfg == nil {
			cfg = map[string]string{}
		}
		cfg["url"] = strings.TrimSpace(r.FormValue("webdav_url"))
		cfg["vendor"] = strings.TrimSpace(r.FormValue("webdav_vendor"))
		cfg["user"] = strings.TrimSpace(r.FormValue("webdav_user"))
		// Only update password if provided
		if pass := strings.TrimSpace(r.FormValue("webdav_pass")); pass != "" {
			obscured, err := rclone.ObscurePassword(pass)
			if err == nil {
				cfg["pass"] = obscured
			}
		}
		backup.ProviderConfig = cfg

	case rclone.ProviderRemote:
		cfg := backup.ProviderConfig
		if cfg == nil {
			cfg = map[string]string{}
		}
		remoteName := strings.TrimSpace(r.FormValue("remote_name"))
		remoteName = strings.TrimSuffix(remoteName, ":")
		if remoteName != "" && !validRemoteName.MatchString(remoteName) {
			http.Redirect(w, r, "/admin/rclone/"+strconv.Itoa(id)+"?error="+i18n.T(lang, "rclone.remote.invalid_name"), http.StatusSeeOther)
			return
		}
		cfg["remote_name"] = remoteName
		backup.ProviderConfig = cfg
	}

	// Encryption
	if cryptPass := strings.TrimSpace(r.FormValue("crypt_password")); cryptPass != "" {
		if backup.ProviderConfig == nil {
			backup.ProviderConfig = map[string]string{}
		}
		obscured, err := rclone.ObscurePassword(cryptPass)
		if err == nil {
			backup.ProviderConfig["crypt_password"] = obscured
		}
	}
	// Allow disabling encryption
	if r.FormValue("crypt_enabled") != "on" {
		if backup.ProviderConfig != nil {
			delete(backup.ProviderConfig, "crypt_password")
		}
	}

	// Schedule fields
	backup.SyncEnabled = r.FormValue("sync_enabled") == "on"
	backup.SyncFrequency = strings.TrimSpace(r.FormValue("sync_frequency"))
	backup.SyncTime = strings.TrimSpace(r.FormValue("sync_time"))

	if dowStr := r.FormValue("sync_day_of_week"); dowStr != "" {
		if dow, err := strconv.Atoi(dowStr); err == nil {
			backup.SyncDayOfWeek = &dow
		}
	}
	if domStr := r.FormValue("sync_day_of_month"); domStr != "" {
		if dom, err := strconv.Atoi(domStr); err == nil {
			backup.SyncDayOfMonth = &dom
		}
	}
	if intervalStr := r.FormValue("sync_interval_minutes"); intervalStr != "" {
		if interval, err := strconv.Atoi(intervalStr); err == nil {
			backup.SyncIntervalMinutes = interval
		}
	}

	if backup.SyncFrequency == "" {
		backup.SyncFrequency = "daily"
	}
	if backup.SyncTime == "" {
		backup.SyncTime = "02:00"
	}
	if backup.SyncIntervalMinutes == 0 {
		backup.SyncIntervalMinutes = 60
	}

	if backup.Name == "" || backup.RemotePath == "" {
		http.Redirect(w, r, "/admin/rclone/"+strconv.Itoa(id)+"?error="+i18n.T(lang, "missing_fields"), http.StatusSeeOther)
		return
	}

	if err := rclone.Update(s.db, backup); err != nil {
		logger.Info("Error updating rclone backup: %v", err)
		http.Redirect(w, r, "/admin/rclone/"+strconv.Itoa(id)+"?error="+i18n.T(lang, "error_updating"), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/backups?tab=cloud&updated=1", http.StatusSeeOther)
}

// handleAdminRcloneKeyInfo returns SSH key information as JSON
func (s *Server) handleAdminRcloneKeyInfo(w http.ResponseWriter, r *http.Request) {
	keyInfo, err := rclone.GetSSHKeyInfo(s.cfg.DataDir)
	if err != nil {
		logger.Info("Error getting SSH key info: %v", err)
		http.Error(w, "Error getting key info", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keyInfo)
}

// handleAdminRcloneGenerateKey generates a new SSH key pair
func (s *Server) handleAdminRcloneGenerateKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keyInfo, err := rclone.GenerateSSHKey(s.cfg.DataDir)
	if err != nil {
		logger.Info("Error generating SSH key: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	logger.Info("SSH key generated for rclone at %s", keyInfo.KeyPath)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keyInfo)
}
