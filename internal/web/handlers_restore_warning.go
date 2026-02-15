// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// This file contains handlers for the restore warning page displayed after server restoration.

package web

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/bulkrestore"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/peers"
)

func (s *Server) handleRestoreWarning(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Get restore date from system_config
	var restoreDate string
	err := s.db.QueryRow("SELECT value FROM system_config WHERE key = 'server_restored_at'").Scan(&restoreDate)
	if err != nil {
		restoreDate = "Unknown"
	}

	// Get available backups from all peers for this user
	type BackupInfo struct {
		PeerID       int
		PeerName     string
		ShareName    string
		FileCount    int
		TotalSize    string
		LastModified string
	}

	var availableBackups []BackupInfo

	// Get all peers
	peersList, err := peers.GetAll(s.db)
	if err == nil {
		// Get master key for password decryption
		var masterKey string
		if err := s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey); err != nil {
			logger.Info("Error getting master key", "error", err)
		} else {
			client := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
				Timeout: 10 * time.Second,
			}

			for _, peer := range peersList {
				// Query peer for user's backups
				url := fmt.Sprintf("https://%s:%d/api/sync/list-user-backups?user_id=%d", peer.Address, peer.Port, session.UserID)

				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					continue
				}

				// Decrypt and add P2P authentication
				if peer.Password != nil && len(*peer.Password) > 0 {
					peerPassword, err := peers.DecryptPeerPassword(peer.Password, masterKey)
					if err != nil {
						logger.Info("Error decrypting peer password", "error", err)
						continue
					}
					req.Header.Set("X-Sync-Password", peerPassword)
				}

				resp, err := client.Do(req)
				if err != nil {
					continue
				}

				if resp.StatusCode == http.StatusOK {
					var backups []struct {
						ShareName    string `json:"share_name"`
						FileCount    int    `json:"file_count"`
						TotalSize    int64  `json:"total_size"`
						LastModified string `json:"last_modified"`
					}

					if err := json.NewDecoder(resp.Body).Decode(&backups); err == nil {
						for _, b := range backups {
							availableBackups = append(availableBackups, BackupInfo{
								PeerID:       peer.ID,
								PeerName:     peer.Name,
								ShareName:    b.ShareName,
								FileCount:    b.FileCount,
								TotalSize:    formatBytes(b.TotalSize),
								LastModified: b.LastModified,
							})
						}
					}
				}
				resp.Body.Close()
			}
		}
	}

	data := struct {
		Lang             string
		Title            string
		Session          *auth.Session
		RestoreDate      string
		AvailableBackups []BackupInfo
	}{
		Lang:             lang,
		Title:            "Server Restored",
		Session:          session,
		RestoreDate:      restoreDate,
		AvailableBackups: availableBackups,
	}

	if err := s.templates.ExecuteTemplate(w, "restore_warning.html", data); err != nil {
		logger.Info("Error rendering restore warning template", "error", err)
	}
}

// handleRestoreWarningAcknowledge marks the restore as acknowledged for the user
func (s *Server) handleRestoreWarningAcknowledge(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Update user's restore_acknowledged flag
	_, err := s.db.Exec("UPDATE users SET restore_acknowledged = 1 WHERE id = ?", session.UserID)
	if err != nil {
		logger.Info("Error updating restore_acknowledged", "error", err)
		http.Error(w, "Failed to acknowledge restore", http.StatusInternalServerError)
		return
	}

	logger.Info("User acknowledged server restore (manual restore)", "username", session.Username)

	// Redirect to dashboard
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// handleRestoreWarningBulk handles automatic bulk restore from a peer
func (s *Server) handleRestoreWarningBulk(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get peer ID, share name, and source server from form
	peerIDStr := r.FormValue("peer_id")
	shareName := r.FormValue("share_name")
	sourceServer := r.FormValue("source_server")

	if peerIDStr == "" || shareName == "" || sourceServer == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Missing peer_id, share_name, or source_server",
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

	logger.Info("User starting bulk restore from peer share source", "username", session.Username, "peer_id", peerID, "share_name", shareName, "source_server", sourceServer)

	// Start bulk restore in background
	go func() {
		// Note: We can't use progressChan in a simple HTTP request/response
		// For now, we'll just do the restore and mark as complete
		err := bulkrestore.BulkRestoreFromPeer(s.db, session.UserID, peerID, shareName, sourceServer, s.cfg.DataDir, nil)
		if err != nil {
			logger.Info("Bulk restore failed for user", "username", session.Username, "error", err)
		} else {
			// Mark restore as completed
			s.db.Exec("UPDATE users SET restore_acknowledged = 1, restore_completed = 1 WHERE id = ?", session.UserID)
			logger.Info("Bulk restore completed successfully for user", "username", session.Username)
		}
	}()

	// Return immediate response (restore runs in background)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Bulk restore started in background",
	})
}
