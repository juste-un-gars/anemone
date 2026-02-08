// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package web provides HTTP handlers for the Anemone web interface.
//
// This file contains core storage page handlers and utilities.
// For ZFS pool/dataset/snapshot handlers, see handlers_admin_storage_zfs.go.
// For disk handlers (format, mount, wipe), see handlers_admin_storage_disk.go.
package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/juste-un-gars/anemone/internal/adminverify"
	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/storage"
)

func (s *Server) handleAdminStorage(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	lang := s.getLang(r)

	// Get storage overview
	overview, err := storage.GetStorageOverview()
	if err != nil {
		logger.Info("Error getting storage overview: %v", err)
		// Continue with empty overview
		overview = &storage.StorageOverview{}
	}

	data := struct {
		V2TemplateData
		Overview       *storage.StorageOverview
		Disks          []storage.Disk
		Pools          []storage.ZFSPool
		SMARTAvailable bool
		ZFSAvailable   bool
	}{
		V2TemplateData: V2TemplateData{
			Lang:       lang,
			Title:      i18n.T(lang, "storage_management"),
			ActivePage: "storage",
			Session:    session,
		},
		Overview:       overview,
		Disks:          overview.Disks,
		Pools:          overview.Pools,
		SMARTAvailable: overview.SMARTAvailable,
		ZFSAvailable:   overview.ZFSAvailable,
	}

	tmpl := s.loadV2Page("v2_storage.html", s.funcMap)
	if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
		logger.Error("Error rendering storage template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleAdminStorageAPI provides JSON API for storage data
func (s *Server) handleAdminStorageAPI(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	overview, err := storage.GetStorageOverview()
	if err != nil {
		logger.Info("Error getting storage overview: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(overview)
}

// handleAdminStoragePoolScrub handles starting/stopping ZFS pool scrub
func (s *Server) handleAdminStoragePoolScrub(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract pool name from URL path: /api/admin/storage/pool/{name}/scrub
	path := r.URL.Path
	parts := splitPath(path)
	if len(parts) < 6 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	poolName := parts[4] // /api/admin/storage/pool/{name}/scrub

	// Check action parameter (start or stop)
	action := r.URL.Query().Get("action")
	if action == "" {
		action = "start"
	}

	var err error
	if action == "stop" {
		err = storage.StopScrub(poolName)
	} else {
		err = storage.StartScrub(poolName)
	}

	if err != nil {
		logger.Info("Error %sing scrub on pool %s: %v", action, poolName, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"action":  action,
		"pool":    poolName,
	})
}

// handleAdminStorageDiskSMART returns detailed SMART info for a disk
func (s *Server) handleAdminStorageDiskSMART(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract disk name from URL path: /api/admin/storage/disk/{name}/smart
	path := r.URL.Path
	parts := splitPath(path)
	if len(parts) < 6 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	diskName := parts[4] // /api/admin/storage/disk/{name}/smart

	// Build device path
	devicePath := "/dev/" + diskName

	smartInfo, err := storage.GetSMARTInfo(devicePath)
	if err != nil {
		logger.Info("Error getting SMART info for %s: %v", devicePath, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(smartInfo)
}

// splitPath splits URL path into parts
func splitPath(path string) []string {
	var parts []string
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// getClientIP extracts client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for reverse proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	parts := strings.Split(r.RemoteAddr, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return r.RemoteAddr
}

// handleAdminVerifyPassword verifies admin password and returns a token
func (s *Server) handleAdminVerifyPassword(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	ip := getClientIP(r)
	verifier := adminverify.GetVerifier()

	token, err := verifier.VerifyPassword(s.db, session.UserID, req.Password, ip)
	if err != nil {
		logger.Info("Password verification failed for user %d from %s: %v", session.UserID, ip, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":              err.Error(),
			"remaining_attempts": verifier.GetRemainingAttempts(ip),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   token,
	})
}

// validateVerificationToken validates a verification token from request
func (s *Server) validateVerificationToken(r *http.Request, session *auth.Session) error {
	token := r.Header.Get("X-Verification-Token")
	if token == "" {
		// Also check in JSON body
		return adminverify.GetVerifier().ValidateToken("", session.UserID)
	}
	return adminverify.GetVerifier().ValidateToken(token, session.UserID)
}
