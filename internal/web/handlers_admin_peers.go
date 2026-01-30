// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"database/sql"
	"fmt"
	"github.com/juste-un-gars/anemone/internal/logger"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/peers"
	"github.com/juste-un-gars/anemone/internal/sync"
)

func (s *Server) handleAdminPeers(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	lang := s.getLang(r)

	// Get all peers
	peersList, err := peers.GetAll(s.db)
	if err != nil {
		logger.Info("Error getting peers: %v", err)
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
		logger.Info("Error getting recent syncs: %v", err)
		// Continue with empty list
	}

	var recentSyncs []RecentSync
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var rs RecentSync
			var startedAtStr, completedAtStr sql.NullString
			if err := rows.Scan(&rs.Username, &rs.PeerName, &startedAtStr, &completedAtStr, &rs.Status, &rs.FilesSynced, &rs.BytesSynced); err != nil {
				logger.Info("Error scanning sync log: %v", err)
				continue
			}
			// Parse SQLite datetime strings (try multiple formats)
			if startedAtStr.Valid {
				rs.StartedAt = parseSQLiteDateTime(startedAtStr.String)
			}
			if completedAtStr.Valid {
				t := parseSQLiteDateTime(completedAtStr.String)
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

	// Check for running syncs for each peer
	runningSyncs := make(map[int]bool)
	for _, peer := range peersList {
		hasRunning, err := sync.HasRunningSyncForPeer(s.db, peer.ID)
		if err != nil {
			logger.Info("Error checking running sync for peer %d: %v", peer.ID, err)
			continue
		}
		if hasRunning {
			runningSyncs[peer.ID] = true
		}
	}

	// Get success/error messages from query params
	successMsg := r.URL.Query().Get("success")
	errorMsg := r.URL.Query().Get("error")

	data := struct {
		Lang         string
		Title        string
		Session      *auth.Session
		Peers        []*peers.Peer
		RecentSyncs  []RecentSync
		RunningSyncs map[int]bool
		Success      string
		Error        string
	}{
		Lang:         lang,
		Title:        i18n.T(lang, "peers.title"),
		Session:      session,
		Peers:        peersList,
		RecentSyncs:  recentSyncs,
		RunningSyncs: runningSyncs,
		Success:      successMsg,
		Error:        errorMsg,
	}

	if err := s.templates.ExecuteTemplate(w, "admin_peers.html", data); err != nil {
		logger.Info("Error rendering peers template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleAdminPeersAdd(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	lang := s.getLang(r)

	if r.Method == http.MethodGet {
		// Show add peer form
		data := struct {
			Lang    string
			Title   string
			Session *auth.Session
			Error   string
		}{
			Lang:    lang,
			Title:   i18n.T(lang, "peers.add.title"),
			Session: session,
		}

		if err := s.templates.ExecuteTemplate(w, "admin_peers_add.html", data); err != nil {
			logger.Info("Error rendering peers add template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		return
	}

	if r.Method == http.MethodPost {
		// Parse form
		name := r.FormValue("name")
		address := r.FormValue("address")
		portStr := r.FormValue("port")
		publicKey := r.FormValue("public_key")
		password := r.FormValue("password")
		enabled := r.FormValue("enabled") == "on"

		// Parse sync configuration
		syncEnabled := r.FormValue("sync_enabled") == "on"
		syncFrequency := r.FormValue("sync_frequency")
		syncTime := r.FormValue("sync_time")
		syncDayOfWeekStr := r.FormValue("sync_day_of_week")
		syncDayOfMonthStr := r.FormValue("sync_day_of_month")

		// Validate
		if name == "" || address == "" {
			data := struct {
				Lang    string
				Title   string
				Session *auth.Session
				Error   string
			}{
				Lang:    lang,
				Title:   i18n.T(lang, "peers.add.title"),
				Session: session,
				Error:   "Le nom et l'adresse sont requis",
			}
			if err := s.templates.ExecuteTemplate(w, "admin_peers_add.html", data); err != nil {
				logger.Info("Error rendering peers add template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		// Parse port
		port := 8443
		if portStr != "" {
			var err error
			port, err = strconv.Atoi(portStr)
			if err != nil || port < 1 || port > 65535 {
				data := struct {
					Lang    string
					Title   string
					Session *auth.Session
					Error   string
				}{
					Lang:    lang,
					Title:   i18n.T(lang, "peers.add.title"),
					Session: session,
					Error:   "Port invalide",
				}
				if err := s.templates.ExecuteTemplate(w, "admin_peers_add.html", data); err != nil {
					logger.Info("Error rendering peers add template: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
				return
			}
		}

		// Parse sync day of week/month
		var syncDayOfWeekPtr *int
		if syncDayOfWeekStr != "" && syncFrequency == "weekly" {
			dayOfWeek, err := strconv.Atoi(syncDayOfWeekStr)
			if err == nil && dayOfWeek >= 0 && dayOfWeek <= 6 {
				syncDayOfWeekPtr = &dayOfWeek
			}
		}
		var syncDayOfMonthPtr *int
		if syncDayOfMonthStr != "" && syncFrequency == "monthly" {
			dayOfMonth, err := strconv.Atoi(syncDayOfMonthStr)
			if err == nil && dayOfMonth >= 1 && dayOfMonth <= 31 {
				syncDayOfMonthPtr = &dayOfMonth
			}
		}

		// Parse sync interval (convert to minutes)
		syncIntervalMinutes := 60 // Default: 1 hour
		if syncFrequency == "interval" {
			intervalValueStr := r.FormValue("sync_interval_value")
			intervalUnit := r.FormValue("sync_interval_unit")

			if intervalValueStr != "" {
				intervalValue, err := strconv.Atoi(intervalValueStr)
				if err == nil && intervalValue > 0 {
					if intervalUnit == "hours" {
						syncIntervalMinutes = intervalValue * 60
					} else {
						syncIntervalMinutes = intervalValue
					}
				}
			}
		}

		// Parse sync timeout
		syncTimeoutHours := 2 // Default: 2 hours
		syncTimeoutHoursStr := r.FormValue("sync_timeout_hours")
		if syncTimeoutHoursStr != "" {
			timeout, err := strconv.Atoi(syncTimeoutHoursStr)
			if err == nil && timeout >= 0 && timeout <= 72 {
				syncTimeoutHours = timeout
			}
		}

		// Get master key for password encryption
		var masterKey string
		if err := s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey); err != nil {
			logger.Info("Error getting master key: %v", err)
			data := struct {
				Lang    string
				Title   string
				Session *auth.Session
				Error   string
			}{
				Lang:    lang,
				Title:   i18n.T(lang, "peers.add.title"),
				Session: session,
				Error:   "Erreur système",
			}
			if err := s.templates.ExecuteTemplate(w, "admin_peers_add.html", data); err != nil {
				logger.Info("Error rendering peers add template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		// Create peer
		var pkPtr *string
		if publicKey != "" {
			pkPtr = &publicKey
		}
		// Encrypt peer password before storing
		var pwPtr *[]byte
		if password != "" {
			encrypted, err := peers.EncryptPeerPassword(password, masterKey)
			if err != nil {
				logger.Info("Error encrypting peer password: %v", err)
				data := struct {
					Lang    string
					Title   string
					Session *auth.Session
					Error   string
				}{
					Lang:    lang,
					Title:   i18n.T(lang, "peers.add.title"),
					Session: session,
					Error:   "Erreur lors du chiffrement du mot de passe",
				}
				if err := s.templates.ExecuteTemplate(w, "admin_peers_add.html", data); err != nil {
					logger.Info("Error rendering peers add template: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
				return
			}
			pwPtr = encrypted
		}
		peer := &peers.Peer{
			Name:                name,
			Address:             address,
			Port:                port,
			PublicKey:           pkPtr,
			Password:            pwPtr,
			Enabled:             enabled,
			Status:              "unknown",
			SyncEnabled:         syncEnabled,
			SyncFrequency:       syncFrequency,
			SyncTime:            syncTime,
			SyncDayOfWeek:       syncDayOfWeekPtr,
			SyncDayOfMonth:      syncDayOfMonthPtr,
			SyncIntervalMinutes: syncIntervalMinutes,
			SyncTimeoutHours:    syncTimeoutHours,
		}

		if err := peers.Create(s.db, peer); err != nil {
			logger.Info("Error creating peer: %v", err)
			data := struct {
				Lang    string
				Title   string
				Session *auth.Session
				Error   string
			}{
				Lang:    lang,
				Title:   i18n.T(lang, "peers.add.title"),
				Session: session,
				Error:   fmt.Sprintf("Erreur lors de la création du pair: %v", err),
			}
			if err := s.templates.ExecuteTemplate(w, "admin_peers_add.html", data); err != nil {
				logger.Info("Error rendering peers add template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		logger.Info("Created peer: %s (ID: %d)", peer.Name, peer.ID)
		http.Redirect(w, r, "/admin/peers", http.StatusSeeOther)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleAdminPeersActions(w http.ResponseWriter, r *http.Request) {
	// Extract peer ID and action from URL
	// URL format: /admin/peers/{id}/{action}
	path := strings.TrimPrefix(r.URL.Path, "/admin/peers/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	peerID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "Invalid peer ID", http.StatusBadRequest)
		return
	}

	action := parts[1]

	switch action {
	case "edit":
		// Display edit form (GET)
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		session, ok := auth.GetSessionFromContext(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		peer, err := peers.GetByID(s.db, peerID)
		if err != nil {
			logger.Info("Error getting peer: %v", err)
			http.Error(w, "Peer not found", http.StatusNotFound)
			return
		}

		data := struct {
			Lang    string
			Session *auth.Session
			Peer    *peers.Peer
			Error   string
		}{
			Lang:    s.cfg.Language,
			Session: session,
			Peer:    peer,
			Error:   r.URL.Query().Get("error"),
		}

		if err := s.templates.ExecuteTemplate(w, "admin_peers_edit.html", data); err != nil {
			logger.Info("Template error: %v", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
		return

	case "update":
		// Process edit form (POST)
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		session, ok := auth.GetSessionFromContext(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, fmt.Sprintf("/admin/peers/%d/edit?error=Invalid+form+data", peerID), http.StatusSeeOther)
			return
		}

		// Get existing peer
		peer, err := peers.GetByID(s.db, peerID)
		if err != nil {
			logger.Info("Error getting peer: %v", err)
			http.Redirect(w, r, "/admin/peers?error=Peer+not+found", http.StatusSeeOther)
			return
		}

		// Get master key for password encryption
		var masterKey string
		if err := s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey); err != nil {
			logger.Info("Error getting master key: %v", err)
			http.Redirect(w, r, fmt.Sprintf("/admin/peers/%d/edit?error=System+configuration+error", peerID), http.StatusSeeOther)
			return
		}

		// Update fields
		peer.Name = r.FormValue("name")
		peer.Address = r.FormValue("address")

		port, err := strconv.Atoi(r.FormValue("port"))
		if err != nil || port < 1 || port > 65535 {
			http.Redirect(w, r, fmt.Sprintf("/admin/peers/%d/edit?error=Invalid+port", peerID), http.StatusSeeOther)
			return
		}
		peer.Port = port

		// Update password if provided
		password := r.FormValue("password")
		if password != "" {
			// Encrypt new password before storing
			encrypted, err := peers.EncryptPeerPassword(password, masterKey)
			if err != nil {
				logger.Info("Error encrypting peer password: %v", err)
				http.Redirect(w, r, fmt.Sprintf("/admin/peers/%d/edit?error=Failed+to+encrypt+password", peerID), http.StatusSeeOther)
				return
			}
			peer.Password = encrypted
		} else if r.FormValue("clear_password") == "1" {
			peer.Password = nil
		}
		// If password is empty and clear_password is not checked, keep existing password (already encrypted)

		// Always keep peer enabled (the only control is sync_enabled for automatic sync)
		peer.Enabled = true

		// Update sync configuration
		peer.SyncEnabled = r.FormValue("sync_enabled") == "on"
		peer.SyncFrequency = r.FormValue("sync_frequency")
		peer.SyncTime = r.FormValue("sync_time")

		// Parse sync day of week/month
		syncDayOfWeekStr := r.FormValue("sync_day_of_week")
		syncDayOfMonthStr := r.FormValue("sync_day_of_month")

		var syncDayOfWeekPtr *int
		if syncDayOfWeekStr != "" && peer.SyncFrequency == "weekly" {
			dayOfWeek, err := strconv.Atoi(syncDayOfWeekStr)
			if err == nil && dayOfWeek >= 0 && dayOfWeek <= 6 {
				syncDayOfWeekPtr = &dayOfWeek
			}
		}
		peer.SyncDayOfWeek = syncDayOfWeekPtr

		var syncDayOfMonthPtr *int
		if syncDayOfMonthStr != "" && peer.SyncFrequency == "monthly" {
			dayOfMonth, err := strconv.Atoi(syncDayOfMonthStr)
			if err == nil && dayOfMonth >= 1 && dayOfMonth <= 31 {
				syncDayOfMonthPtr = &dayOfMonth
			}
		}
		peer.SyncDayOfMonth = syncDayOfMonthPtr

		// Parse sync interval (convert to minutes)
		if peer.SyncFrequency == "interval" {
			intervalValueStr := r.FormValue("sync_interval_value")
			intervalUnit := r.FormValue("sync_interval_unit")

			if intervalValueStr != "" {
				intervalValue, err := strconv.Atoi(intervalValueStr)
				if err == nil && intervalValue > 0 {
					if intervalUnit == "hours" {
						peer.SyncIntervalMinutes = intervalValue * 60
					} else {
						peer.SyncIntervalMinutes = intervalValue
					}
				} else {
					peer.SyncIntervalMinutes = 60 // Default: 1 hour
				}
			} else {
				peer.SyncIntervalMinutes = 60 // Default: 1 hour
			}
		}

		// Parse sync timeout
		syncTimeoutHoursStr := r.FormValue("sync_timeout_hours")
		if syncTimeoutHoursStr != "" {
			timeout, err := strconv.Atoi(syncTimeoutHoursStr)
			if err == nil && timeout >= 0 && timeout <= 72 {
				peer.SyncTimeoutHours = timeout
			} else {
				peer.SyncTimeoutHours = 2 // Default: 2 hours
			}
		} else {
			peer.SyncTimeoutHours = 2 // Default: 2 hours
		}

		// Save to database
		if err := peers.Update(s.db, peer); err != nil {
			logger.Info("Error updating peer: %v", err)
			http.Redirect(w, r, fmt.Sprintf("/admin/peers/%d/edit?error=Failed+to+update+peer", peerID), http.StatusSeeOther)
			return
		}

		logger.Info("Admin %s updated peer ID %d: %s", session.Username, peerID, peer.Name)
		http.Redirect(w, r, "/admin/peers", http.StatusSeeOther)
		return

	case "delete":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := peers.Delete(s.db, peerID); err != nil {
			logger.Info("Error deleting peer: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		logger.Info("Deleted peer ID: %d", peerID)
		w.WriteHeader(http.StatusOK)
		return

	case "test":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		peer, err := peers.GetByID(s.db, peerID)
		if err != nil {
			logger.Info("Error getting peer: %v", err)
			http.Error(w, "Peer not found", http.StatusNotFound)
			return
		}

		// Get master key for password decryption
		var masterKey string
		if err := s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey); err != nil {
			logger.Info("Error getting master key: %v", err)
			http.Error(w, "System configuration error", http.StatusInternalServerError)
			return
		}

		online, err := peers.TestConnection(peer, masterKey)
		if err != nil {
			logger.Info("Error testing peer connection: %v", err)
		}

		status := "offline"
		if online {
			status = "online"
		}

		// Update peer status
		if err := peers.UpdateStatus(s.db, peerID, status); err != nil {
			logger.Info("Error updating peer status: %v", err)
		}

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		if online {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status": "online", "message": "Connexion réussie"}`)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status": "offline", "message": "Impossible de se connecter"}`)
		}
		return

	default:
		http.Error(w, "Unknown action", http.StatusBadRequest)
		return
	}
}

// parseSQLiteDateTime parses a datetime string from SQLite, trying multiple formats.
// SQLite can return datetimes in various formats depending on how they were stored.
func parseSQLiteDateTime(s string) time.Time {
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
	logger.Info("Warning: Could not parse datetime string: %q", s)
	return time.Time{}
}
