// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"encoding/json"
	"github.com/juste-un-gars/anemone/internal/logger"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/incoming"
	"github.com/juste-un-gars/anemone/internal/shares"
	"github.com/juste-un-gars/anemone/internal/storage"
	"github.com/juste-un-gars/anemone/internal/sync"
	"github.com/juste-un-gars/anemone/internal/usbbackup"
	"github.com/juste-un-gars/anemone/internal/users"
)

// handleAdminUSBBackup displays the USB backup management page
func (s *Server) handleAdminUSBBackup(w http.ResponseWriter, r *http.Request) {
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

	// Get all USB backup configurations
	backups, err := usbbackup.GetAll(s.db)
	if err != nil {
		logger.Info("Error getting USB backups: %v", err)
		backups = []*usbbackup.USBBackup{}
	}

	// Detect available drives
	drives, err := usbbackup.DetectDrives()
	if err != nil {
		logger.Info("Error detecting drives: %v", err)
		drives = []usbbackup.DriveInfo{}
	}

	// Check mount status for each backup
	type BackupWithStatus struct {
		*usbbackup.USBBackup
		IsMounted     bool
		FreeSpace     string
		TotalSpace    string
		FreeBytes     int64
		TotalBytes    int64
		LastSyncAgo   string
		StatusClass   string
	}

	var backupsWithStatus []BackupWithStatus
	for _, b := range backups {
		bws := BackupWithStatus{
			USBBackup: b,
			IsMounted: b.IsMounted(),
		}

		// Get disk space if mounted
		if bws.IsMounted {
			for _, d := range drives {
				if d.MountPath == b.MountPath {
					bws.FreeBytes = d.FreeBytes
					bws.TotalBytes = d.TotalBytes
					bws.FreeSpace = usbbackup.FormatBytes(d.FreeBytes)
					bws.TotalSpace = usbbackup.FormatBytes(d.TotalBytes)
					break
				}
			}
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

	// Available drives (not yet configured)
	var availableDrives []usbbackup.DriveInfo
	for _, d := range drives {
		isConfigured := false
		for _, b := range backups {
			if b.MountPath == d.MountPath {
				isConfigured = true
				break
			}
		}
		if !isConfigured {
			availableDrives = append(availableDrives, d)
		}
	}

	// Detect unmounted disks (can be formatted)
	unmountedDisks, err := usbbackup.DetectUnmountedDisks()
	if err != nil {
		logger.Info("Error detecting unmounted disks: %v", err)
		unmountedDisks = []usbbackup.UnmountedDisk{}
	}

	// Get all shares for selection
	type ShareWithUser struct {
		ID       int
		Name     string
		UserID   int
		Username string
		Path     string
		Size     string
		SizeBytes int64
	}
	var sharesWithUsers []ShareWithUser
	allShares, _ := shares.GetAll(s.db)
	for _, sh := range allShares {
		swu := ShareWithUser{
			ID:     sh.ID,
			Name:   sh.Name,
			UserID: sh.UserID,
			Path:   sh.Path,
		}
		// Get username
		if user, err := users.GetByID(s.db, sh.UserID); err == nil {
			swu.Username = user.Username
		}
		// Calculate size
		if sizeBytes, err := usbbackup.CalculateDirSize(sh.Path); err == nil {
			swu.SizeBytes = sizeBytes
			swu.Size = usbbackup.FormatBytes(sizeBytes)
		}
		sharesWithUsers = append(sharesWithUsers, swu)
	}

	data := map[string]interface{}{
		"Session":         session,
		"Title":           i18n.T(lang, "usb_backup.title"),
		"Lang":            lang,
		"Backups":         backupsWithStatus,
		"AvailableDrives": availableDrives,
		"UnmountedDisks":  unmountedDisks,
		"AllShares":       sharesWithUsers,
		"FormatBytes":     usbbackup.FormatBytes,
		"Success":         r.URL.Query().Get("success") != "",
		"Syncing":         r.URL.Query().Get("syncing") != "",
		"Formatted":       r.URL.Query().Get("formatted") != "",
		"Error":           r.URL.Query().Get("error"),
	}

	if err := s.templates.ExecuteTemplate(w, "admin_usb_backup.html", data); err != nil {
		logger.Info("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleAdminUSBBackupAdd handles adding a new USB backup configuration
func (s *Server) handleAdminUSBBackupAdd(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	lang := s.getLang(r)

	if r.Method == http.MethodGet {
		// Show add form
		type ShareInfo struct {
			ID       int
			Name     string
			Username string
			Size     string
		}
		var shareList []ShareInfo
		allShares, _ := shares.GetAll(s.db)
		for _, sh := range allShares {
			si := ShareInfo{ID: sh.ID, Name: sh.Name}
			if user, err := users.GetByID(s.db, sh.UserID); err == nil {
				si.Username = user.Username
			}
			if sizeBytes, err := usbbackup.CalculateDirSize(sh.Path); err == nil {
				si.Size = usbbackup.FormatBytes(sizeBytes)
			}
			shareList = append(shareList, si)
		}

		// Get detected USB drives
		drives, _ := usbbackup.DetectDrives()

		data := struct {
			V2TemplateData
			Shares []ShareInfo
			Drives []usbbackup.DriveInfo
			Error  string
		}{
			V2TemplateData: V2TemplateData{
				Lang:       lang,
				Title:      i18n.T(lang, "v2.backups.add"),
				ActivePage: "backups",
				Session:    session,
			},
			Shares: shareList,
			Drives: drives,
			Error:  r.URL.Query().Get("error"),
		}

		tmpl := s.loadV2Page("v2_usb_backup_add.html", s.funcMap)
		if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
			logger.Info("Error rendering USB backup add template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	mountPath := strings.TrimSpace(r.FormValue("mount_path"))
	backupPath := strings.TrimSpace(r.FormValue("backup_path"))
	backupType := strings.TrimSpace(r.FormValue("backup_type"))
	enabled := r.FormValue("enabled") == "on"
	autoDetect := r.FormValue("auto_detect") == "on"

	if name == "" || mountPath == "" {
		http.Redirect(w, r, "/admin/usb-backup/add?error="+i18n.T(lang, "missing_fields"), http.StatusSeeOther)
		return
	}

	if backupPath == "" {
		backupPath = "anemone-backup"
	}

	// Default to full backup
	if backupType == "" {
		backupType = usbbackup.BackupTypeFull
	}

	// Parse selected shares (checkbox values)
	var selectedShareIDs []int
	if backupType == usbbackup.BackupTypeFull {
		for _, idStr := range r.Form["selected_shares"] {
			if id, err := strconv.Atoi(idStr); err == nil {
				selectedShareIDs = append(selectedShareIDs, id)
			}
		}
	}

	backup := &usbbackup.USBBackup{
		Name:       name,
		MountPath:  mountPath,
		BackupPath: backupPath,
		BackupType: backupType,
		Enabled:    enabled,
		AutoDetect: autoDetect,
	}
	backup.SetSelectedShareIDs(selectedShareIDs)

	if err := usbbackup.Create(s.db, backup); err != nil {
		logger.Info("Error creating USB backup: %v", err)
		http.Redirect(w, r, "/admin/usb-backup/add?error="+i18n.T(lang, "error_creating"), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/backups", http.StatusSeeOther)
}

// handleAdminUSBBackupActions handles edit, delete, sync actions for USB backups
func (s *Server) handleAdminUSBBackupActions(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL: /admin/usb-backup/{id}/{action}
	path := strings.TrimPrefix(r.URL.Path, "/admin/usb-backup/")
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
		s.handleUSBBackupDelete(w, r, id)
	case "sync":
		s.handleUSBBackupSync(w, r, id)
	case "edit":
		s.handleUSBBackupEdit(w, r, id)
	default:
		// Show edit form
		s.handleUSBBackupEditForm(w, r, id)
	}
}

// handleUSBBackupDelete deletes a USB backup configuration
func (s *Server) handleUSBBackupDelete(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := usbbackup.Delete(s.db, id); err != nil {
		logger.Info("Error deleting USB backup: %v", err)
		http.Error(w, "Error deleting backup", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/usb-backup?deleted=1", http.StatusSeeOther)
}

// handleUSBBackupSync triggers a manual sync for a USB backup
func (s *Server) handleUSBBackupSync(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := s.getLang(r)

	backup, err := usbbackup.GetByID(s.db, id)
	if err != nil {
		logger.Info("Error getting USB backup: %v", err)
		http.Redirect(w, r, "/admin/usb-backup?error="+i18n.T(lang, "backup_not_found"), http.StatusSeeOther)
		return
	}

	if !backup.IsMounted() {
		http.Redirect(w, r, "/admin/usb-backup?error="+i18n.T(lang, "drive_not_mounted"), http.StatusSeeOther)
		return
	}

	// Get server name
	serverName, _ := sync.GetServerName(s.db)
	if serverName == "" {
		serverName = "anemone"
	}

	// Get master key
	var masterKey string
	if err := s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey); err != nil {
		logger.Info("Error getting master key: %v", err)
		http.Redirect(w, r, "/admin/usb-backup?error=internal_error", http.StatusSeeOther)
		return
	}

	// Get data directory for config backup
	dataDir := s.cfg.DataDir

	// Run sync in background
	go func() {
		var result *usbbackup.SyncResult
		var syncErr error

		if backup.BackupType == usbbackup.BackupTypeConfig {
			// Config-only backup
			configInfo := &usbbackup.ConfigBackupInfo{
				DataDir:  dataDir,
				DBPath:   filepath.Join(dataDir, "db", "anemone.db"),
				CertsDir: filepath.Join(dataDir, "certs"),
				SMBConf:  filepath.Join(dataDir, "smb", "smb.conf"),
			}
			result, syncErr = usbbackup.SyncConfig(s.db, backup, configInfo, masterKey, serverName)
		} else {
			// Full backup (config + data)
			// First backup config
			configInfo := &usbbackup.ConfigBackupInfo{
				DataDir:  dataDir,
				DBPath:   filepath.Join(dataDir, "db", "anemone.db"),
				CertsDir: filepath.Join(dataDir, "certs"),
				SMBConf:  filepath.Join(dataDir, "smb", "smb.conf"),
			}
			configResult, _ := usbbackup.SyncConfig(s.db, backup, configInfo, masterKey, serverName)

			// Then backup selected shares
			result, syncErr = usbbackup.SyncAllShares(s.db, backup, masterKey, serverName)
			if result != nil && configResult != nil {
				result.FilesAdded += configResult.FilesAdded
				result.BytesSynced += configResult.BytesSynced
			}
		}

		if syncErr != nil {
			logger.Info("USB backup sync error: %v", syncErr)
		} else if result != nil {
			logger.Info("USB backup sync completed: %d added, %d updated, %d deleted, %s",
				result.FilesAdded, result.FilesUpdated, result.FilesDeleted,
				usbbackup.FormatBytes(result.BytesSynced))
		}
	}()

	http.Redirect(w, r, "/admin/usb-backup?syncing=1", http.StatusSeeOther)
}

// handleUSBBackupEditForm shows the edit form for a USB backup
func (s *Server) handleUSBBackupEditForm(w http.ResponseWriter, r *http.Request, id int) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	lang := s.getLang(r)

	backup, err := usbbackup.GetByID(s.db, id)
	if err != nil {
		http.Redirect(w, r, "/admin/usb-backup?error=not_found", http.StatusSeeOther)
		return
	}

	// Get all shares for selection
	type ShareWithUser struct {
		ID         int
		Name       string
		UserID     int
		Username   string
		Path       string
		Size       string
		SizeBytes  int64
		IsSelected bool
	}
	var sharesWithUsers []ShareWithUser
	selectedIDs := backup.GetSelectedShareIDs()
	allShares, _ := shares.GetAll(s.db)
	for _, sh := range allShares {
		swu := ShareWithUser{
			ID:     sh.ID,
			Name:   sh.Name,
			UserID: sh.UserID,
			Path:   sh.Path,
		}
		// Get username
		if user, err := users.GetByID(s.db, sh.UserID); err == nil {
			swu.Username = user.Username
		}
		// Calculate size
		if sizeBytes, err := usbbackup.CalculateDirSize(sh.Path); err == nil {
			swu.SizeBytes = sizeBytes
			swu.Size = usbbackup.FormatBytes(sizeBytes)
		}
		// Check if selected
		if len(selectedIDs) == 0 {
			swu.IsSelected = true // All selected by default
		} else {
			for _, selID := range selectedIDs {
				if selID == sh.ID {
					swu.IsSelected = true
					break
				}
			}
		}
		sharesWithUsers = append(sharesWithUsers, swu)
	}

	data := struct {
		V2TemplateData
		Backup    *usbbackup.USBBackup
		AllShares []ShareWithUser
	}{
		V2TemplateData: V2TemplateData{
			Lang:       lang,
			Title:      i18n.T(lang, "usb_backup.edit"),
			ActivePage: "backups",
			Session:    session,
		},
		Backup:    backup,
		AllShares: sharesWithUsers,
	}

	tmpl := s.loadV2Page("v2_usb_backup_edit.html", s.funcMap)
	if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
		logger.Info("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleUSBBackupEdit processes the edit form submission
func (s *Server) handleUSBBackupEdit(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := s.getLang(r)

	backup, err := usbbackup.GetByID(s.db, id)
	if err != nil {
		http.Redirect(w, r, "/admin/usb-backup?error=not_found", http.StatusSeeOther)
		return
	}

	backup.Name = strings.TrimSpace(r.FormValue("name"))
	backup.MountPath = strings.TrimSpace(r.FormValue("mount_path"))
	backup.BackupPath = strings.TrimSpace(r.FormValue("backup_path"))
	backup.BackupType = strings.TrimSpace(r.FormValue("backup_type"))
	backup.Enabled = r.FormValue("enabled") == "on"
	backup.AutoDetect = r.FormValue("auto_detect") == "on"

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
		backup.SyncTime = "23:00"
	}
	if backup.SyncIntervalMinutes == 0 {
		backup.SyncIntervalMinutes = 60
	}

	// Default to full backup
	if backup.BackupType == "" {
		backup.BackupType = usbbackup.BackupTypeFull
	}

	// Parse selected shares (checkbox values)
	var selectedShareIDs []int
	if backup.BackupType == usbbackup.BackupTypeFull {
		for _, idStr := range r.Form["selected_shares"] {
			if shareID, err := strconv.Atoi(idStr); err == nil {
				selectedShareIDs = append(selectedShareIDs, shareID)
			}
		}
	}
	backup.SetSelectedShareIDs(selectedShareIDs)

	if backup.Name == "" || backup.MountPath == "" {
		http.Redirect(w, r, "/admin/usb-backup/"+strconv.Itoa(id)+"?error="+i18n.T(lang, "missing_fields"), http.StatusSeeOther)
		return
	}

	if err := usbbackup.Update(s.db, backup); err != nil {
		logger.Info("Error updating USB backup: %v", err)
		http.Redirect(w, r, "/admin/usb-backup/"+strconv.Itoa(id)+"?error="+i18n.T(lang, "error_updating"), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/usb-backup?updated=1", http.StatusSeeOther)
}

// handleAdminUSBBackupAPI provides JSON API for USB backup status
func (s *Server) handleAdminUSBBackupAPI(w http.ResponseWriter, r *http.Request) {
	backups, err := usbbackup.GetAll(s.db)
	if err != nil {
		http.Error(w, "Error getting backups", http.StatusInternalServerError)
		return
	}

	type BackupStatus struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		IsMounted  bool   `json:"is_mounted"`
		LastStatus string `json:"last_status"`
		LastSync   string `json:"last_sync,omitempty"`
	}

	var statuses []BackupStatus
	for _, b := range backups {
		status := BackupStatus{
			ID:         b.ID,
			Name:       b.Name,
			IsMounted:  b.IsMounted(),
			LastStatus: b.LastStatus,
		}
		if b.LastSync != nil {
			status.LastSync = b.LastSync.Format("2006-01-02 15:04")
		}
		statuses = append(statuses, status)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

// handleAdminUSBDrives provides JSON API to list detected drives
func (s *Server) handleAdminUSBDrives(w http.ResponseWriter, r *http.Request) {
	drives, err := usbbackup.DetectDrives()
	if err != nil {
		http.Error(w, "Error detecting drives", http.StatusInternalServerError)
		return
	}

	type DriveResponse struct {
		MountPath  string `json:"mount_path"`
		Label      string `json:"label"`
		Filesystem string `json:"filesystem"`
		TotalGB    string `json:"total_gb"`
		FreeGB     string `json:"free_gb"`
		Removable  bool   `json:"removable"`
	}

	var response []DriveResponse
	for _, d := range drives {
		response = append(response, DriveResponse{
			MountPath:  d.MountPath,
			Label:      d.Label,
			Filesystem: d.Filesystem,
			TotalGB:    usbbackup.FormatBytes(d.TotalBytes),
			FreeGB:     usbbackup.FormatBytes(d.FreeBytes),
			Removable:  d.IsRemovable,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleAdminUSBFormat handles formatting a USB disk
func (s *Server) handleAdminUSBFormat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := s.getLang(r)

	device := strings.TrimSpace(r.FormValue("device"))
	filesystem := strings.TrimSpace(r.FormValue("filesystem"))
	label := strings.TrimSpace(r.FormValue("label"))

	// Validate device
	if device == "" {
		http.Redirect(w, r, "/admin/usb-backup?error="+i18n.T(lang, "usb_format.error.no_device"), http.StatusSeeOther)
		return
	}

	// Validate filesystem (only allow FAT32 and exFAT for USB)
	if filesystem != "fat32" && filesystem != "exfat" {
		http.Redirect(w, r, "/admin/usb-backup?error="+i18n.T(lang, "usb_format.error.invalid_fs"), http.StatusSeeOther)
		return
	}

	// Validate device path for security
	if err := storage.ValidateDevicePath(device); err != nil {
		logger.Info("Invalid device path: %s - %v", device, err)
		http.Redirect(w, r, "/admin/usb-backup?error="+i18n.T(lang, "usb_format.error.invalid_device"), http.StatusSeeOther)
		return
	}

	// Check if device is in use
	inUse, usedBy, err := storage.IsDiskInUse(device)
	if err != nil {
		logger.Info("Error checking disk: %v", err)
		http.Redirect(w, r, "/admin/usb-backup?error="+i18n.T(lang, "usb_format.error.check_failed"), http.StatusSeeOther)
		return
	}
	if inUse {
		logger.Info("Disk %s is in use: %s", device, usedBy)
		http.Redirect(w, r, "/admin/usb-backup?error="+i18n.T(lang, "usb_format.error.in_use"), http.StatusSeeOther)
		return
	}

	// Create partition and format
	logger.Info("Formatting %s as %s with label %q", device, filesystem, label)

	opts := storage.CreatePartitionOptions{
		Device:     device,
		TableType:  "gpt",
		Filesystem: filesystem,
		Label:      label,
	}

	if err := storage.CreatePartition(opts); err != nil {
		logger.Info("Format failed: %v", err)
		http.Redirect(w, r, "/admin/usb-backup?error="+i18n.T(lang, "usb_format.error.format_failed"), http.StatusSeeOther)
		return
	}

	logger.Info("Successfully formatted %s as %s", device, filesystem)
	http.Redirect(w, r, "/admin/usb-backup?formatted=1", http.StatusSeeOther)
}

// handleAdminUSBUnmountedDisks provides JSON API to list unmounted disks
func (s *Server) handleAdminUSBUnmountedDisks(w http.ResponseWriter, r *http.Request) {
	disks, err := usbbackup.DetectUnmountedDisks()
	if err != nil {
		http.Error(w, "Error detecting disks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(disks)
}
