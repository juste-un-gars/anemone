// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"fmt"
	"github.com/juste-un-gars/anemone/internal/logger"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/peers"
	"github.com/juste-un-gars/anemone/internal/quota"
	"github.com/juste-un-gars/anemone/internal/shares"
	"github.com/juste-un-gars/anemone/internal/smb"
	"github.com/juste-un-gars/anemone/internal/sync"
	"github.com/juste-un-gars/anemone/internal/trash"
	"github.com/juste-un-gars/anemone/internal/updater"
	"github.com/juste-un-gars/anemone/internal/users"
)

// handleHome handles the root path
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	// Check setup first
	if !s.isSetupCompleted() {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}

	// If not authenticated, redirect to login
	if !auth.IsAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Redirect to dashboard
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// handleDashboard handles the dashboard page
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Get stats
	stats := s.getDashboardStats(session, lang)

	// Get update info for admins
	var updateInfo *updater.UpdateInfo
	if session.IsAdmin {
		var err error
		updateInfo, err = updater.GetUpdateInfo(s.db)
		if err != nil {
			logger.Info("Warning: Failed to get update info: %v", err)
			// Continue without update info rather than failing
		}
	}

	if session.IsAdmin {
		// Admin: render v2 dashboard
		activity := s.getRecentActivity(lang, 5)

		data := V2DashboardData{
			V2TemplateData: V2TemplateData{
				Lang:       lang,
				Title:      i18n.T(lang, "v2.nav.dashboard"),
				ActivePage: "dashboard",
				Session:    session,
			},
			Stats:          stats,
			RecentActivity: activity,
			UpdateInfo:     updateInfo,
		}

		tmpl := s.loadV2Page("v2_dashboard.html", s.funcMap)
		if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
			logger.Info("Error rendering v2 dashboard: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// User: render v1 dashboard (unchanged)
	data := TemplateData{
		Lang:    lang,
		Title:   i18n.T(lang, "dashboard.title"),
		Session: session,
		Stats:   stats,
	}

	if err := s.templates.ExecuteTemplate(w, "dashboard_user.html", data); err != nil {
		logger.Info("Error rendering dashboard template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleTrash displays the trash management page
func (s *Server) handleTrash(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Get user info
	user, err := users.GetByID(s.db, session.UserID)
	if err != nil {
		logger.Info("Error getting user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get user's shares
	userShares, err := shares.GetByUser(s.db, session.UserID)
	if err != nil {
		logger.Info("Error getting shares: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Collect all trash items from all user's shares
	type TrashItemWithShare struct {
		*trash.TrashItem
		ShareName string
	}

	var allTrashItems []TrashItemWithShare
	for _, share := range userShares {
		items, err := trash.ListTrashItems(share.Path, user.Username)
		if err != nil {
			logger.Info("Error listing trash for share %s: %v", share.Name, err)
			continue
		}

		for _, item := range items {
			allTrashItems = append(allTrashItems, TrashItemWithShare{
				TrashItem: item,
				ShareName: share.Name,
			})
		}
	}

	data := struct {
		Lang    string
		Title   string
		Session *auth.Session
		Items   []TrashItemWithShare
	}{
		Lang:    lang,
		Title:   i18n.T(lang, "trash.title"),
		Session: session,
		Items:   allTrashItems,
	}

	if err := s.templates.ExecuteTemplate(w, "trash.html", data); err != nil {
		logger.Info("Error rendering trash template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleTrashActions handles trash item actions (restore, delete)
func (s *Server) handleTrashActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse URL: /trash/{action}?share={shareName}&path={relPath}
	path := strings.TrimPrefix(r.URL.Path, "/trash/")
	action := path

	// Get parameters
	shareName := r.URL.Query().Get("share")
	relPath := r.URL.Query().Get("path")

	if shareName == "" || relPath == "" {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}

	// Get user
	user, err := users.GetByID(s.db, session.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Find the share
	userShares, err := shares.GetByUser(s.db, session.UserID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var targetShare *shares.Share
	for _, share := range userShares {
		if share.Name == shareName {
			targetShare = share
			break
		}
	}

	if targetShare == nil {
		http.Error(w, "Share not found", http.StatusNotFound)
		return
	}

	// Execute action
	switch action {
	case "restore":
		err = trash.RestoreItem(targetShare.Path, user.Username, relPath)
		if err != nil {
			logger.Info("Error restoring item: %v", err)
			http.Error(w, fmt.Sprintf("Failed to restore: %v", err), http.StatusInternalServerError)
			return
		}
		logger.Info("User %s restored file: %s from %s", user.Username, relPath, shareName)
		w.WriteHeader(http.StatusOK)

	case "delete":
		err = trash.DeleteItem(targetShare.Path, user.Username, relPath)
		if err != nil {
			logger.Info("Error deleting item: %v", err)
			http.Error(w, fmt.Sprintf("Failed to delete: %v", err), http.StatusInternalServerError)
			return
		}
		logger.Info("User %s permanently deleted file: %s from %s", user.Username, relPath, shareName)
		w.WriteHeader(http.StatusOK)

	case "empty":
		// Empty entire trash for this share
		err = trash.EmptyTrash(targetShare.Path, user.Username)
		if err != nil {
			logger.Info("Error emptying trash: %v", err)
			http.Error(w, fmt.Sprintf("Failed to empty trash: %v", err), http.StatusInternalServerError)
			return
		}
		logger.Info("User %s emptied trash for share %s", user.Username, shareName)
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Unknown action", http.StatusBadRequest)
	}
}

// handleAdminShares displays all shares for all users (admin only)
func (s *Server) handleAdminShares(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	lang := s.getLang(r)

	// Get all shares
	allShares, err := shares.GetAll(s.db)
	if err != nil {
		logger.Info("Error getting shares: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get SMB status
	smbStatus, _ := smb.GetServiceStatus()
	smbInstalled := smb.CheckSambaInstalled()

	data := struct {
		V2TemplateData
		Shares       []*shares.Share
		SMBStatus    string
		SMBInstalled bool
	}{
		V2TemplateData: V2TemplateData{
			Lang:       lang,
			Title:      i18n.T(lang, "shares.title"),
			ActivePage: "shares",
			Session:    session,
		},
		Shares:       allShares,
		SMBStatus:    smbStatus,
		SMBInstalled: smbInstalled,
	}

	tmpl := s.loadV2Page("v2_shares.html", s.funcMap)
	if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
		logger.Info("Error rendering shares template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleSyncShare triggers manual synchronization of a share to all enabled peers
func (s *Server) handleSyncShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract share ID from URL: /sync/share/{id}
	path := strings.TrimPrefix(r.URL.Path, "/sync/share/")
	shareID, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid share ID", http.StatusBadRequest)
		return
	}

	// Get share
	share, err := shares.GetByID(s.db, shareID)
	if err != nil {
		logger.Info("Error getting share: %v", err)
		http.Error(w, "Share not found", http.StatusNotFound)
		return
	}

	// Check if user has permission (either owner or admin)
	if !session.IsAdmin && share.UserID != session.UserID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Check if sync is enabled for this share
	if !share.SyncEnabled {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"success": false, "message": "Synchronisation non activée pour ce partage"}`)
		return
	}

	// Get all enabled peers
	allPeers, err := peers.GetAll(s.db)
	if err != nil {
		logger.Info("Error getting peers: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	enabledPeers := []*peers.Peer{}
	for _, peer := range allPeers {
		if peer.Enabled {
			enabledPeers = append(enabledPeers, peer)
		}
	}

	if len(enabledPeers) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"success": false, "message": "Aucun pair actif configuré"}`)
		return
	}

	// Get server name for manifest identification
	serverName, err := sync.GetServerName(s.db)
	if err != nil {
		logger.Info("Error getting server name: %v", err)
		http.Error(w, "Failed to get server name", http.StatusInternalServerError)
		return
	}

	// Synchronize to each enabled peer
	successCount := 0
	errorCount := 0
	var lastError string

	for _, peer := range enabledPeers {
		req := &sync.SyncRequest{
			ShareID:          shareID,
			PeerID:           peer.ID,
			UserID:           share.UserID,
			SharePath:        share.Path,
			PeerAddress:      peer.Address,
			PeerPort:         peer.Port,
			SourceServer:     serverName,
			PeerTimeoutHours: peer.SyncTimeoutHours,
		}

		// Use incremental sync (manifest-based)
		err := sync.SyncShareIncremental(s.db, req)
		if err != nil {
			errorCount++
			lastError = err.Error()
			logger.Info("Error syncing to peer %s: %v", peer.Name, err)
		} else {
			successCount++
			logger.Info("Successfully synced share %d to peer %s (incremental)", shareID, peer.Name)
		}
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if errorCount > 0 {
		fmt.Fprintf(w, `{"success": false, "message": "Synchronisation partielle: %d réussis, %d échecs. Dernière erreur: %s"}`,
			successCount, errorCount, lastError)
	} else {
		fmt.Fprintf(w, `{"success": true, "message": "Synchronisation réussie vers %d pair(s)"}`, successCount)
	}
}

// handleSettings shows the user settings page
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.GetSessionFromContext(r)

	// Get user from database
	user, err := users.GetByID(s.db, session.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	data := struct {
		Lang    string
		Title   string
		Session *auth.Session
		User    *users.User
		Success string
		Error   string
	}{
		Lang:    s.getLang(r),
		Title:   "Settings",
		Session: session,
		User:    user,
		Success: r.URL.Query().Get("success"),
		Error:   r.URL.Query().Get("error"),
	}

	if err := s.templates.ExecuteTemplate(w, "settings.html", data); err != nil {
		logger.Info("Error executing settings template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleSettingsLanguage handles language change
func (s *Server) handleSettingsLanguage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, _ := auth.GetSessionFromContext(r)

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/settings?error=Invalid+form+data", http.StatusSeeOther)
		return
	}

	language := r.FormValue("language")

	// Update user language in database
	if err := users.UpdateUserLanguage(s.db, session.UserID, language); err != nil {
		logger.Info("Error updating user language: %v", err)
		http.Redirect(w, r, "/settings?error=Failed+to+update+language", http.StatusSeeOther)
		return
	}

	// Redirect back to settings with success message
	successMsg := "Language changed successfully"
	if language == "fr" {
		successMsg = "Langue modifiée avec succès"
	}
	http.Redirect(w, r, "/settings?success="+successMsg, http.StatusSeeOther)
}

// handleSettingsPassword handles password change
func (s *Server) handleSettingsPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, _ := auth.GetSessionFromContext(r)

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/settings?error=Invalid+form+data", http.StatusSeeOther)
		return
	}

	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate inputs
	if currentPassword == "" || newPassword == "" || confirmPassword == "" {
		http.Redirect(w, r, "/settings?error=All+fields+are+required", http.StatusSeeOther)
		return
	}

	if newPassword != confirmPassword {
		http.Redirect(w, r, "/settings?error=New+passwords+do+not+match", http.StatusSeeOther)
		return
	}

	if len(newPassword) < 8 {
		http.Redirect(w, r, "/settings?error=Password+must+be+at+least+8+characters", http.StatusSeeOther)
		return
	}

	// Get master key
	var masterKey string
	err := s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		logger.Info("Error getting master key: %v", err)
		http.Redirect(w, r, "/settings?error=System+configuration+error", http.StatusSeeOther)
		return
	}

	// Change password (DB + SMB)
	if err := users.ChangePassword(s.db, session.UserID, currentPassword, newPassword, masterKey); err != nil {
		logger.Info("Error changing password for user %d: %v", session.UserID, err)

		// Check for specific error messages
		errMsg := err.Error()
		if errMsg == "incorrect current password" {
			http.Redirect(w, r, "/settings?error=Incorrect+current+password", http.StatusSeeOther)
		} else if errMsg == "new password must be at least 8 characters" {
			http.Redirect(w, r, "/settings?error=Password+must+be+at+least+8+characters", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/settings?error=Failed+to+change+password", http.StatusSeeOther)
		}
		return
	}

	// Get user to determine language for success message
	user, _ := users.GetByID(s.db, session.UserID)
	successMsg := "Password changed successfully"
	if user != nil && user.Language == "fr" {
		successMsg = "Mot de passe modifié avec succès"
	}

	http.Redirect(w, r, "/settings?success="+successMsg, http.StatusSeeOther)
}

// handleAdminUserQuota handles quota management for a user
func (s *Server) handleAdminUserQuota(w http.ResponseWriter, r *http.Request, userID int, session *auth.Session, lang string) {
	if r.Method == http.MethodGet {
		// Display quota edit form
		user, err := users.GetByID(s.db, userID)
		if err != nil {
			logger.Info("Error getting user: %v", err)
			http.NotFound(w, r)
			return
		}

		// Get quota info
		quotaInfo, err := quota.GetUserQuota(s.db, userID)
		if err != nil {
			logger.Info("Error getting quota info: %v", err)
			http.Error(w, "Failed to get quota information", http.StatusInternalServerError)
			return
		}

		data := struct {
			Lang      string
			Title     string
			Session   *auth.Session
			User      *users.User
			QuotaInfo *quota.QuotaInfo
		}{
			Lang:      lang,
			Title:     i18n.T(lang, "users.quota.title"),
			Session:   session,
			User:      user,
			QuotaInfo: quotaInfo,
		}

		if err := s.templates.ExecuteTemplate(w, "admin_users_quota.html", data); err != nil {
			logger.Info("Error rendering quota template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	} else if r.Method == http.MethodPost {
		// Update quotas
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		quotaBackupGB, err := strconv.Atoi(r.FormValue("quota_backup_gb"))
		if err != nil || quotaBackupGB < 0 {
			http.Error(w, "Invalid backup quota", http.StatusBadRequest)
			return
		}

		quotaDataGB, err := strconv.Atoi(r.FormValue("quota_data_gb"))
		if err != nil || quotaDataGB < 0 {
			http.Error(w, "Invalid data quota", http.StatusBadRequest)
			return
		}

		// Calculate total (backup + data)
		quotaTotalGB := quotaBackupGB + quotaDataGB

		// Update quota in database
		if err := quota.UpdateUserQuota(s.db, userID, quotaTotalGB, quotaBackupGB); err != nil {
			logger.Info("Error updating quota: %v", err)
			http.Error(w, "Failed to update quota", http.StatusInternalServerError)
			return
		}

		// Update Btrfs quotas for user shares
		user, err := users.GetByID(s.db, userID)
		if err == nil {
			backupPath := filepath.Join(s.cfg.SharesDir, user.Username, "backup")
			dataPath := filepath.Join(s.cfg.SharesDir, user.Username, "data")

			// Initialize quota manager
			qm, err := quota.NewQuotaManager(s.cfg.SharesDir)
			if err == nil {
				// Update backup quota
				if err := qm.UpdateQuota(backupPath, quotaBackupGB); err != nil {
					logger.Info("Warning: Failed to update Btrfs quota for backup: %v", err)
				} else {
					logger.Info("Updated Btrfs quota for %s: %dGB", backupPath, quotaBackupGB)
				}

				// Update data quota
				if err := qm.UpdateQuota(dataPath, quotaDataGB); err != nil {
					logger.Info("Warning: Failed to update Btrfs quota for data: %v", err)
				} else {
					logger.Info("Updated Btrfs quota for %s: %dGB", dataPath, quotaDataGB)
				}
			} else {
				logger.Info("Warning: Failed to initialize quota manager: %v", err)
			}
		}

		logger.Info("Admin %s updated quotas for user %d: backup=%dGB, data=%dGB, total=%dGB",
			session.Username, userID, quotaBackupGB, quotaDataGB, quotaTotalGB)

		// Redirect back to users page with success message
		http.Redirect(w, r, "/admin/users?success="+i18n.T(lang, "users.quota.success"), http.StatusSeeOther)

	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
