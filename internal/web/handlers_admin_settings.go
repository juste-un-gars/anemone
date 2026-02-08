// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"net/http"
	"strconv"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/logger"
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
		logger.Error("Error checking sync auth config", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	s.renderSettingsPage(w, session, lang, isConfigured, "", "")
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
		s.renderSettingsPage(w, session, lang, isConfigured, "", "Le mot de passe doit contenir au moins 8 caractères")
		return
	}

	if password != passwordConfirm {
		isConfigured, _ := syncauth.IsConfigured(s.db)
		s.renderSettingsPage(w, session, lang, isConfigured, "", "Les mots de passe ne correspondent pas")
		return
	}

	// Set password
	if err := syncauth.SetSyncAuthPassword(s.db, password); err != nil {
		logger.Error("Error setting sync auth password", "error", err)
		isConfigured, _ := syncauth.IsConfigured(s.db)
		s.renderSettingsPage(w, session, lang, isConfigured, "", "Erreur lors de la configuration du mot de passe")
		return
	}

	// Success
	logger.Info("Admin configured sync auth password", "admin", session.Username)
	isConfigured, _ := syncauth.IsConfigured(s.db)
	s.renderSettingsPage(w, session, lang, isConfigured, "Mot de passe de synchronisation configuré avec succès", "")
}

// renderSettingsPage renders the v2 settings page with optional messages.
func (s *Server) renderSettingsPage(w http.ResponseWriter, session *auth.Session, lang string, isConfigured bool, success, errMsg string) {
	data := struct {
		V2TemplateData
		IsConfigured bool
		Success      string
		Error        string
	}{
		V2TemplateData: V2TemplateData{
			Lang:       lang,
			Title:      "Sync Security",
			ActivePage: "settings",
			Session:    session,
		},
		IsConfigured: isConfigured,
		Success:      success,
		Error:        errMsg,
	}

	tmpl := s.loadV2Page("v2_settings.html", s.funcMap)
	if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
		logger.Error("Error rendering admin settings template", "error", err)
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
			logger.Error("Error getting trash retention days", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		s.renderTrashPage(w, session, lang, retentionDays, "", "")
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
			s.renderTrashPage(w, session, lang, currentRetentionDays, "", "La durée de rétention doit être un nombre positif")
			return
		}

		// Update retention days
		if err := sysconfig.SetTrashRetentionDays(s.db, retentionDays); err != nil {
			logger.Error("Error setting trash retention days", "error", err)
			currentRetentionDays, _ := sysconfig.GetTrashRetentionDays(s.db)
			s.renderTrashPage(w, session, lang, currentRetentionDays, "", "Erreur lors de la mise à jour de la durée de rétention")
			return
		}

		// Success
		logger.Info("Admin updated trash retention days", "admin", session.Username, "days", retentionDays)
		s.renderTrashPage(w, session, lang, retentionDays, "Durée de rétention mise à jour avec succès", "")
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// renderTrashPage renders the v2 trash settings page with optional messages.
func (s *Server) renderTrashPage(w http.ResponseWriter, session *auth.Session, lang string, retentionDays int, success, errMsg string) {
	data := struct {
		V2TemplateData
		RetentionDays int
		Success       string
		Error         string
	}{
		V2TemplateData: V2TemplateData{
			Lang:       lang,
			Title:      "Trash Settings",
			ActivePage: "trash",
			Session:    session,
		},
		RetentionDays: retentionDays,
		Success:       success,
		Error:         errMsg,
	}

	tmpl := s.loadV2Page("v2_settings_trash.html", s.funcMap)
	if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
		logger.Error("Error rendering admin settings trash template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
