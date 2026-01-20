// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/i18n"
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
		log.Printf("Error getting storage overview: %v", err)
		// Continue with empty overview
		overview = &storage.StorageOverview{}
	}

	data := map[string]interface{}{
		"Title":          i18n.T(lang, "storage_management"),
		"Session":        session,
		"Overview":       overview,
		"Disks":          overview.Disks,
		"Pools":          overview.Pools,
		"SMARTAvailable": overview.SMARTAvailable,
		"ZFSAvailable":   overview.ZFSAvailable,
		"Lang":           lang,
		"ActivePage":     "storage",
	}

	if err := s.templates.ExecuteTemplate(w, "admin_storage.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
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
		log.Printf("Error getting storage overview: %v", err)
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
		log.Printf("Error %sing scrub on pool %s: %v", action, poolName, err)
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
		log.Printf("Error getting SMART info for %s: %v", devicePath, err)
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
