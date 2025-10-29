// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/juste-un-gars/anemone/internal/activation"
	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/config"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/peers"
	"github.com/juste-un-gars/anemone/internal/shares"
	"github.com/juste-un-gars/anemone/internal/smb"
	"github.com/juste-un-gars/anemone/internal/users"
)

// Server holds the web server state
type Server struct {
	db        *sql.DB
	cfg       *config.Config
	templates *template.Template
}

// TemplateData holds data passed to templates
type TemplateData struct {
	Lang          string
	Title         string
	EncryptionKey string
	Error         string
	Session       *auth.Session
	Stats         *DashboardStats
	Users         []*users.User
}

// DashboardStats holds dashboard statistics
type DashboardStats struct {
	UserCount      int
	StorageUsed    string
	StorageQuota   string
	StoragePercent int
	LastBackup     string
	PeerCount      int
	TrashCount     int
}

// NewRouter creates and configures the HTTP router
func NewRouter(db *sql.DB, cfg *config.Config) http.Handler {
	// Initialize i18n
	if err := i18n.Init(cfg.Language); err != nil {
		log.Printf("Warning: Failed to initialize i18n: %v", err)
	}

	// Create template with translation function
	funcMap := template.FuncMap{
		"T": func(lang, key string) string {
			return i18n.T(lang, key)
		},
	}

	templates := template.Must(template.New("").Funcs(funcMap).ParseGlob(filepath.Join("web", "templates", "*.html")))

	server := &Server{
		db:        db,
		cfg:       cfg,
		templates: templates,
	}

	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	// Public routes
	mux.HandleFunc("/", server.handleHome)
	mux.HandleFunc("/setup", server.handleSetup)
	mux.HandleFunc("/setup/confirm", server.handleSetupConfirm)
	mux.HandleFunc("/login", auth.RedirectIfAuthenticated(server.handleLogin))
	mux.HandleFunc("/logout", server.handleLogout)

	// Activation routes (public)
	mux.HandleFunc("/activate/", server.handleActivate)
	mux.HandleFunc("/activate/confirm", server.handleActivateConfirm)

	// Protected routes
	mux.HandleFunc("/dashboard", auth.RequireAuth(server.handleDashboard))

	// Admin routes - Users
	mux.HandleFunc("/admin/users", auth.RequireAdmin(server.handleAdminUsers))
	mux.HandleFunc("/admin/users/add", auth.RequireAdmin(server.handleAdminUsersAdd))
	mux.HandleFunc("/admin/users/", auth.RequireAdmin(server.handleAdminUsersActions))

	// Admin routes - Peers
	mux.HandleFunc("/admin/peers", auth.RequireAdmin(server.handleAdminPeers))
	mux.HandleFunc("/admin/peers/add", auth.RequireAdmin(server.handleAdminPeersAdd))
	mux.HandleFunc("/admin/peers/", auth.RequireAdmin(server.handleAdminPeersActions))

	// Admin routes - Settings
	mux.HandleFunc("/admin/settings", auth.RequireAdmin(server.handleAdminSettings))

	// User routes
	mux.HandleFunc("/trash", auth.RequireAuth(server.handleTrash))

	// Admin routes - Shares
	mux.HandleFunc("/admin/shares", auth.RequireAdmin(server.handleAdminShares))

	return mux
}

// isSetupCompleted checks if initial setup is done
func (s *Server) isSetupCompleted() bool {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM system_config WHERE key = 'setup_completed'").Scan(&count)
	return err == nil && count > 0
}

// getLang gets language from query param or config
func (s *Server) getLang(r *http.Request) string {
	lang := ""
	if l := r.URL.Query().Get("lang"); l != "" {
		lang = l
	} else {
		lang = s.cfg.Language
	}

	// Normalize language code (fr_FR -> fr, en_US -> en, etc.)
	if len(lang) > 2 {
		lang = lang[:2]
	}

	// Default to fr if unknown
	if lang != "fr" && lang != "en" {
		lang = "fr"
	}

	return lang
}

// handleHome handles the root path
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	// Check setup first
	if !s.isSetupCompleted() {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}

	// If not authenticated, redirect to login
	if !auth.IsAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Redirect to dashboard
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// handleLogin handles the login page
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	lang := s.getLang(r)

	if r.Method == http.MethodGet {
		// Show login form
		data := TemplateData{
			Lang:  lang,
			Title: i18n.T(lang, "login.title"),
		}

		if err := s.templates.ExecuteTemplate(w, "login.html", data); err != nil {
			log.Printf("Error rendering login template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	} else if r.Method == http.MethodPost {
		// Process login
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")

		// Get user from database
		user, err := users.GetByUsername(s.db, username)
		if err != nil || !user.CheckPassword(password) {
			// Show error
			data := TemplateData{
				Lang:  lang,
				Title: i18n.T(lang, "login.title"),
				Error: i18n.T(lang, "login.error"),
			}

			if err := s.templates.ExecuteTemplate(w, "login.html", data); err != nil {
				log.Printf("Error rendering login template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		// Create session
		sm := auth.GetSessionManager()
		session, err := sm.CreateSession(user.ID, user.Username, user.IsAdmin)
		if err != nil {
			log.Printf("Error creating session: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Set cookie
		auth.SetSessionCookie(w, session.ID)

		// Update last login
		user.UpdateLastLogin(s.db)

		log.Printf("User logged in: %s (admin: %v)", user.Username, user.IsAdmin)

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
		log.Printf("User logged out: %s", session.Username)
	}

	// Clear cookie
	auth.ClearSessionCookie(w)

	// Redirect to login
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// handleDashboard handles the dashboard page
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Get stats
	stats := s.getDashboardStats(session)

	data := TemplateData{
		Lang:    lang,
		Title:   i18n.T(lang, "dashboard.title"),
		Session: session,
		Stats:   stats,
	}

	// Choose template based on role
	template := "dashboard_user.html"
	if session.IsAdmin {
		template = "dashboard_admin.html"
	}

	if err := s.templates.ExecuteTemplate(w, template, data); err != nil {
		log.Printf("Error rendering dashboard template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// getDashboardStats retrieves dashboard statistics
func (s *Server) getDashboardStats(session *auth.Session) *DashboardStats {
	stats := &DashboardStats{
		StorageUsed:    "0 GB",
		StorageQuota:   "100 GB",
		StoragePercent: 0,
		LastBackup:     "Jamais",
		UserCount:      0,
		PeerCount:      0,
		TrashCount:     0,
	}

	// Count users (admin only)
	if session.IsAdmin {
		s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.UserCount)
		s.db.QueryRow("SELECT COUNT(*) FROM peers").Scan(&stats.PeerCount)
	}

	// Count trash items for this user
	s.db.QueryRow("SELECT COUNT(*) FROM trash_items WHERE user_id = ?", session.UserID).Scan(&stats.TrashCount)

	// TODO: Calculate actual storage usage
	// TODO: Get last backup time from sync_log

	return stats
}

// handleSetup handles the setup page
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
			Lang:  lang,
			Title: i18n.T(lang, "setup.title"),
		}

		if err := s.templates.ExecuteTemplate(w, "setup.html", data); err != nil {
			log.Printf("Error rendering setup template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	} else if r.Method == http.MethodPost {
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
			log.Printf("Error generating master key: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		masterKey := base64.StdEncoding.EncodeToString(masterKeyBytes)

		// Create first admin user
		user, encryptionKey, err := users.CreateFirstAdmin(s.db, username, password, email, masterKey)
		if err != nil {
			log.Printf("Error creating admin user: %v", err)
			http.Error(w, "Failed to create admin user", http.StatusInternalServerError)
			return
		}

		log.Printf("Created admin user: %s (ID: %d)", user.Username, user.ID)

		// Save system configuration
		tx, err := s.db.Begin()
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
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
				log.Printf("Error saving config %s: %v", key, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		if err := tx.Commit(); err != nil {
			log.Printf("Error committing transaction: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Store encryption key in session/cookie temporarily
		http.SetCookie(w, &http.Cookie{
			Name:     "setup_key",
			Value:    encryptionKey,
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // Set to true in production with HTTPS
			MaxAge:   600,   // 10 minutes to complete setup
		})

		// Show success page with encryption key
		data := TemplateData{
			Lang:          lang,
			Title:         i18n.T(lang, "setup.success.title"),
			EncryptionKey: encryptionKey,
		}

		if err := s.templates.ExecuteTemplate(w, "setup_success.html", data); err != nil {
			log.Printf("Error rendering success template: %v", err)
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

	// Mark setup as completed
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO system_config (key, value, updated_at) VALUES (?, ?, ?)",
		"setup_completed",
		time.Now().Format(time.RFC3339),
		time.Now(),
	)
	if err != nil {
		log.Printf("Error marking setup as completed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Clear the setup key cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "setup_key",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	log.Println("✅ Initial setup completed successfully")

	// Redirect to login
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// handleAdminUsers displays the list of all users
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
		log.Printf("Error getting users: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := TemplateData{
		Lang:    lang,
		Title:   i18n.T(lang, "users.title"),
		Session: session,
		Users:   allUsers,
	}

	if err := s.templates.ExecuteTemplate(w, "admin_users.html", data); err != nil {
		log.Printf("Error rendering users template: %v", err)
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
		data := TemplateData{
			Lang:    lang,
			Title:   i18n.T(lang, "users.add.title"),
			Session: session,
		}

		if err := s.templates.ExecuteTemplate(w, "admin_users_add.html", data); err != nil {
			log.Printf("Error rendering add user template: %v", err)
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
		quotaTotalStr := r.FormValue("quota_total")
		quotaBackupStr := r.FormValue("quota_backup")

		// Validate
		if username == "" {
			data := TemplateData{
				Lang:    lang,
				Title:   i18n.T(lang, "users.add.title"),
				Session: session,
				Error:   i18n.T(lang, "users.errors.username_required"),
			}
			s.templates.ExecuteTemplate(w, "admin_users_add.html", data)
			return
		}

		// Check if username already exists
		_, err := users.GetByUsername(s.db, username)
		if err == nil {
			data := TemplateData{
				Lang:    lang,
				Title:   i18n.T(lang, "users.add.title"),
				Session: session,
				Error:   i18n.T(lang, "users.errors.username_exists"),
			}
			s.templates.ExecuteTemplate(w, "admin_users_add.html", data)
			return
		}

		isAdmin := isAdminStr == "true"
		quotaTotal, _ := strconv.Atoi(quotaTotalStr)
		quotaBackup, _ := strconv.Atoi(quotaBackupStr)

		if quotaTotal <= 0 {
			quotaTotal = 100
		}
		if quotaBackup <= 0 {
			quotaBackup = 50
		}

		// Create pending user
		user, err := users.CreatePendingUser(s.db, username, email, isAdmin, quotaTotal, quotaBackup)
		if err != nil {
			log.Printf("Error creating user: %v", err)
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		log.Printf("Created pending user: %s (ID: %d)", user.Username, user.ID)

		// Create activation token
		token, err := activation.CreateActivationToken(s.db, user.ID, user.Username, user.Email)
		if err != nil {
			log.Printf("Error creating activation token: %v", err)
			http.Error(w, "Failed to create activation token", http.StatusInternalServerError)
			return
		}

		log.Printf("Created activation token for user %s (expires: %v)", user.Username, token.ExpiresAt)

		// Redirect to token display page
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%d/token", user.ID), http.StatusSeeOther)
	}
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
			log.Printf("Error getting user: %v", err)
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
				log.Printf("Error creating token: %v", err)
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
			Lang          string
			Title         string
			Session       *auth.Session
			Username      string
			Email         string
			ActivationURL string
			ExpiresAt     time.Time
			T             func(string) string
		}{
			Lang:          lang,
			Title:         i18n.T(lang, "users.token.title"),
			Session:       session,
			Username:      user.Username,
			Email:         user.Email,
			ActivationURL: activationURL,
			ExpiresAt:     token.ExpiresAt,
		}

		if err := s.templates.ExecuteTemplate(w, "admin_users_token.html", data); err != nil {
			log.Printf("Error rendering token template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	case "delete":
		// Delete user
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err := users.DeleteUser(s.db, userID)
		if err != nil {
			log.Printf("Error deleting user: %v", err)
			http.Error(w, "Failed to delete user", http.StatusInternalServerError)
			return
		}

		log.Printf("User %d deleted by admin %s", userID, session.Username)
		w.WriteHeader(http.StatusOK)

	default:
		http.NotFound(w, r)
	}
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
			Lang     string
			Title    string
			Username string
			Token    string
			Error    string
			T        func(string) string
		}{
			Lang:     lang,
			Title:    i18n.T(lang, "activate.title"),
			Username: token.Username,
			Token:    tokenString,
		}

		if err := s.templates.ExecuteTemplate(w, "activate.html", data); err != nil {
			log.Printf("Error rendering activate template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	} else if r.Method == http.MethodPost {
		// Process activation
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		password := r.FormValue("password")
		passwordConfirm := r.FormValue("password_confirm")

		// Validate
		if password != passwordConfirm {
			data := struct {
				Lang     string
				Title    string
				Username string
				Token    string
				Error    string
				T        func(string) string
			}{
				Lang:     lang,
				Title:    i18n.T(lang, "activate.title"),
				Username: token.Username,
				Token:    tokenString,
				Error:    i18n.T(lang, "activate.errors.password_mismatch"),
			}
			s.templates.ExecuteTemplate(w, "activate.html", data)
			return
		}

		if len(password) < 8 {
			data := struct {
				Lang     string
				Title    string
				Username string
				Token    string
				Error    string
				T        func(string) string
			}{
				Lang:     lang,
				Title:    i18n.T(lang, "activate.title"),
				Username: token.Username,
				Token:    tokenString,
				Error:    i18n.T(lang, "activate.errors.password_length"),
			}
			s.templates.ExecuteTemplate(w, "activate.html", data)
			return
		}

		// Get master key from system config
		var masterKey string
		err := s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
		if err != nil {
			log.Printf("Error getting master key: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Activate user (sets password and generates encryption key)
		encryptionKey, err := users.ActivateUser(s.db, token.UserID, password, masterKey)
		if err != nil {
			log.Printf("Error activating user: %v", err)
			http.Error(w, "Failed to activate user", http.StatusInternalServerError)
			return
		}

		// Mark token as used
		if err := token.MarkAsUsed(s.db); err != nil {
			log.Printf("Error marking token as used: %v", err)
		}

		log.Printf("User activated: %s (ID: %d)", token.Username, token.UserID)

		// Create SMB user with same password
		if err := smb.AddSMBUser(token.Username, password); err != nil {
			log.Printf("Warning: Failed to create SMB user: %v", err)
		}

		// Create default shares: backup and data
		backupShare := &shares.Share{
			UserID:      token.UserID,
			Name:        fmt.Sprintf("backup_%s", token.Username),
			Path:        filepath.Join(s.cfg.SharesDir, token.Username, "backup"),
			Protocol:    "smb",
			SyncEnabled: true,
		}
		if err := shares.Create(s.db, backupShare); err != nil {
			log.Printf("Warning: Failed to create backup share: %v", err)
		} else {
			log.Printf("Created backup share: backup_%s", token.Username)
		}

		dataShare := &shares.Share{
			UserID:      token.UserID,
			Name:        fmt.Sprintf("data_%s", token.Username),
			Path:        filepath.Join(s.cfg.SharesDir, token.Username, "data"),
			Protocol:    "smb",
			SyncEnabled: false,
		}
		if err := shares.Create(s.db, dataShare); err != nil {
			log.Printf("Warning: Failed to create data share: %v", err)
		} else {
			log.Printf("Created data share: data_%s", token.Username)
		}

		// Regenerate SMB config (reload done manually by admin)
		smbCfg := &smb.Config{
			ConfigPath: filepath.Join(s.cfg.DataDir, "smb", "smb.conf"),
			WorkGroup:  "ANEMONE",
			ServerName: "Anemone NAS",
			SharesDir:  s.cfg.SharesDir,
		}
		if err := smb.GenerateConfig(s.db, smbCfg); err != nil {
			log.Printf("Warning: Failed to regenerate SMB config: %v", err)
		}
		log.Printf("SMB config updated. Admin should run: sudo systemctl reload smbd")

		// Store encryption key in cookie temporarily
		http.SetCookie(w, &http.Cookie{
			Name:     "activation_key",
			Value:    encryptionKey,
			Path:     "/",
			HttpOnly: true,
			Secure:   false,
			MaxAge:   600, // 10 minutes
		})

		// Show success page with encryption key
		data := struct {
			Lang          string
			Title         string
			Username      string
			EncryptionKey string
			T             func(string) string
		}{
			Lang:          lang,
			Title:         i18n.T(lang, "activate.success.title"),
			Username:      token.Username,
			EncryptionKey: encryptionKey,
		}

		if err := s.templates.ExecuteTemplate(w, "activate_success.html", data); err != nil {
			log.Printf("Error rendering activation success template: %v", err)
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

	// Clear the activation key cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "activation_key",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	log.Println("✅ User activation confirmed")

	// Redirect to login
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Placeholder handlers for future implementation
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
		log.Printf("Error getting peers: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Lang    string
		Title   string
		Session *auth.Session
		Peers   []*peers.Peer
	}{
		Lang:    lang,
		Title:   i18n.T(lang, "peers.title"),
		Session: session,
		Peers:   peersList,
	}

	if err := s.templates.ExecuteTemplate(w, "admin_peers.html", data); err != nil {
		log.Printf("Error rendering peers template: %v", err)
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
			log.Printf("Error rendering peers add template: %v", err)
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
		enabled := r.FormValue("enabled") == "on"

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
				log.Printf("Error rendering peers add template: %v", err)
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
					log.Printf("Error rendering peers add template: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
				return
			}
		}

		// Create peer
		peer := &peers.Peer{
			Name:      name,
			Address:   address,
			Port:      port,
			PublicKey: publicKey,
			Enabled:   enabled,
			Status:    "unknown",
		}

		if err := peers.Create(s.db, peer); err != nil {
			log.Printf("Error creating peer: %v", err)
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
				log.Printf("Error rendering peers add template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		log.Printf("Created peer: %s (ID: %d)", peer.Name, peer.ID)
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
	case "delete":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := peers.Delete(s.db, peerID); err != nil {
			log.Printf("Error deleting peer: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		log.Printf("Deleted peer ID: %d", peerID)
		w.WriteHeader(http.StatusOK)
		return

	case "test":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		peer, err := peers.GetByID(s.db, peerID)
		if err != nil {
			log.Printf("Error getting peer: %v", err)
			http.Error(w, "Peer not found", http.StatusNotFound)
			return
		}

		online, err := peers.TestConnection(peer)
		if err != nil {
			log.Printf("Error testing peer connection: %v", err)
		}

		status := "offline"
		if online {
			status = "online"
		}

		// Update peer status
		if err := peers.UpdateStatus(s.db, peerID, status); err != nil {
			log.Printf("Error updating peer status: %v", err)
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

func (s *Server) handleAdminSettings(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Admin Settings Page (Coming soon)")
}

func (s *Server) handleTrash(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Trash Page (Coming soon)")
}

// handleAdminShares displays all shares for all users (admin only)
func (s *Server) handleAdminShares(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	lang := s.getLang(r)

	// Get all shares
	allShares, err := shares.GetAll(s.db)
	if err != nil {
		log.Printf("Error getting shares: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get SMB status
	smbStatus, _ := smb.GetServiceStatus()
	smbInstalled := smb.CheckSambaInstalled()

	data := struct {
		Lang         string
		Title        string
		Session      *auth.Session
		Shares       []*shares.Share
		SMBStatus    string
		SMBInstalled bool
	}{
		Lang:         lang,
		Title:        i18n.T(lang, "shares.title"),
		Session:      session,
		Shares:       allShares,
		SMBStatus:    smbStatus,
		SMBInstalled: smbInstalled,
	}

	if err := s.templates.ExecuteTemplate(w, "admin_shares.html", data); err != nil {
		log.Printf("Error rendering shares template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
