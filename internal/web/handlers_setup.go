// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/syncauth"
	"github.com/juste-un-gars/anemone/internal/users"
)

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	// Redirect if already configured
	if s.isSetupCompleted() {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	if r.Method == http.MethodGet {
		// Show setup form
		data := TemplateData{
			Lang:      lang,
			Title:     i18n.T(lang, "setup.title"),
			CSRFToken: auth.GetCSRFFromRequest(r),
		}

		if err := s.templates.ExecuteTemplate(w, "setup.html", data); err != nil {
			logger.Info("Error rendering setup template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	} else if r.Method == http.MethodPost {
		// Validate CSRF token
		if !auth.ValidateCSRF(r) {
			logger.Warn("CSRF validation failed on setup")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Process setup form
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		nasName := r.FormValue("nas_name")
		timezone := r.FormValue("timezone")
		language := r.FormValue("language")
		username := r.FormValue("username")
		password := r.FormValue("password")
		passwordConfirm := r.FormValue("password_confirm")
		email := r.FormValue("email")

		// Validate
		if nasName == "" || timezone == "" || username == "" || password == "" {
			http.Error(w, i18n.T(lang, "setup.errors.required"), http.StatusBadRequest)
			return
		}

		if password != passwordConfirm {
			http.Error(w, i18n.T(lang, "setup.errors.password_mismatch"), http.StatusBadRequest)
			return
		}

		if len(password) < 8 {
			http.Error(w, i18n.T(lang, "setup.errors.password_length"), http.StatusBadRequest)
			return
		}

		// Generate master key for encrypting user keys
		masterKeyBytes := make([]byte, 32)
		if _, err := rand.Read(masterKeyBytes); err != nil {
			logger.Info("Error generating master key", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		masterKey := base64.StdEncoding.EncodeToString(masterKeyBytes)

		// Create first admin user with language preference
		user, encryptionKey, err := users.CreateFirstAdmin(s.db, username, password, email, masterKey, language)
		if err != nil {
			logger.Info("Error creating admin user", "error", err)
			http.Error(w, "Failed to create admin user", http.StatusInternalServerError)
			return
		}

		logger.Info("Created admin user", "username", user.Username, "id", user.ID)

		// Save system configuration
		tx, err := s.db.Begin()
		if err != nil {
			logger.Info("Error starting transaction", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		configs := map[string]string{
			"nas_name":   nasName,
			"timezone":   timezone,
			"language":   language,
			"master_key": masterKey,
		}

		for key, value := range configs {
			_, err = tx.Exec("INSERT OR REPLACE INTO system_config (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)", key, value)
			if err != nil {
				logger.Info("Error saving config", "key", key, "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		if err := tx.Commit(); err != nil {
			logger.Info("Error committing transaction", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Generate random sync authentication password (secure by default)
		syncPasswordBytes := make([]byte, 24) // 24 bytes = 192 bits
		if _, err := rand.Read(syncPasswordBytes); err != nil {
			logger.Info("Error generating sync password", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		// Use base64 URL encoding for password-safe characters
		syncPassword := base64.URLEncoding.EncodeToString(syncPasswordBytes)

		// Save sync password hash to database
		if err := syncauth.SetSyncAuthPassword(s.db, syncPassword); err != nil {
			logger.Info("Error setting sync password", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		logger.Info("✅ Generated sync authentication password (must be changed by admin)")

		// Store encryption key in session/cookie temporarily
		http.SetCookie(w, &http.Cookie{
			Name:     "setup_key",
			Value:    encryptionKey,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   600, // 10 minutes to complete setup
		})

		// Show success page with encryption key and sync password
		data := TemplateData{
			Lang:          lang,
			Title:         i18n.T(lang, "setup.success.title"),
			EncryptionKey: encryptionKey,
			SyncPassword:  syncPassword, // Display generated sync password
			CSRFToken:     auth.GetCSRFFromRequest(r),
		}

		if err := s.templates.ExecuteTemplate(w, "setup_success.html", data); err != nil {
			logger.Info("Error rendering success template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}

// handleSetupConfirm handles the final confirmation of setup
func (s *Server) handleSetupConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate CSRF token
	if !auth.ValidateCSRF(r) {
		logger.Warn("CSRF validation failed on setup/confirm")
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Setup completion is tracked by the existence of the database
	// Cleanup removes temporary setup state file

	// Clear the setup key cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "setup_key",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	logger.Info("✅ Initial setup completed successfully")

	// Redirect to login
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// handleAdminUsers displays the list of all users
