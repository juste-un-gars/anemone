// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"log"
	"net/http"
	"strconv"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/syncauth"
	"github.com/juste-un-gars/anemone/internal/sysconfig"
)

func (s *Server) handleAdminSettings(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Check if sync auth password is configured
	isConfigured, err := syncauth.IsConfigured(s.db)
	if err != nil {
		log.Printf("Error checking sync auth config: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Lang          string
		Session       *auth.Session
		IsConfigured  bool
		Success       string
		Error         string
	}{
		Lang:          lang,
		Session:       session,
		IsConfigured:  isConfigured,
	}

	if err := s.templates.ExecuteTemplate(w, "admin_settings.html", data); err != nil {
		log.Printf("Error rendering admin settings template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (s *Server) handleAdminSettingsSyncPassword(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/settings", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Parse form
	password := r.FormValue("password")
	passwordConfirm := r.FormValue("password_confirm")

	// Validate
	if password == "" || len(password) < 8 {
		isConfigured, _ := syncauth.IsConfigured(s.db)
		data := struct {
			Lang          string
			Session       *auth.Session
			IsConfigured  bool
			Success       string
			Error         string
		}{
			Lang:          lang,
			Session:       session,
			IsConfigured:  isConfigured,
			Error:         "Le mot de passe doit contenir au moins 8 caractères",
		}
		if err := s.templates.ExecuteTemplate(w, "admin_settings.html", data); err != nil {
			log.Printf("Error rendering admin settings template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	if password != passwordConfirm {
		isConfigured, _ := syncauth.IsConfigured(s.db)
		data := struct {
			Lang          string
			Session       *auth.Session
			IsConfigured  bool
			Success       string
			Error         string
		}{
			Lang:          lang,
			Session:       session,
			IsConfigured:  isConfigured,
			Error:         "Les mots de passe ne correspondent pas",
		}
		if err := s.templates.ExecuteTemplate(w, "admin_settings.html", data); err != nil {
			log.Printf("Error rendering admin settings template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Set password
	if err := syncauth.SetSyncAuthPassword(s.db, password); err != nil {
		log.Printf("Error setting sync auth password: %v", err)
		isConfigured, _ := syncauth.IsConfigured(s.db)
		data := struct {
			Lang          string
			Session       *auth.Session
			IsConfigured  bool
			Success       string
			Error         string
		}{
			Lang:          lang,
			Session:       session,
			IsConfigured:  isConfigured,
			Error:         "Erreur lors de la configuration du mot de passe",
		}
		if err := s.templates.ExecuteTemplate(w, "admin_settings.html", data); err != nil {
			log.Printf("Error rendering admin settings template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Success
	log.Printf("Admin %s configured sync auth password", session.Username)
	isConfigured, _ := syncauth.IsConfigured(s.db)
	data := struct {
		Lang          string
		Session       *auth.Session
		IsConfigured  bool
		Success       string
		Error         string
	}{
		Lang:          lang,
		Session:       session,
		IsConfigured:  isConfigured,
		Success:       "Mot de passe de synchronisation configuré avec succès",
	}

	if err := s.templates.ExecuteTemplate(w, "admin_settings.html", data); err != nil {
		log.Printf("Error rendering admin settings template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (s *Server) handleAdminSettingsTrash(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// GET: Display settings
	if r.Method == http.MethodGet {
		// Get current retention days
		retentionDays, err := sysconfig.GetTrashRetentionDays(s.db)
		if err != nil {
			log.Printf("Error getting trash retention days: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		data := struct {
			Lang           string
			Session        *auth.Session
			RetentionDays  int
			Success        string
			Error          string
		}{
			Lang:           lang,
			Session:        session,
			RetentionDays:  retentionDays,
		}

		if err := s.templates.ExecuteTemplate(w, "admin_settings_trash.html", data); err != nil {
			log.Printf("Error rendering admin settings trash template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// POST: Update settings
	if r.Method == http.MethodPost {
		// Parse form
		retentionDaysStr := r.FormValue("retention_days")

		// Parse days
		retentionDays, err := strconv.Atoi(retentionDaysStr)
		if err != nil || retentionDays < 0 {
			currentRetentionDays, _ := sysconfig.GetTrashRetentionDays(s.db)
			data := struct {
				Lang           string
				Session        *auth.Session
				RetentionDays  int
				Success        string
				Error          string
			}{
				Lang:           lang,
				Session:        session,
				RetentionDays:  currentRetentionDays,
				Error:          "La durée de rétention doit être un nombre positif",
			}
			if err := s.templates.ExecuteTemplate(w, "admin_settings_trash.html", data); err != nil {
				log.Printf("Error rendering admin settings trash template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		// Update retention days
		if err := sysconfig.SetTrashRetentionDays(s.db, retentionDays); err != nil {
			log.Printf("Error setting trash retention days: %v", err)
			currentRetentionDays, _ := sysconfig.GetTrashRetentionDays(s.db)
			data := struct {
				Lang           string
				Session        *auth.Session
				RetentionDays  int
				Success        string
				Error          string
			}{
				Lang:           lang,
				Session:        session,
				RetentionDays:  currentRetentionDays,
				Error:          "Erreur lors de la mise à jour de la durée de rétention",
			}
			if err := s.templates.ExecuteTemplate(w, "admin_settings_trash.html", data); err != nil {
				log.Printf("Error rendering admin settings trash template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		// Success
		log.Printf("Admin %s updated trash retention days to %d", session.Username, retentionDays)
		data := struct {
			Lang           string
			Session        *auth.Session
			RetentionDays  int
			Success        string
			Error          string
		}{
			Lang:           lang,
			Session:        session,
			RetentionDays:  retentionDays,
			Success:        "Durée de rétention mise à jour avec succès",
		}

		if err := s.templates.ExecuteTemplate(w, "admin_settings_trash.html", data); err != nil {
			log.Printf("Error rendering admin settings trash template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

