// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/juste-un-gars/anemone/internal/logger"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	backupPkg "github.com/juste-un-gars/anemone/internal/backup"
	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/bulkrestore"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/peers"
	"github.com/juste-un-gars/anemone/internal/serverbackup"
	"github.com/juste-un-gars/anemone/internal/sync"
	"github.com/juste-un-gars/anemone/internal/updater"
)

func (s *Server) handleAdminBackupExport(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	if r.Method == "GET" {
		// Display export form
		data := struct {
			Lang    string
			Title   string
			Session *auth.Session
		}{
			Lang:    lang,
			Title:   i18n.T(lang, "backup.export.title"),
			Session: session,
		}
		if err := s.templates.ExecuteTemplate(w, "admin_backup_export.html", data); err != nil {
			logger.Info("Error rendering backup export template: %v", err)
		}
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get passphrase from form
	passphrase := r.FormValue("passphrase")
	if passphrase == "" {
		http.Error(w, "Passphrase is required", http.StatusBadRequest)
		return
	}

	// Confirm passphrase
	passphraseConfirm := r.FormValue("passphrase_confirm")
	if passphrase != passphraseConfirm {
		http.Error(w, "Passphrases do not match", http.StatusBadRequest)
		return
	}

	// Get server name (optional)
	serverName := r.FormValue("server_name")
	if serverName == "" {
		serverName = "Anemone Server"
	}

	// Export configuration
	backup, err := backupPkg.ExportConfiguration(s.db, serverName)
	if err != nil {
		logger.Info("Error exporting configuration: %v", err)
		http.Error(w, "Failed to export configuration", http.StatusInternalServerError)
		return
	}

	// Encrypt backup
	encryptedData, err := backupPkg.EncryptBackup(backup, passphrase)
	if err != nil {
		logger.Info("Error encrypting backup: %v", err)
		http.Error(w, "Failed to encrypt backup", http.StatusInternalServerError)
		return
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("anemone_backup_%s.enc", timestamp)

	// Send encrypted file as download
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(encryptedData)))
	w.WriteHeader(http.StatusOK)
	w.Write(encryptedData)

	logger.Info("Admin exported server configuration (backup size: %d bytes)", len(encryptedData))
}

// handleAdminBackup displays the list of server backups

func (s *Server) handleAdminBackup(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Get backup directory path
	backupDir := filepath.Join(s.cfg.DataDir, "backups", "server")

	// List all backups
	backupFiles, err := serverbackup.ListBackups(backupDir)
	if err != nil {
		logger.Info("Error listing backups: %v", err)
		http.Error(w, "Failed to list backups", http.StatusInternalServerError)
		return
	}

	// Format backups for template
	type BackupInfo struct {
		Filename      string
		FormattedDate string
		FormattedSize string
	}

	backups := make([]BackupInfo, 0, len(backupFiles))
	for _, bf := range backupFiles {
		// Format size in KB or MB
		var sizeStr string
		if bf.Size < 1024*1024 {
			sizeStr = fmt.Sprintf("%.1f KB", float64(bf.Size)/1024)
		} else {
			sizeStr = fmt.Sprintf("%.2f MB", float64(bf.Size)/(1024*1024))
		}

		backups = append(backups, BackupInfo{
			Filename:      bf.Filename,
			FormattedDate: bf.CreatedAt.Format("02/01/2006 15:04:05"),
			FormattedSize: sizeStr,
		})
	}

	data := struct {
		Lang    string
		Session *auth.Session
		Backups []BackupInfo
	}{
		Lang:    lang,
		Session: session,
		Backups: backups,
	}

	if err := s.templates.ExecuteTemplate(w, "admin_backup.html", data); err != nil {
		logger.Info("Error rendering backup template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleAdminBackupCreate creates a manual server backup
func (s *Server) handleAdminBackupCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get backup directory path
	backupDir := filepath.Join(s.cfg.DataDir, "backups", "server")

	// Create backup
	backupPath, err := serverbackup.CreateServerBackup(s.db, backupDir)
	if err != nil {
		logger.Info("Error creating manual backup: %v", err)
		http.Error(w, "Failed to create backup", http.StatusInternalServerError)
		return
	}

	logger.Info("Manual server backup created: %s", backupPath)

	// Redirect back to backup list
	http.Redirect(w, r, "/admin/backup", http.StatusSeeOther)
}

// handleAdminBackupDownload downloads a backup re-encrypted with user passphrase
func (s *Server) handleAdminBackupDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get parameters from form
	filename := r.FormValue("filename")
	passphrase := r.FormValue("passphrase")

	if filename == "" || passphrase == "" {
		http.Error(w, "Missing filename or passphrase", http.StatusBadRequest)
		return
	}

	// Validate passphrase length
	if len(passphrase) < 12 {
		http.Error(w, "Passphrase must be at least 12 characters", http.StatusBadRequest)
		return
	}

	// Get backup directory path
	backupDir := filepath.Join(s.cfg.DataDir, "backups", "server")
	backupPath := filepath.Join(backupDir, filename)

	// Check if file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		http.Error(w, "Backup file not found", http.StatusNotFound)
		return
	}

	// Re-encrypt backup with user passphrase
	reEncryptedData, err := serverbackup.ReEncryptBackup(s.db, backupPath, passphrase)
	if err != nil {
		logger.Info("Error re-encrypting backup: %v", err)
		http.Error(w, "Failed to prepare backup for download", http.StatusInternalServerError)
		return
	}

	// Send as download
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(reEncryptedData)))
	w.WriteHeader(http.StatusOK)
	w.Write(reEncryptedData)

	logger.Info("Admin downloaded backup %s (re-encrypted, size: %d bytes)", filename, len(reEncryptedData))
}

// handleAdminBackupDelete deletes a server backup file
func (s *Server) handleAdminBackupDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get filename from form
	filename := r.FormValue("filename")
	if filename == "" {
		http.Error(w, "Missing filename", http.StatusBadRequest)
		return
	}

	// Security: prevent path traversal - filename must not contain directory separators
	// Note: we allow dots in filenames (e.g., "backup...sql") since separators are blocked
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Get backup directory path
	backupDir := filepath.Join(s.cfg.DataDir, "backups", "server")
	backupPath := filepath.Join(backupDir, filename)

	// Check if file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		http.Error(w, "Backup file not found", http.StatusNotFound)
		return
	}

	// Delete the backup file
	if err := os.Remove(backupPath); err != nil {
		logger.Info("Error deleting backup %s: %v", filename, err)
		http.Error(w, "Failed to delete backup", http.StatusInternalServerError)
		return
	}

	logger.Info("Admin deleted backup %s", filename)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Backup deleted successfully"))
}

// handleRestoreWarning displays the restore warning page

func (s *Server) handleAdminRestoreUsers(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Get current server name to filter backups
	currentServerName, err := sync.GetServerName(s.db)
	if err != nil {
		logger.Info("Error getting server name: %v", err)
		http.Error(w, "Failed to get server name", http.StatusInternalServerError)
		return
	}

	// Get all users (except admin)
	rows, err := s.db.Query("SELECT id, username FROM users WHERE is_admin = 0 ORDER BY username")
	if err != nil {
		logger.Info("Error getting users: %v", err)
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type UserBackup struct {
		UserID       int
		Username     string
		PeerID       int
		PeerName     string
		SourceServer string
		ShareName    string
		FileCount    int
		TotalSize    int64
		LastModified time.Time
	}

	var allBackups []UserBackup

	// Get master key for password decryption
	var masterKey string
	if err := s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey); err != nil {
		logger.Info("Error getting master key: %v", err)
		http.Error(w, "System configuration error", http.StatusInternalServerError)
		return
	}

	// For each user, check available backups on all peers
	for rows.Next() {
		var userID int
		var username string
		if err := rows.Scan(&userID, &username); err != nil {
			continue
		}

		// Get all peers
		allPeers, err := peers.GetAll(s.db)
		if err != nil {
			logger.Info("Error getting peers: %v", err)
			continue
		}

		// Query each peer for this user's backups
		// Note: We query ALL peers, even disabled ones, because we want to list
		// available backups for restoration (peers are disabled after server restore)
		for _, peer := range allPeers {
			// Build URL
			url := fmt.Sprintf("https://%s:%d/api/sync/list-user-backups?user_id=%d",
				peer.Address, peer.Port, userID)

			// Create HTTP client
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client := &http.Client{
				Transport: tr,
				Timeout:   10 * time.Second,
			}

			// Create request
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				continue
			}

			// Decrypt and add P2P authentication header
			if peer.Password != nil && len(*peer.Password) > 0 {
				peerPassword, err := peers.DecryptPeerPassword(peer.Password, masterKey)
				if err != nil {
					logger.Info("Error decrypting peer password: %v", err)
					continue
				}
				req.Header.Set("X-Sync-Password", peerPassword)
			}

			// Execute request
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				continue
			}

			// Parse response
			type BackupInfo struct {
				SourceServer string    `json:"source_server"`
				ShareName    string    `json:"share_name"`
				FileCount    int       `json:"file_count"`
				TotalSize    int64     `json:"total_size"`
				LastModified time.Time `json:"last_modified"`
			}
			var peerBackups []BackupInfo
			if err := json.NewDecoder(resp.Body).Decode(&peerBackups); err != nil {
				continue
			}

			// Add to results (filter by current server name)
			for _, backup := range peerBackups {
				// Only show backups from the current server
				if backup.SourceServer == currentServerName {
					allBackups = append(allBackups, UserBackup{
						UserID:       userID,
						Username:     username,
						PeerID:       peer.ID,
						PeerName:     peer.Name,
						SourceServer: backup.SourceServer,
						ShareName:    backup.ShareName,
						FileCount:    backup.FileCount,
						TotalSize:    backup.TotalSize,
						LastModified: backup.LastModified,
					})
				}
			}
		}
	}

	// Render template
	data := map[string]interface{}{
		"Session": session,
		"Lang":    lang,
		"Backups": allBackups,
	}

	if err := s.templates.ExecuteTemplate(w, "admin_restore_users.html", data); err != nil {
		logger.Info("Error executing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleAdminRestoreUsersRestore handles bulk restoration of a user's files
// POST /admin/restore-users/restore
func (s *Server) handleAdminRestoreUsersRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get parameters
	userIDStr := r.FormValue("user_id")
	peerIDStr := r.FormValue("peer_id")
	shareName := r.FormValue("share_name")
	sourceServer := r.FormValue("source_server")

	if userIDStr == "" || peerIDStr == "" || shareName == "" || sourceServer == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Missing parameters",
		})
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid user_id",
		})
		return
	}

	peerID, err := strconv.Atoi(peerIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid peer_id",
		})
		return
	}

	// Get username for logging
	var username string
	err = s.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "User not found",
		})
		return
	}

	logger.Info("Admin %s starting bulk restore for user %s (id %d) from peer %d share %s source %s",
		session.Username, username, userID, peerID, shareName, sourceServer)

	// Start bulk restore in background
	go func() {
		err := bulkrestore.BulkRestoreFromPeer(s.db, userID, peerID, shareName, sourceServer, s.cfg.DataDir, nil)
		if err != nil {
			logger.Info("Admin bulk restore failed for user %s: %v", username, err)
		} else {
			logger.Info("Admin bulk restore completed successfully for user %s", username)
		}
	}()

	// Return immediate response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Bulk restore started in background for user " + username,
	})
}

// handleAdminSystemUpdate displays the system update page

func (s *Server) handleAdminSystemUpdate(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Get update info
	updateInfo, err := updater.GetUpdateInfo(s.db)
	if err != nil {
		logger.Info("Error getting update info: %v", err)
		updateInfo = &updater.UpdateInfo{
			CurrentVersion: updater.GetCurrentVersion(),
			LatestVersion:  updater.GetCurrentVersion(),
			Available:      false,
		}
	}

	// Get last check time
	lastCheck, err := updater.GetLastUpdateCheck(s.db)
	if err != nil {
		logger.Info("Error getting last update check: %v", err)
	}

	data := TemplateData{
		Lang:       lang,
		Title:      i18n.T(lang, "update.page.title"),
		Session:    session,
		UpdateInfo: updateInfo,
		Data: map[string]interface{}{
			"LastCheck": lastCheck,
		},
	}

	if err := s.templates.ExecuteTemplate(w, "admin_system_update.html", data); err != nil {
		logger.Info("Error rendering system update template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleAdminSystemUpdateCheck triggers a manual update check
func (s *Server) handleAdminSystemUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := s.getLang(r)

	// Perform update check
	logger.Info("🔍 Manual update check triggered by admin")
	info, err := updater.CheckUpdate()
	if err != nil {
		logger.Info("Error checking for updates: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   i18n.T(lang, "update.check.error"),
		})
		return
	}

	// Save to database
	if err := updater.SaveUpdateInfo(s.db, info); err != nil {
		logger.Info("Error saving update info: %v", err)
	}

	// Log result
	if info.Available {
		logger.Info("✨ Update available: %s → %s", info.CurrentVersion, info.LatestVersion)
	} else {
		logger.Info("✅ Up to date: %s", info.CurrentVersion)
	}

	// Return result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"updateInfo":     info,
		"updateMessage":  i18n.T(lang, "update.check.success"),
	})
}

// handleAdminSystemUpdateInstall performs the automatic update
func (s *Server) handleAdminSystemUpdateInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := s.getLang(r)

	// Get update info from database
	updateInfo, err := updater.GetUpdateInfo(s.db)
	if err != nil {
		logger.Info("Error getting update info: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   i18n.T(lang, "update.install.error_check"),
		})
		return
	}

	// Check if an update is actually available
	if !updateInfo.Available {
		logger.Info("❌ Update installation requested but no update is available")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   i18n.T(lang, "update.install.no_update"),
		})
		return
	}

	// Launch the auto-update script
	logger.Info("🚀 Starting automatic update: %s → %s", updateInfo.CurrentVersion, updateInfo.LatestVersion)

	// No password needed - sudo NOPASSWD should be configured for systemctl restart
	if err := updater.PerformAutoUpdate(updateInfo.LatestVersion); err != nil {
		logger.Info("Error starting auto-update: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   i18n.T(lang, "update.install.error_start"),
		})
		return
	}

	// Return success response
	// The update script will continue in the background
	logger.Info("✅ Auto-update script launched successfully")
	logger.Info("📝 Update log: %s", updater.GetUpdateLogPath())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": i18n.T(lang, "update.install.success"),
		"logPath": updater.GetUpdateLogPath(),
	})
}
