// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"encoding/json"
	"log"
	"net/http"

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
