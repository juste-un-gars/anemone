// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/juste-un-gars/anemone/internal/activation"
	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/quota"
	"github.com/juste-un-gars/anemone/internal/reset"
	"github.com/juste-un-gars/anemone/internal/shares"
	"github.com/juste-un-gars/anemone/internal/smb"
	"github.com/juste-un-gars/anemone/internal/users"
)

// clientIP extracts the client IP from the request, stripping the port.
func clientIP(r *http.Request) string {
	ip := r.RemoteAddr
	// Strip port from RemoteAddr (e.g. "192.168.1.1:12345" -> "192.168.1.1")
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// handleLogin handles the login page
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	lang := s.getLang(r)

	// Only allow GET and POST
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	csrfToken := auth.GetCSRFFromRequest(r)

	if r.Method == http.MethodGet {
		// Show login form
		data := TemplateData{
			Lang:      lang,
			Title:     i18n.T(lang, "login.title"),
			CSRFToken: csrfToken,
		}

		if err := s.templates.ExecuteTemplate(w, "login.html", data); err != nil {
			logger.Info("Error rendering login template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	} else if r.Method == http.MethodPost {
		// Validate CSRF token
		if !auth.ValidateCSRF(r) {
			logger.Warn("CSRF validation failed on login", "ip", clientIP(r))
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Check rate limiting
		ip := clientIP(r)
		rl := auth.GetLoginRateLimiter()
		if blocked, remaining := rl.IsBlocked(ip); blocked {
			logger.Warn("Login blocked by rate limiter", "ip", ip, "remaining", remaining.Round(time.Second))
			data := TemplateData{
				Lang:      lang,
				Title:     i18n.T(lang, "login.title"),
				Error:     i18n.T(lang, "login.rate_limited"),
				CSRFToken: csrfToken,
			}
			w.WriteHeader(http.StatusTooManyRequests)
			if err := s.templates.ExecuteTemplate(w, "login.html", data); err != nil {
				logger.Info("Error rendering login template", "error", err)
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			}
			return
		}

		// Process login
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")
		rememberMe := r.FormValue("remember_me") == "on"

		// Get user from database
		user, err := users.GetByUsername(s.db, username)
		if err != nil || !user.CheckPassword(password) {
			// Record failed attempt
			locked := rl.RecordFailure(ip)
			remaining := rl.RemainingAttempts(ip)
			logger.Warn("Failed login attempt", "username", username, "ip", ip, "remaining_attempts", remaining, "locked_out", locked)

			// Show error
			data := TemplateData{
				Lang:      lang,
				Title:     i18n.T(lang, "login.title"),
				Error:     i18n.T(lang, "login.error"),
				CSRFToken: csrfToken,
			}

			if err := s.templates.ExecuteTemplate(w, "login.html", data); err != nil {
				logger.Info("Error rendering login template", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		// Successful login - clear rate limit counter
		rl.RecordSuccess(ip)

		// Get client info for session tracking
		userAgent := r.UserAgent()
		ipAddress := r.RemoteAddr

		// Create session with remember me option
		sm := auth.GetSessionManager()
		session, err := sm.CreateSessionWithOptions(user.ID, user.Username, user.IsAdmin, rememberMe, userAgent, ipAddress)
		if err != nil {
			logger.Info("Error creating session", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Set cookie
		auth.SetSessionCookie(w, session.ID, rememberMe)

		// Update last login
		user.UpdateLastLogin(s.db)

		logger.Info("User logged in", "username", user.Username, "is_admin", user.IsAdmin, "ip", ip)

		// Redirect to dashboard
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

// handleLogout handles logout
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Get session
	session, err := auth.GetSessionFromRequest(r)
	if err == nil {
		// Delete session
		sm := auth.GetSessionManager()
		sm.DeleteSession(session.ID)
		logger.Info("User logged out", "username", session.Username)
	}

	// Clear cookie
	auth.ClearSessionCookie(w)

	// Redirect to login
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// handleActivate handles user account activation
func (s *Server) handleActivate(w http.ResponseWriter, r *http.Request) {
	lang := s.getLang(r)

	// Extract token from URL: /activate/{token}
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}

	tokenString := parts[1]

	// Get token from database
	token, err := activation.GetTokenByString(s.db, tokenString)
	if err != nil {
		// Token not found
		data := TemplateData{
			Lang:  lang,
			Title: i18n.T(lang, "activate.title"),
			Error: i18n.T(lang, "activate.errors.invalid_token"),
		}
		s.templates.ExecuteTemplate(w, "activate.html", data)
		return
	}

	// Validate token
	if !token.IsValid() {
		var errorMsg string
		if token.UsedAt != nil {
			errorMsg = i18n.T(lang, "activate.errors.token_used")
		} else {
			errorMsg = i18n.T(lang, "activate.errors.invalid_token")
		}

		data := TemplateData{
			Lang:  lang,
			Title: i18n.T(lang, "activate.title"),
			Error: errorMsg,
		}
		s.templates.ExecuteTemplate(w, "activate.html", data)
		return
	}

	if r.Method == http.MethodGet {
		// Show activation form
		data := struct {
			Lang      string
			Title     string
			Username  string
			Token     string
			Error     string
			CSRFToken string
			T         func(string) string
		}{
			Lang:      lang,
			Title:     i18n.T(lang, "activate.title"),
			Username:  token.Username,
			Token:     tokenString,
			CSRFToken: auth.GetCSRFFromRequest(r),
		}

		if err := s.templates.ExecuteTemplate(w, "activate.html", data); err != nil {
			logger.Info("Error rendering activate template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	} else if r.Method == http.MethodPost {
		// Validate CSRF token
		if !auth.ValidateCSRF(r) {
			logger.Warn("CSRF validation failed on activate")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Process activation
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		password := r.FormValue("password")
		passwordConfirm := r.FormValue("password_confirm")

		// Validate
		csrfToken := auth.GetCSRFFromRequest(r)

		if password != passwordConfirm {
			data := struct {
				Lang      string
				Title     string
				Username  string
				Token     string
				Error     string
				CSRFToken string
				T         func(string) string
			}{
				Lang:      lang,
				Title:     i18n.T(lang, "activate.title"),
				Username:  token.Username,
				Token:     tokenString,
				Error:     i18n.T(lang, "activate.errors.password_mismatch"),
				CSRFToken: csrfToken,
			}
			s.templates.ExecuteTemplate(w, "activate.html", data)
			return
		}

		if len(password) < 8 {
			data := struct {
				Lang      string
				Title     string
				Username  string
				Token     string
				Error     string
				CSRFToken string
				T         func(string) string
			}{
				Lang:      lang,
				Title:     i18n.T(lang, "activate.title"),
				Username:  token.Username,
				Token:     tokenString,
				Error:     i18n.T(lang, "activate.errors.password_length"),
				CSRFToken: csrfToken,
			}
			s.templates.ExecuteTemplate(w, "activate.html", data)
			return
		}

		// Get master key from system config
		var masterKey string
		err := s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
		if err != nil {
			logger.Info("Error getting master key", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Activate user (sets password and generates encryption key)
		encryptionKey, err := users.ActivateUser(s.db, token.UserID, password, masterKey)
		if err != nil {
			logger.Info("Error activating user", "error", err)
			http.Error(w, "Failed to activate user", http.StatusInternalServerError)
			return
		}

		// Mark token as used
		if err := token.MarkAsUsed(s.db); err != nil {
			logger.Info("Error marking token as used", "error", err)
		}

		logger.Info("User activated", "username", token.Username, "user_id", token.UserID)

		// Create SMB user with same password
		if err := smb.AddSMBUser(token.Username, password); err != nil {
			logger.Info("Warning: Failed to create SMB user", "error", err)
		}

		// Get user info to retrieve quotas
		user, err := users.GetByID(s.db, token.UserID)
		if err != nil {
			logger.Info("Warning: Failed to get user info", "error", err)
			// Default quotas if we can't retrieve them
			user = &users.User{
				QuotaBackupGB: 50,
				QuotaTotalGB:  100,
			}
		}

		// Calculate data quota (total - backup)
		dataQuotaGB := user.QuotaTotalGB - user.QuotaBackupGB
		if dataQuotaGB < 0 {
			dataQuotaGB = user.QuotaTotalGB / 2 // Fallback: split evenly
		}

		// Initialize quota manager for creating directories with quota enforcement
		qm, err := quota.NewQuotaManager(s.cfg.SharesDir)
		if err != nil {
			logger.Info("Warning: Failed to initialize quota manager", "error", err)
			qm = nil // Will create regular directories as fallback
		}

		// Create default shares: backup and data with quota enforcement
		// Owner is set atomically during creation to avoid separate chown
		owner := fmt.Sprintf("%s:%s", token.Username, token.Username)
		backupPath := filepath.Join(s.cfg.SharesDir, token.Username, "backup")
		if qm != nil {
			if err := qm.CreateQuotaDir(backupPath, user.QuotaBackupGB, owner); err != nil {
				logger.Info("Warning: Failed to create backup quota directory", "error", err)
			} else {
				logger.Info("Created backup subvolume", "quota_gb", user.QuotaBackupGB, "owner", token.Username)
			}
		}

		backupShare := &shares.Share{
			UserID:      token.UserID,
			Name:        fmt.Sprintf("backup_%s", token.Username),
			Path:        backupPath,
			Protocol:    "smb",
			SyncEnabled: true,
		}
		if err := shares.Create(s.db, backupShare, token.Username); err != nil {
			logger.Info("Warning: Failed to create backup share", "error", err)
		} else {
			logger.Info("Created backup share: backup_", "username", token.Username)
		}

		dataPath := filepath.Join(s.cfg.SharesDir, token.Username, "data")
		if qm != nil {
			if err := qm.CreateQuotaDir(dataPath, dataQuotaGB, owner); err != nil {
				logger.Info("Warning: Failed to create data quota directory", "error", err)
			} else {
				logger.Info("Created data subvolume", "quota_gb", dataQuotaGB, "owner", token.Username)
			}
		}

		dataShare := &shares.Share{
			UserID:      token.UserID,
			Name:        fmt.Sprintf("data_%s", token.Username),
			Path:        dataPath,
			Protocol:    "smb",
			SyncEnabled: false,
		}
		if err := shares.Create(s.db, dataShare, token.Username); err != nil {
			logger.Info("Warning: Failed to create data share", "error", err)
		} else {
			logger.Info("Created data share: data_", "username", token.Username)
		}

		// Regenerate SMB config
		// Use system-wide dfree wrapper
		dfreePath := "/usr/local/bin/anemone-dfree-wrapper.sh"

		smbCfg := &smb.Config{
			ConfigPath: filepath.Join(s.cfg.DataDir, "smb", "smb.conf"),
			WorkGroup:  "ANEMONE",
			ServerName: "Anemone NAS",
			SharesDir:  s.cfg.SharesDir,
			DfreePath:  dfreePath,
		}
		if err := smb.GenerateConfig(s.db, smbCfg); err != nil {
			logger.Info("Warning: Failed to regenerate SMB config", "error", err)
		} else {
			// Try to reload smbd (requires sudoers configuration)
			if err := smb.ReloadConfig(); err != nil {
				logger.Info("Warning: Could not reload smbd automatically. Run: sudo systemctl reload smbd")
			} else {
				logger.Info("✅ SMB config reloaded successfully")
			}
		}

		// Store encryption key in cookie temporarily
		http.SetCookie(w, &http.Cookie{
			Name:     "activation_key",
			Value:    encryptionKey,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   600, // 10 minutes
		})

		// Show success page with encryption key
		data := struct {
			Lang          string
			Title         string
			Username      string
			EncryptionKey string
			CSRFToken     string
			T             func(string) string
		}{
			Lang:          lang,
			Title:         i18n.T(lang, "activate.success.title"),
			Username:      token.Username,
			EncryptionKey: encryptionKey,
			CSRFToken:     auth.GetCSRFFromRequest(r),
		}

		if err := s.templates.ExecuteTemplate(w, "activate_success.html", data); err != nil {
			logger.Info("Error rendering activation success template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}

// handleActivateConfirm handles the final confirmation of activation
func (s *Server) handleActivateConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate CSRF token
	if !auth.ValidateCSRF(r) {
		logger.Warn("CSRF validation failed on activate/confirm")
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Clear the activation key cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "activation_key",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	logger.Info("✅ User activation confirmed")

	// Redirect to login
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// handleResetPasswordForm displays the password reset form
func (s *Server) handleResetPasswordForm(w http.ResponseWriter, r *http.Request) {
	lang := s.getLang(r)

	// Get token from query string
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		http.Error(w, "Token required", http.StatusBadRequest)
		return
	}

	// Get token from database
	token, err := reset.GetTokenByString(s.db, tokenString)
	if err != nil {
		logger.Info("Token not found", "error", err)
		data := struct {
			Lang    string
			Title   string
			Error   string
		}{
			Lang:  lang,
			Title: i18n.T(lang, "reset.title"),
			Error: i18n.T(lang, "reset.token_invalid"),
		}
		if err := s.templates.ExecuteTemplate(w, "reset_password.html", data); err != nil {
			logger.Info("Error rendering reset password template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Check if token is valid
	if !token.IsValid() {
		data := struct {
			Lang    string
			Title   string
			Error   string
		}{
			Lang:  lang,
			Title: i18n.T(lang, "reset.title"),
			Error: i18n.T(lang, "reset.token_invalid"),
		}
		if err := s.templates.ExecuteTemplate(w, "reset_password.html", data); err != nil {
			logger.Info("Error rendering reset password template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Get user info
	user, err := users.GetByID(s.db, token.UserID)
	if err != nil {
		logger.Info("Error getting user", "error", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Show form
	data := struct {
		Lang      string
		Title     string
		Token     string
		Username  string
		Error     string
		CSRFToken string
	}{
		Lang:      lang,
		Title:     i18n.T(lang, "reset.title"),
		Token:     tokenString,
		Username:  user.Username,
		Error:     r.URL.Query().Get("error"),
		CSRFToken: auth.GetCSRFFromRequest(r),
	}

	if err := s.templates.ExecuteTemplate(w, "reset_password.html", data); err != nil {
		logger.Info("Error rendering reset password template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleResetPasswordSubmit processes the password reset form
func (s *Server) handleResetPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate CSRF token
	if !auth.ValidateCSRF(r) {
		logger.Warn("CSRF validation failed on reset-password")
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/reset-password?error=Invalid+form+data", http.StatusSeeOther)
		return
	}

	tokenString := r.FormValue("token")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate
	if tokenString == "" || newPassword == "" || confirmPassword == "" {
		http.Redirect(w, r, fmt.Sprintf("/reset-password?token=%s&error=All+fields+are+required", tokenString), http.StatusSeeOther)
		return
	}

	if newPassword != confirmPassword {
		http.Redirect(w, r, fmt.Sprintf("/reset-password?token=%s&error=Passwords+do+not+match", tokenString), http.StatusSeeOther)
		return
	}

	if len(newPassword) < 8 {
		http.Redirect(w, r, fmt.Sprintf("/reset-password?token=%s&error=Password+must+be+at+least+8+characters", tokenString), http.StatusSeeOther)
		return
	}

	// Get token from database
	token, err := reset.GetTokenByString(s.db, tokenString)
	if err != nil || !token.IsValid() {
		http.Redirect(w, r, fmt.Sprintf("/reset-password?token=%s&error=Invalid+or+expired+token", tokenString), http.StatusSeeOther)
		return
	}

	// Get user
	user, err := users.GetByID(s.db, token.UserID)
	if err != nil {
		logger.Info("Error getting user", "error", err)
		http.Redirect(w, r, fmt.Sprintf("/reset-password?token=%s&error=User+not+found", tokenString), http.StatusSeeOther)
		return
	}

	// Get master key
	var masterKey string
	err = s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		logger.Info("Error getting master key", "error", err)
		http.Redirect(w, r, fmt.Sprintf("/reset-password?token=%s&error=System+configuration+error", tokenString), http.StatusSeeOther)
		return
	}

	// Reset password (update DB + SMB)
	err = users.ResetPassword(s.db, user.ID, user.Username, newPassword, masterKey)
	if err != nil {
		logger.Info("Error resetting password", "error", err)
		http.Redirect(w, r, fmt.Sprintf("/reset-password?token=%s&error=Failed+to+reset+password", tokenString), http.StatusSeeOther)
		return
	}

	// Mark token as used
	if err := token.MarkAsUsed(s.db); err != nil {
		logger.Info("Error marking token as used", "error", err)
		// Non-critical, continue
	}

	logger.Info("Password reset successfully for user", "username", user.Username)

	// Redirect to login with success message
	http.Redirect(w, r, "/login?success=Password+reset+successfully", http.StatusSeeOther)
}
