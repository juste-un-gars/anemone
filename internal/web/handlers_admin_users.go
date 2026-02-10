// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"fmt"
	"github.com/juste-un-gars/anemone/internal/logger"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/juste-un-gars/anemone/internal/activation"
	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/reset"
	"github.com/juste-un-gars/anemone/internal/users"
)

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Get all users
	allUsers, err := users.GetAllUsers(s.db)
	if err != nil {
		logger.Info("Error getting users: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		V2TemplateData
		Users []*users.User
	}{
		V2TemplateData: V2TemplateData{
			Lang:       lang,
			Title:      i18n.T(lang, "users.title"),
			ActivePage: "users",
			Session:    session,
		},
		Users: allUsers,
	}

	tmpl := s.loadV2Page("v2_users.html", s.funcMap)
	if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
		logger.Info("Error rendering users template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleAdminUsersAdd handles adding a new user
func (s *Server) handleAdminUsersAdd(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	if r.Method == http.MethodGet {
		// Show form
		data := struct {
			V2TemplateData
			Error string
		}{
			V2TemplateData: V2TemplateData{
				Lang:       lang,
				Title:      i18n.T(lang, "users.add.title"),
				ActivePage: "users",
				Session:    session,
			},
			Error: r.URL.Query().Get("error"),
		}

		tmpl := s.loadV2Page("v2_users_add.html", s.funcMap)
		if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
			logger.Info("Error rendering add user template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	} else if r.Method == http.MethodPost {
		// Process form
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		username := strings.TrimSpace(r.FormValue("username"))
		email := strings.TrimSpace(r.FormValue("email"))
		isAdminStr := r.FormValue("is_admin")
		quotaDataStr := r.FormValue("quota_data")
		quotaBackupStr := r.FormValue("quota_backup")

		// Validate
		if username == "" {
			s.renderUsersAddError(w, lang, session, i18n.T(lang, "users.errors.username_required"))
			return
		}

		// Validate username format (prevent command injection)
		if err := users.ValidateUsername(username); err != nil {
			s.renderUsersAddError(w, lang, session, fmt.Sprintf("Invalid username format: %v", err))
			return
		}

		// Check if username already exists
		_, err := users.GetByUsername(s.db, username)
		if err == nil {
			s.renderUsersAddError(w, lang, session, i18n.T(lang, "users.errors.username_exists"))
			return
		}

		isAdmin := isAdminStr == "true"
		quotaData, _ := strconv.Atoi(quotaDataStr)
		quotaBackup, _ := strconv.Atoi(quotaBackupStr)

		if quotaData <= 0 {
			quotaData = 50
		}
		if quotaBackup <= 0 {
			quotaBackup = 50
		}

		// Calculate total quota (backup + data)
		quotaTotal := quotaBackup + quotaData

		// Create pending user
		user, err := users.CreatePendingUser(s.db, username, email, isAdmin, quotaTotal, quotaBackup)
		if err != nil {
			logger.Info("Error creating user: %v", err)
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		logger.Info("Created pending user: %s (ID: %d)", user.Username, user.ID)

		// Create activation token
		token, err := activation.CreateActivationToken(s.db, user.ID, user.Username, user.Email)
		if err != nil {
			logger.Info("Error creating activation token: %v", err)
			http.Error(w, "Failed to create activation token", http.StatusInternalServerError)
			return
		}

		logger.Info("Created activation token for user %s (expires: %v)", user.Username, token.ExpiresAt)

		// Redirect to token display page
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%d/token", user.ID), http.StatusSeeOther)
	}
}

// renderUsersAddError renders the add user form with an error message.
func (s *Server) renderUsersAddError(w http.ResponseWriter, lang string, session *auth.Session, errMsg string) {
	data := struct {
		V2TemplateData
		Error string
	}{
		V2TemplateData: V2TemplateData{
			Lang:       lang,
			Title:      i18n.T(lang, "users.add.title"),
			ActivePage: "users",
			Session:    session,
		},
		Error: errMsg,
	}
	tmpl := s.loadV2Page("v2_users_add.html", s.funcMap)
	tmpl.ExecuteTemplate(w, "v2_base", data)
}

// handleAdminUsersActions handles user-specific actions (token display, delete)
func (s *Server) handleAdminUsersActions(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)
	path := r.URL.Path

	// Parse URL: /admin/users/{id}/{action}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}

	userIDStr := parts[2]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Get action
	action := ""
	if len(parts) >= 4 {
		action = parts[3]
	}

	switch action {
	case "token":
		// Display activation token
		user, err := users.GetByID(s.db, userID)
		if err != nil {
			logger.Info("Error getting user: %v", err)
			http.NotFound(w, r)
			return
		}

		// Get or create token
		var token *activation.Token
		pendingTokens, err := activation.GetPendingTokens(s.db)
		if err == nil {
			for _, t := range pendingTokens {
				if t.UserID == userID {
					token = t
					break
				}
			}
		}

		// If no existing token, create new one
		if token == nil {
			token, err = activation.CreateActivationToken(s.db, user.ID, user.Username, user.Email)
			if err != nil {
				logger.Info("Error creating token: %v", err)
				http.Error(w, "Failed to create token", http.StatusInternalServerError)
				return
			}
		}

		// Build activation URL - use Host from request (includes IP if accessed via IP)
		host := r.Host
		if host == "" || host == "localhost" || strings.HasPrefix(host, "localhost:") {
			// Fallback to configured port if Host is empty or localhost
			if s.cfg.EnableHTTPS {
				host = fmt.Sprintf("localhost:%s", s.cfg.HTTPSPort)
			} else {
				host = fmt.Sprintf("localhost:%s", s.cfg.Port)
			}
		}
			// Use HTTPS if enabled, otherwise HTTP
		protocol := "https"
		if !s.cfg.EnableHTTPS {
			protocol = "http"
		}
		activationURL := fmt.Sprintf("%s://%s/activate/%s", protocol, host, token.Token)

		data := struct {
			V2TemplateData
			Username      string
			Email         string
			ActivationURL string
			ExpiresAt     time.Time
		}{
			V2TemplateData: V2TemplateData{
				Lang:       lang,
				Title:      i18n.T(lang, "users.token.title"),
				ActivePage: "users",
				Session:    session,
			},
			Username:      user.Username,
			Email:         user.Email,
			ActivationURL: activationURL,
			ExpiresAt:     token.ExpiresAt,
		}

		tmpl := s.loadV2Page("v2_users_token.html", s.funcMap)
		if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
			logger.Info("Error rendering token template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	case "reset":
		// Generate password reset token
		user, err := users.GetByID(s.db, userID)
		if err != nil {
			logger.Info("Error getting user: %v", err)
			http.NotFound(w, r)
			return
		}

		// Create reset token
		token, err := reset.CreatePasswordResetToken(s.db, user.ID)
		if err != nil {
			logger.Info("Error creating reset token: %v", err)
			http.Error(w, "Failed to create reset token", http.StatusInternalServerError)
			return
		}

		// Build reset URL - use Host from request
		host := r.Host
		if host == "" || host == "localhost" || strings.HasPrefix(host, "localhost:") {
			if s.cfg.EnableHTTPS {
				host = fmt.Sprintf("localhost:%s", s.cfg.HTTPSPort)
			} else {
				host = fmt.Sprintf("localhost:%s", s.cfg.Port)
			}
		}
		protocol := "https"
		if !s.cfg.EnableHTTPS {
			protocol = "http"
		}
		resetURL := fmt.Sprintf("%s://%s/reset-password?token=%s", protocol, host, token.Token)

		data := struct {
			V2TemplateData
			Username  string
			Email     string
			ResetURL  string
			ExpiresAt time.Time
		}{
			V2TemplateData: V2TemplateData{
				Lang:       lang,
				Title:      i18n.T(lang, "reset.token.title"),
				ActivePage: "users",
				Session:    session,
			},
			Username:  user.Username,
			Email:     user.Email,
			ResetURL:  resetURL,
			ExpiresAt: token.ExpiresAt,
		}

		tmpl := s.loadV2Page("v2_users_reset_token.html", s.funcMap)
		if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
			logger.Info("Error rendering reset token template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	case "quota":
		// Edit user quota
		s.handleAdminUserQuota(w, r, userID, session, lang)
		return

	case "delete":
		// Delete user
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Prevent users from deleting themselves
		if session.UserID == userID {
			http.Error(w, "Cannot delete your own account", http.StatusForbidden)
			return
		}

		err := users.DeleteUser(s.db, userID, s.cfg.DataDir, s.cfg.SharesDir)
		if err != nil {
			logger.Info("Error deleting user: %v", err)
			http.Error(w, "Failed to delete user", http.StatusInternalServerError)
			return
		}

		logger.Info("User %d deleted by admin %s", userID, session.Username)
		w.WriteHeader(http.StatusOK)

	default:
		http.NotFound(w, r)
	}
}

