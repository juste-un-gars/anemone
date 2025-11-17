// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/juste-un-gars/anemone/internal/activation"
	"github.com/juste-un-gars/anemone/internal/auth"
	backupPkg "github.com/juste-un-gars/anemone/internal/backup"
	"github.com/juste-un-gars/anemone/internal/bulkrestore"
	"github.com/juste-un-gars/anemone/internal/config"
	"github.com/juste-un-gars/anemone/internal/crypto"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/incoming"
	"github.com/juste-un-gars/anemone/internal/peers"
	"github.com/juste-un-gars/anemone/internal/quota"
	"github.com/juste-un-gars/anemone/internal/reset"
	"github.com/juste-un-gars/anemone/internal/restore"
	"github.com/juste-un-gars/anemone/internal/serverbackup"
	"github.com/juste-un-gars/anemone/internal/shares"
	"github.com/juste-un-gars/anemone/internal/smb"
	"github.com/juste-un-gars/anemone/internal/sync"
	"github.com/juste-un-gars/anemone/internal/syncauth"
	"github.com/juste-un-gars/anemone/internal/syncconfig"
	"github.com/juste-un-gars/anemone/internal/trash"
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
	QuotaInfo      *quota.QuotaInfo
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
		"divf": func(a, b int64) float64 {
			return float64(a) / float64(b)
		},
		"FormatBytes": func(bytes int64) string {
			const unit = 1024
			if bytes < unit {
				return fmt.Sprintf("%d B", bytes)
			}
			div, exp := int64(unit), 0
			for n := bytes / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}
			return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
		},
		"FormatTime": func(t time.Time, lang string) string {
			now := time.Now()
			diff := now.Sub(t)

			if diff < time.Minute {
				if lang == "fr" {
					return "À l'instant"
				}
				return "Just now"
			}
			if diff < time.Hour {
				mins := int(diff.Minutes())
				if lang == "fr" {
					return fmt.Sprintf("Il y a %d min", mins)
				}
				return fmt.Sprintf("%d min ago", mins)
			}
			if diff < 24*time.Hour {
				hours := int(diff.Hours())
				if lang == "fr" {
					return fmt.Sprintf("Il y a %d h", hours)
				}
				return fmt.Sprintf("%d h ago", hours)
			}
			days := int(diff.Hours() / 24)
			if lang == "fr" {
				return fmt.Sprintf("Il y a %d j", days)
			}
			return fmt.Sprintf("%d d ago", days)
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

	// Password reset routes (public)
	mux.HandleFunc("/reset-password", server.handleResetPasswordForm)
	mux.HandleFunc("/reset-password/confirm", server.handleResetPasswordSubmit)

	// Restore warning routes (protected but bypass restore check)
	mux.HandleFunc("/restore-warning", auth.RequireAuth(server.handleRestoreWarning))
	mux.HandleFunc("/restore-warning/acknowledge", auth.RequireAuth(server.handleRestoreWarningAcknowledge))
	mux.HandleFunc("/restore-warning/bulk", auth.RequireAuth(server.handleRestoreWarningBulk))

	// Protected routes (with restore check)
	mux.HandleFunc("/dashboard", auth.RequireAuth(auth.RequireRestoreCheck(server.db, server.handleDashboard)))

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
	mux.HandleFunc("/admin/settings/sync-password", auth.RequireAdmin(server.handleAdminSettingsSyncPassword))

	// Admin routes - Sync
	mux.HandleFunc("/admin/sync", auth.RequireAdmin(server.handleAdminSync))
	mux.HandleFunc("/admin/sync/config", auth.RequireAdmin(server.handleAdminSyncConfig))
	mux.HandleFunc("/admin/sync/force", auth.RequireAdmin(server.handleAdminSyncForce))

	// Admin routes - Incoming backups
	mux.HandleFunc("/admin/incoming", auth.RequireAdmin(server.handleAdminIncoming))
	mux.HandleFunc("/admin/incoming/delete", auth.RequireAdmin(server.handleAdminIncomingDelete))

	// Admin routes - Server backup
	mux.HandleFunc("/admin/backup", auth.RequireAdmin(server.handleAdminBackup))
	mux.HandleFunc("/admin/backup/create", auth.RequireAdmin(server.handleAdminBackupCreate))
	mux.HandleFunc("/admin/backup/download", auth.RequireAdmin(server.handleAdminBackupDownload))
	mux.HandleFunc("/admin/backup/delete", auth.RequireAdmin(server.handleAdminBackupDelete))

	// Admin routes - Restore all users (after server restoration)
	mux.HandleFunc("/admin/restore-users", auth.RequireAdmin(server.handleAdminRestoreUsers))
	mux.HandleFunc("/admin/restore-users/restore", auth.RequireAdmin(server.handleAdminRestoreUsersRestore))

	// User routes (with restore check)
	mux.HandleFunc("/trash", auth.RequireAuth(auth.RequireRestoreCheck(server.db, server.handleTrash)))
	mux.HandleFunc("/trash/", auth.RequireAuth(auth.RequireRestoreCheck(server.db, server.handleTrashActions)))
	mux.HandleFunc("/settings", auth.RequireAuth(auth.RequireRestoreCheck(server.db, server.handleSettings)))
	mux.HandleFunc("/settings/language", auth.RequireAuth(auth.RequireRestoreCheck(server.db, server.handleSettingsLanguage)))
	mux.HandleFunc("/settings/password", auth.RequireAuth(auth.RequireRestoreCheck(server.db, server.handleSettingsPassword)))

	// Restore routes (user can restore their own backups) (with restore check)
	mux.HandleFunc("/restore", auth.RequireAuth(auth.RequireRestoreCheck(server.db, server.handleRestore)))
	mux.HandleFunc("/api/restore/backups", auth.RequireAuth(auth.RequireRestoreCheck(server.db, server.handleAPIRestoreBackups)))
	mux.HandleFunc("/api/restore/files", auth.RequireAuth(auth.RequireRestoreCheck(server.db, server.handleAPIRestoreFiles)))
	mux.HandleFunc("/api/restore/download", auth.RequireAuth(auth.RequireRestoreCheck(server.db, server.handleAPIRestoreDownload)))
	mux.HandleFunc("/api/restore/download-multiple", auth.RequireAuth(auth.RequireRestoreCheck(server.db, server.handleAPIRestoreDownloadMultiple)))

	// Admin routes - Shares
	mux.HandleFunc("/admin/shares", auth.RequireAdmin(server.handleAdminShares))

	// Sync routes
	mux.HandleFunc("/sync/share/", auth.RequireAuth(server.handleSyncShare))

	// API routes - Sync (protected by password authentication)
	mux.HandleFunc("/api/sync/receive", server.syncAuthMiddleware(server.handleAPISyncReceive))

	// API routes - Incremental sync (manifest-based, protected by password authentication)
	mux.HandleFunc("/api/sync/manifest", server.syncAuthMiddleware(server.handleAPISyncManifest))       // GET/PUT
	mux.HandleFunc("/api/sync/file", server.syncAuthMiddleware(server.handleAPISyncFile))               // POST/DELETE

	// API routes - Remote restore (protected by password authentication)
	mux.HandleFunc("/api/sync/list-user-backups", server.syncAuthMiddleware(server.handleAPISyncListUserBackups))
	mux.HandleFunc("/api/sync/download-encrypted-manifest", server.syncAuthMiddleware(server.handleAPISyncDownloadEncryptedManifest))
	mux.HandleFunc("/api/sync/download-encrypted-file", server.syncAuthMiddleware(server.handleAPISyncDownloadEncryptedFile))

	return mux
}

// isSetupCompleted checks if initial setup is done
func (s *Server) isSetupCompleted() bool {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM system_config WHERE key = 'setup_completed'").Scan(&count)
	return err == nil && count > 0
}

// syncAuthMiddleware checks for sync authentication password in X-Sync-Password header
// This middleware protects /api/sync/* endpoints from unauthorized access
func (s *Server) syncAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if sync auth password is configured
		isConfigured, err := syncauth.IsConfigured(s.db)
		if err != nil {
			log.Printf("Error checking sync auth config: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// If no password is configured, allow access (backward compatibility)
		if !isConfigured {
			next(w, r)
			return
		}

		// Get password from header
		password := r.Header.Get("X-Sync-Password")
		if password == "" {
			log.Printf("Sync auth failed: No X-Sync-Password header from %s", r.RemoteAddr)
			http.Error(w, "Unauthorized: X-Sync-Password header required", http.StatusUnauthorized)
			return
		}

		// Check if password is correct
		valid, err := syncauth.CheckSyncAuthPassword(s.db, password)
		if err != nil {
			log.Printf("Error checking sync auth password: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if !valid {
			log.Printf("Sync auth failed: Invalid password from %s", r.RemoteAddr)
			http.Error(w, "Forbidden: Invalid password", http.StatusForbidden)
			return
		}

		// Password is valid, continue to handler
		next(w, r)
	}
}

// getLang gets language from user preference (DB), query param, or config
func (s *Server) getLang(r *http.Request) string {
	lang := ""

	// Priority 1: User language preference from database (if logged in)
	if session, ok := auth.GetSessionFromContext(r); ok {
		user, err := users.GetByID(s.db, session.UserID)
		if err == nil && user.Language != "" {
			lang = user.Language
		}
	}

	// Priority 2: Query parameter (e.g., ?lang=en)
	if lang == "" {
		if l := r.URL.Query().Get("lang"); l != "" {
			lang = l
		}
	}

	// Priority 3: Config (environment variable or default)
	if lang == "" {
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
		StorageUsed:    "0 B",
		StorageQuota:   "∞",
		StoragePercent: 0,
		LastBackup:     "Jamais",
		UserCount:      0,
		PeerCount:      0,
		TrashCount:     0,
	}

	// Admin stats: all users
	if session.IsAdmin {
		s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.UserCount)
		s.db.QueryRow("SELECT COUNT(*) FROM peers").Scan(&stats.PeerCount)

		// Calculate total storage used by all users
		var totalUsedGB float64
		rows, err := s.db.Query("SELECT id FROM users")
		if err != nil {
			log.Printf("Error querying users for storage stats: %v", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var userID int
				if err := rows.Scan(&userID); err != nil {
					continue
				}
				quotaInfo, err := quota.GetUserQuota(s.db, userID)
				if err != nil {
					continue
				}
				totalUsedGB += quotaInfo.UsedTotalGB
			}
			stats.StorageUsed = fmt.Sprintf("%.2f GB", totalUsedGB)
		}

		// Get last backup time from any user
		var lastSync sql.NullTime
		err = s.db.QueryRow(`
			SELECT completed_at
			FROM sync_log
			WHERE status = 'success'
			ORDER BY completed_at DESC
			LIMIT 1
		`).Scan(&lastSync)

		if err == nil && lastSync.Valid {
			duration := time.Since(lastSync.Time)
			if duration < time.Hour {
				stats.LastBackup = fmt.Sprintf("Il y a %d minutes", int(duration.Minutes()))
			} else if duration < 24*time.Hour {
				stats.LastBackup = fmt.Sprintf("Il y a %d heures", int(duration.Hours()))
			} else {
				stats.LastBackup = fmt.Sprintf("Il y a %d jours", int(duration.Hours()/24))
			}
		}
	} else {
		// User stats: personal quota
		quotaInfo, err := quota.GetUserQuota(s.db, session.UserID)
		if err != nil {
			log.Printf("Error getting quota info: %v", err)
		} else {
			stats.QuotaInfo = quotaInfo
			stats.StorageUsed = fmt.Sprintf("%.2f GB", quotaInfo.UsedTotalGB)
			stats.StorageQuota = fmt.Sprintf("%d GB", quotaInfo.QuotaTotalGB)
			stats.StoragePercent = int(quotaInfo.PercentUsed)
		}

		// Get user's shares for trash count
		userShares, err := shares.GetByUser(s.db, session.UserID)
		if err != nil {
			log.Printf("Error getting shares for stats: %v", err)
			return stats
		}

		// Count trash items
		totalTrashCount := 0
		for _, share := range userShares {
			username := session.Username
			trashItems, err := trash.ListTrashItems(share.Path, username)
			if err != nil {
				log.Printf("Error listing trash for share %s: %v", share.Name, err)
				continue
			}
			totalTrashCount += len(trashItems)
		}
		stats.TrashCount = totalTrashCount

		// Get last backup time from sync_log
		var lastSync sql.NullTime
		err = s.db.QueryRow(`
			SELECT completed_at
			FROM sync_log
			WHERE user_id = ? AND status = 'success'
			ORDER BY completed_at DESC
			LIMIT 1
		`, session.UserID).Scan(&lastSync)

		if err == nil && lastSync.Valid {
			duration := time.Since(lastSync.Time)
			if duration < time.Hour {
				stats.LastBackup = fmt.Sprintf("Il y a %d minutes", int(duration.Minutes()))
			} else if duration < 24*time.Hour {
				stats.LastBackup = fmt.Sprintf("Il y a %d heures", int(duration.Hours()))
			} else {
				stats.LastBackup = fmt.Sprintf("Il y a %d jours", int(duration.Hours()/24))
			}
		}
	}

	return stats
}

// calculateDirectorySize calculates the total size of a directory
func calculateDirectorySize(path string) int64 {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				size += info.Size()
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("Error calculating directory size for %s: %v", path, err)
	}
	return size
}

// formatBytes formats bytes to human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
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
		quotaDataStr := r.FormValue("quota_data")
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

	case "reset":
		// Generate password reset token
		user, err := users.GetByID(s.db, userID)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			http.NotFound(w, r)
			return
		}

		// Create reset token
		token, err := reset.CreatePasswordResetToken(s.db, user.ID)
		if err != nil {
			log.Printf("Error creating reset token: %v", err)
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
			Lang      string
			Title     string
			Session   *auth.Session
			Username  string
			Email     string
			ResetURL  string
			ExpiresAt time.Time
			T         func(string) string
		}{
			Lang:      lang,
			Title:     i18n.T(lang, "reset.token.title"),
			Session:   session,
			Username:  user.Username,
			Email:     user.Email,
			ResetURL:  resetURL,
			ExpiresAt: token.ExpiresAt,
		}

		if err := s.templates.ExecuteTemplate(w, "admin_users_reset_token.html", data); err != nil {
			log.Printf("Error rendering reset token template: %v", err)
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

		err := users.DeleteUser(s.db, userID, s.cfg.DataDir)
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

		// Get user info to retrieve quotas
		user, err := users.GetByID(s.db, token.UserID)
		if err != nil {
			log.Printf("Warning: Failed to get user info: %v", err)
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
			log.Printf("Warning: Failed to initialize quota manager: %v", err)
			qm = nil // Will create regular directories as fallback
		}

		// Create default shares: backup and data with quota enforcement
		backupPath := filepath.Join(s.cfg.SharesDir, token.Username, "backup")
		if qm != nil {
			if err := qm.CreateQuotaDir(backupPath, user.QuotaBackupGB); err != nil {
				log.Printf("Warning: Failed to create backup quota directory: %v", err)
			} else {
				log.Printf("Created backup subvolume with %dGB quota", user.QuotaBackupGB)

				// Set ownership of subvolume to user (needed for .trash creation)
				chownCmd := exec.Command("sudo", "/usr/bin/chown", "-R", fmt.Sprintf("%s:%s", token.Username, token.Username), backupPath)
				if err := chownCmd.Run(); err != nil {
					log.Printf("Warning: Failed to set backup subvolume ownership: %v", err)
				}
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
			log.Printf("Warning: Failed to create backup share: %v", err)
		} else {
			log.Printf("Created backup share: backup_%s", token.Username)
		}

		dataPath := filepath.Join(s.cfg.SharesDir, token.Username, "data")
		if qm != nil {
			if err := qm.CreateQuotaDir(dataPath, dataQuotaGB); err != nil {
				log.Printf("Warning: Failed to create data quota directory: %v", err)
			} else {
				log.Printf("Created data subvolume with %dGB quota", dataQuotaGB)

				// Set ownership of subvolume to user (needed for .trash creation)
				chownCmd := exec.Command("sudo", "/usr/bin/chown", "-R", fmt.Sprintf("%s:%s", token.Username, token.Username), dataPath)
				if err := chownCmd.Run(); err != nil {
					log.Printf("Warning: Failed to set data subvolume ownership: %v", err)
				}
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
			log.Printf("Warning: Failed to create data share: %v", err)
		} else {
			log.Printf("Created data share: data_%s", token.Username)
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
			log.Printf("Warning: Failed to regenerate SMB config: %v", err)
		} else {
			// Try to reload smbd (requires sudoers configuration)
			if err := smb.ReloadConfig(); err != nil {
				log.Printf("Warning: Could not reload smbd automatically. Run: sudo systemctl reload smbd")
			} else {
				log.Printf("✅ SMB config reloaded successfully")
			}
		}

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
		log.Printf("Token not found: %v", err)
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
			log.Printf("Error rendering reset password template: %v", err)
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
			log.Printf("Error rendering reset password template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Get user info
	user, err := users.GetByID(s.db, token.UserID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Show form
	data := struct {
		Lang     string
		Title    string
		Token    string
		Username string
		Error    string
	}{
		Lang:     lang,
		Title:    i18n.T(lang, "reset.title"),
		Token:    tokenString,
		Username: user.Username,
		Error:    r.URL.Query().Get("error"),
	}

	if err := s.templates.ExecuteTemplate(w, "reset_password.html", data); err != nil {
		log.Printf("Error rendering reset password template: %v", err)
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
		log.Printf("Error getting user: %v", err)
		http.Redirect(w, r, fmt.Sprintf("/reset-password?token=%s&error=User+not+found", tokenString), http.StatusSeeOther)
		return
	}

	// Get master key
	var masterKey string
	err = s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		log.Printf("Error getting master key: %v", err)
		http.Redirect(w, r, fmt.Sprintf("/reset-password?token=%s&error=System+configuration+error", tokenString), http.StatusSeeOther)
		return
	}

	// Reset password (update DB + SMB)
	err = users.ResetPassword(s.db, user.ID, user.Username, newPassword, masterKey)
	if err != nil {
		log.Printf("Error resetting password: %v", err)
		http.Redirect(w, r, fmt.Sprintf("/reset-password?token=%s&error=Failed+to+reset+password", tokenString), http.StatusSeeOther)
		return
	}

	// Mark token as used
	if err := token.MarkAsUsed(s.db); err != nil {
		log.Printf("Error marking token as used: %v", err)
		// Non-critical, continue
	}

	log.Printf("Password reset successfully for user: %s", user.Username)

	// Redirect to login with success message
	http.Redirect(w, r, "/login?success=Password+reset+successfully", http.StatusSeeOther)
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
		password := r.FormValue("password")
		enabled := r.FormValue("enabled") == "on"

		// Parse sync configuration
		syncEnabled := r.FormValue("sync_enabled") == "on"
		syncFrequency := r.FormValue("sync_frequency")
		syncTime := r.FormValue("sync_time")
		syncDayOfWeekStr := r.FormValue("sync_day_of_week")
		syncDayOfMonthStr := r.FormValue("sync_day_of_month")

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

		// Parse sync day of week/month
		var syncDayOfWeekPtr *int
		if syncDayOfWeekStr != "" && syncFrequency == "weekly" {
			dayOfWeek, err := strconv.Atoi(syncDayOfWeekStr)
			if err == nil && dayOfWeek >= 0 && dayOfWeek <= 6 {
				syncDayOfWeekPtr = &dayOfWeek
			}
		}
		var syncDayOfMonthPtr *int
		if syncDayOfMonthStr != "" && syncFrequency == "monthly" {
			dayOfMonth, err := strconv.Atoi(syncDayOfMonthStr)
			if err == nil && dayOfMonth >= 1 && dayOfMonth <= 31 {
				syncDayOfMonthPtr = &dayOfMonth
			}
		}

		// Parse sync interval (convert to minutes)
		syncIntervalMinutes := 60 // Default: 1 hour
		if syncFrequency == "interval" {
			intervalValueStr := r.FormValue("sync_interval_value")
			intervalUnit := r.FormValue("sync_interval_unit")

			if intervalValueStr != "" {
				intervalValue, err := strconv.Atoi(intervalValueStr)
				if err == nil && intervalValue > 0 {
					if intervalUnit == "hours" {
						syncIntervalMinutes = intervalValue * 60
					} else {
						syncIntervalMinutes = intervalValue
					}
				}
			}
		}

		// Create peer
		var pkPtr *string
		if publicKey != "" {
			pkPtr = &publicKey
		}
		var pwPtr *string
		if password != "" {
			pwPtr = &password
		}
		peer := &peers.Peer{
			Name:               name,
			Address:            address,
			Port:               port,
			PublicKey:          pkPtr,
			Password:           pwPtr,
			Enabled:            enabled,
			Status:             "unknown",
			SyncEnabled:        syncEnabled,
			SyncFrequency:      syncFrequency,
			SyncTime:           syncTime,
			SyncDayOfWeek:      syncDayOfWeekPtr,
			SyncDayOfMonth:     syncDayOfMonthPtr,
			SyncIntervalMinutes: syncIntervalMinutes,
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
	case "edit":
		// Display edit form (GET)
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		session, ok := auth.GetSessionFromContext(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		peer, err := peers.GetByID(s.db, peerID)
		if err != nil {
			log.Printf("Error getting peer: %v", err)
			http.Error(w, "Peer not found", http.StatusNotFound)
			return
		}

		data := struct {
			Lang    string
			Session *auth.Session
			Peer    *peers.Peer
			Error   string
		}{
			Lang:    s.cfg.Language,
			Session: session,
			Peer:    peer,
			Error:   r.URL.Query().Get("error"),
		}

		if err := s.templates.ExecuteTemplate(w, "admin_peers_edit.html", data); err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
		return

	case "update":
		// Process edit form (POST)
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		session, ok := auth.GetSessionFromContext(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, fmt.Sprintf("/admin/peers/%d/edit?error=Invalid+form+data", peerID), http.StatusSeeOther)
			return
		}

		// Get existing peer
		peer, err := peers.GetByID(s.db, peerID)
		if err != nil {
			log.Printf("Error getting peer: %v", err)
			http.Redirect(w, r, "/admin/peers?error=Peer+not+found", http.StatusSeeOther)
			return
		}

		// Update fields
		peer.Name = r.FormValue("name")
		peer.Address = r.FormValue("address")

		port, err := strconv.Atoi(r.FormValue("port"))
		if err != nil || port < 1 || port > 65535 {
			http.Redirect(w, r, fmt.Sprintf("/admin/peers/%d/edit?error=Invalid+port", peerID), http.StatusSeeOther)
			return
		}
		peer.Port = port

		// Update password if provided
		password := r.FormValue("password")
		if password != "" {
			peer.Password = &password
		} else if r.FormValue("clear_password") == "1" {
			peer.Password = nil
		}
		// If password is empty and clear_password is not checked, keep existing password

		// Update enabled status
		peer.Enabled = r.FormValue("enabled") == "1"

		// Update sync configuration
		peer.SyncEnabled = r.FormValue("sync_enabled") == "on"
		peer.SyncFrequency = r.FormValue("sync_frequency")
		peer.SyncTime = r.FormValue("sync_time")

		// Parse sync day of week/month
		syncDayOfWeekStr := r.FormValue("sync_day_of_week")
		syncDayOfMonthStr := r.FormValue("sync_day_of_month")

		var syncDayOfWeekPtr *int
		if syncDayOfWeekStr != "" && peer.SyncFrequency == "weekly" {
			dayOfWeek, err := strconv.Atoi(syncDayOfWeekStr)
			if err == nil && dayOfWeek >= 0 && dayOfWeek <= 6 {
				syncDayOfWeekPtr = &dayOfWeek
			}
		}
		peer.SyncDayOfWeek = syncDayOfWeekPtr

		var syncDayOfMonthPtr *int
		if syncDayOfMonthStr != "" && peer.SyncFrequency == "monthly" {
			dayOfMonth, err := strconv.Atoi(syncDayOfMonthStr)
			if err == nil && dayOfMonth >= 1 && dayOfMonth <= 31 {
				syncDayOfMonthPtr = &dayOfMonth
			}
		}
		peer.SyncDayOfMonth = syncDayOfMonthPtr

		// Parse sync interval (convert to minutes)
		if peer.SyncFrequency == "interval" {
			intervalValueStr := r.FormValue("sync_interval_value")
			intervalUnit := r.FormValue("sync_interval_unit")

			if intervalValueStr != "" {
				intervalValue, err := strconv.Atoi(intervalValueStr)
				if err == nil && intervalValue > 0 {
					if intervalUnit == "hours" {
						peer.SyncIntervalMinutes = intervalValue * 60
					} else {
						peer.SyncIntervalMinutes = intervalValue
					}
				} else {
					peer.SyncIntervalMinutes = 60 // Default: 1 hour
				}
			} else {
				peer.SyncIntervalMinutes = 60 // Default: 1 hour
			}
		}

		// Save to database
		if err := peers.Update(s.db, peer); err != nil {
			log.Printf("Error updating peer: %v", err)
			http.Redirect(w, r, fmt.Sprintf("/admin/peers/%d/edit?error=Failed+to+update+peer", peerID), http.StatusSeeOther)
			return
		}

		log.Printf("Admin %s updated peer ID %d: %s", session.Username, peerID, peer.Name)
		http.Redirect(w, r, "/admin/peers", http.StatusSeeOther)
		return

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

func (s *Server) handleTrash(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Get user info
	user, err := users.GetByID(s.db, session.UserID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get user's shares
	userShares, err := shares.GetByUser(s.db, session.UserID)
	if err != nil {
		log.Printf("Error getting shares: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Collect all trash items from all user's shares
	type TrashItemWithShare struct {
		*trash.TrashItem
		ShareName string
	}

	var allTrashItems []TrashItemWithShare
	for _, share := range userShares {
		items, err := trash.ListTrashItems(share.Path, user.Username)
		if err != nil {
			log.Printf("Error listing trash for share %s: %v", share.Name, err)
			continue
		}

		for _, item := range items {
			allTrashItems = append(allTrashItems, TrashItemWithShare{
				TrashItem: item,
				ShareName: share.Name,
			})
		}
	}

	data := struct {
		Lang    string
		Title   string
		Session *auth.Session
		Items   []TrashItemWithShare
	}{
		Lang:    lang,
		Title:   i18n.T(lang, "trash.title"),
		Session: session,
		Items:   allTrashItems,
	}

	if err := s.templates.ExecuteTemplate(w, "trash.html", data); err != nil {
		log.Printf("Error rendering trash template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleTrashActions handles trash item actions (restore, delete)
func (s *Server) handleTrashActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse URL: /trash/{action}?share={shareName}&path={relPath}
	path := strings.TrimPrefix(r.URL.Path, "/trash/")
	action := path

	// Get parameters
	shareName := r.URL.Query().Get("share")
	relPath := r.URL.Query().Get("path")

	if shareName == "" || relPath == "" {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}

	// Get user
	user, err := users.GetByID(s.db, session.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Find the share
	userShares, err := shares.GetByUser(s.db, session.UserID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var targetShare *shares.Share
	for _, share := range userShares {
		if share.Name == shareName {
			targetShare = share
			break
		}
	}

	if targetShare == nil {
		http.Error(w, "Share not found", http.StatusNotFound)
		return
	}

	// Execute action
	switch action {
	case "restore":
		err = trash.RestoreItem(targetShare.Path, user.Username, relPath)
		if err != nil {
			log.Printf("Error restoring item: %v", err)
			http.Error(w, fmt.Sprintf("Failed to restore: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("User %s restored file: %s from %s", user.Username, relPath, shareName)
		w.WriteHeader(http.StatusOK)

	case "delete":
		err = trash.DeleteItem(targetShare.Path, user.Username, relPath)
		if err != nil {
			log.Printf("Error deleting item: %v", err)
			http.Error(w, fmt.Sprintf("Failed to delete: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("User %s permanently deleted file: %s from %s", user.Username, relPath, shareName)
		w.WriteHeader(http.StatusOK)

	case "empty":
		// Empty entire trash for this share
		err = trash.EmptyTrash(targetShare.Path, user.Username)
		if err != nil {
			log.Printf("Error emptying trash: %v", err)
			http.Error(w, fmt.Sprintf("Failed to empty trash: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("User %s emptied trash for share %s", user.Username, shareName)
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Unknown action", http.StatusBadRequest)
	}
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

// handleSyncShare triggers manual synchronization of a share to all enabled peers
func (s *Server) handleSyncShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract share ID from URL: /sync/share/{id}
	path := strings.TrimPrefix(r.URL.Path, "/sync/share/")
	shareID, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid share ID", http.StatusBadRequest)
		return
	}

	// Get share
	share, err := shares.GetByID(s.db, shareID)
	if err != nil {
		log.Printf("Error getting share: %v", err)
		http.Error(w, "Share not found", http.StatusNotFound)
		return
	}

	// Check if user has permission (either owner or admin)
	if !session.IsAdmin && share.UserID != session.UserID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Check if sync is enabled for this share
	if !share.SyncEnabled {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"success": false, "message": "Synchronisation non activée pour ce partage"}`)
		return
	}

	// Get all enabled peers
	allPeers, err := peers.GetAll(s.db)
	if err != nil {
		log.Printf("Error getting peers: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	enabledPeers := []*peers.Peer{}
	for _, peer := range allPeers {
		if peer.Enabled {
			enabledPeers = append(enabledPeers, peer)
		}
	}

	if len(enabledPeers) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"success": false, "message": "Aucun pair actif configuré"}`)
		return
	}

	// Synchronize to each enabled peer
	successCount := 0
	errorCount := 0
	var lastError string

	for _, peer := range enabledPeers {
		req := &sync.SyncRequest{
			ShareID:     shareID,
			PeerID:      peer.ID,
			UserID:      share.UserID,
			SharePath:   share.Path,
			PeerAddress: peer.Address,
			PeerPort:    peer.Port,
		}

		// Use incremental sync (manifest-based)
		err := sync.SyncShareIncremental(s.db, req)
		if err != nil {
			errorCount++
			lastError = err.Error()
			log.Printf("Error syncing to peer %s: %v", peer.Name, err)
		} else {
			successCount++
			log.Printf("Successfully synced share %d to peer %s (incremental)", shareID, peer.Name)
		}
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if errorCount > 0 {
		fmt.Fprintf(w, `{"success": false, "message": "Synchronisation partielle: %d réussis, %d échecs. Dernière erreur: %s"}`,
			successCount, errorCount, lastError)
	} else {
		fmt.Fprintf(w, `{"success": true, "message": "Synchronisation réussie vers %d pair(s)"}`, successCount)
	}
}

// handleAPISyncReceive receives and extracts a share archive from a peer
func (s *Server) handleAPISyncReceive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 10GB)
	if err := r.ParseMultipartForm(10 << 30); err != nil {
		log.Printf("Error parsing multipart form: %v", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get metadata from form
	userIDStr := r.FormValue("user_id")
	shareName := r.FormValue("share_name")

	if userIDStr == "" || shareName == "" {
		http.Error(w, "Missing user_id or share_name", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Find matching share in local database
	userShares, err := shares.GetByUser(s.db, userID)
	if err != nil {
		log.Printf("Error getting user shares: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var targetShare *shares.Share
	for _, share := range userShares {
		if share.Name == shareName || share.Name == "backup_"+shareName || shareName == "backup" {
			targetShare = share
			break
		}
	}

	if targetShare == nil {
		log.Printf("No matching share found for user %d, share %s", userID, shareName)
		http.Error(w, "Share not found", http.StatusNotFound)
		return
	}

	// Get archive file
	file, _, err := r.FormFile("archive")
	if err != nil {
		log.Printf("Error getting archive file: %v", err)
		http.Error(w, "Missing archive file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check if archive is encrypted
	encrypted := r.FormValue("encrypted") == "true"

	var reader io.Reader = file
	if encrypted {
		// Get user's encryption key
		encryptionKey, err := sync.GetUserEncryptionKey(s.db, userID)
		if err != nil {
			log.Printf("Error getting encryption key: %v", err)
			http.Error(w, "Failed to get encryption key", http.StatusInternalServerError)
			return
		}

		// Decrypt archive
		var decryptedBuf bytes.Buffer
		if err := crypto.DecryptStream(file, &decryptedBuf, encryptionKey); err != nil {
			log.Printf("Error decrypting archive: %v", err)
			http.Error(w, fmt.Sprintf("Failed to decrypt archive: %v", err), http.StatusInternalServerError)
			return
		}
		reader = &decryptedBuf
	}

	// Extract archive to local share path
	if err := sync.ExtractTarGz(reader, targetShare.Path); err != nil {
		log.Printf("Error extracting archive: %v", err)
		http.Error(w, fmt.Sprintf("Failed to extract archive: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully received and extracted sync to: %s (user %d, share %s)", targetShare.Path, userID, shareName)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "message": "Sync received and extracted"}`)
}

// handleAPISyncManifest handles GET and PUT requests for sync manifests
func (s *Server) handleAPISyncManifest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleAPISyncManifestGet(w, r)
	case http.MethodPut:
		s.handleAPISyncManifestPut(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAPISyncManifestGet returns the manifest for a given share
// GET /api/sync/manifest?user_id=5&share_name=backup
func (s *Server) handleAPISyncManifestGet(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	userIDStr := r.URL.Query().Get("user_id")
	shareName := r.URL.Query().Get("share_name")

	if userIDStr == "" || shareName == "" {
		http.Error(w, "Missing user_id or share_name", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Build backup directory path directly (no need to check if user exists locally)
	// Format: /srv/anemone/backups/incoming/{user_id}_{share_name}/
	backupDirName := fmt.Sprintf("%d_%s", userID, shareName)
	backupDir := filepath.Join("/srv/anemone/backups/incoming", backupDirName)
	manifestPath := filepath.Join(backupDir, ".anemone-manifest.json.enc")

	// Check if manifest file exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		// No manifest yet (first sync) - return 404
		http.Error(w, "No manifest found (first sync)", http.StatusNotFound)
		return
	}

	// Read encrypted manifest
	encryptedData, err := os.ReadFile(manifestPath)
	if err != nil {
		log.Printf("Error reading manifest file: %v", err)
		http.Error(w, "Failed to read manifest", http.StatusInternalServerError)
		return
	}

	// Return encrypted manifest
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\".anemone-manifest.json.enc\"")
	w.WriteHeader(http.StatusOK)
	w.Write(encryptedData)
}

// handleAPISyncManifestPut updates the manifest for a given share
// PUT /api/sync/manifest?user_id=5&share_name=backup
// Body: encrypted manifest data
func (s *Server) handleAPISyncManifestPut(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	userIDStr := r.URL.Query().Get("user_id")
	shareName := r.URL.Query().Get("share_name")

	if userIDStr == "" || shareName == "" {
		http.Error(w, "Missing user_id or share_name", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Read encrypted manifest from request body
	encryptedData, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Failed to read manifest data", http.StatusBadRequest)
		return
	}

	// Build backup directory path directly (no need to check if user exists locally)
	backupDirName := fmt.Sprintf("%d_%s", userID, shareName)
	backupDir := filepath.Join("/srv/anemone/backups/incoming", backupDirName)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		log.Printf("Error creating backup directory: %v", err)
		http.Error(w, "Failed to create backup directory", http.StatusInternalServerError)
		return
	}

	// Write encrypted manifest
	manifestPath := filepath.Join(backupDir, ".anemone-manifest.json.enc")
	if err := os.WriteFile(manifestPath, encryptedData, 0644); err != nil {
		log.Printf("Error writing manifest file: %v", err)
		http.Error(w, "Failed to write manifest", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully updated manifest for user %d, share %s", userID, shareName)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "message": "Manifest updated"}`)
}

// handleAPISyncFile handles POST and DELETE requests for individual files
func (s *Server) handleAPISyncFile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleAPISyncFileUpload(w, r)
	case http.MethodDelete:
		s.handleAPISyncFileDelete(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAPISyncFileUpload handles uploading a single encrypted file
// POST /api/sync/file
// Multipart form with: user_id, share_name, relative_path, file
func (s *Server) handleAPISyncFileUpload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 10GB)
	if err := r.ParseMultipartForm(10 << 30); err != nil {
		log.Printf("Error parsing multipart form: %v", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get metadata from form
	userIDStr := r.FormValue("user_id")
	shareName := r.FormValue("share_name")
	relativePath := r.FormValue("relative_path")

	if userIDStr == "" || shareName == "" || relativePath == "" {
		http.Error(w, "Missing user_id, share_name, or relative_path", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Security check: prevent path traversal
	if strings.Contains(relativePath, "..") {
		http.Error(w, "Invalid relative_path (path traversal detected)", http.StatusBadRequest)
		return
	}

	// Get file from multipart form
	file, _, err := r.FormFile("file")
	if err != nil {
		log.Printf("Error getting file: %v", err)
		http.Error(w, "Missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Build backup directory path directly (no need to check if user exists locally)
	backupDirName := fmt.Sprintf("%d_%s", userID, shareName)
	backupDir := filepath.Join("/srv/anemone/backups/incoming", backupDirName)
	targetPath := filepath.Join(backupDir, relativePath)

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		log.Printf("Error creating directory: %v", err)
		http.Error(w, "Failed to create directory", http.StatusInternalServerError)
		return
	}

	// Write file to disk
	outFile, err := os.Create(targetPath)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, file); err != nil {
		log.Printf("Error writing file: %v", err)
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully uploaded file: %s (user %d, share %s)", relativePath, userID, shareName)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "message": "File uploaded"}`)
}

// handleAPISyncFileDelete handles deleting a single file from backup
// DELETE /api/sync/file?user_id=5&share_name=backup&path=documents/report.pdf.enc
func (s *Server) handleAPISyncFileDelete(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	userIDStr := r.URL.Query().Get("user_id")
	shareName := r.URL.Query().Get("share_name")
	relativePath := r.URL.Query().Get("path")

	if userIDStr == "" || shareName == "" || relativePath == "" {
		http.Error(w, "Missing user_id, share_name, or path", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Security check: prevent path traversal
	if strings.Contains(relativePath, "..") {
		http.Error(w, "Invalid path (path traversal detected)", http.StatusBadRequest)
		return
	}

	// Build backup directory path directly (no need to check if user exists locally)
	backupDirName := fmt.Sprintf("%d_%s", userID, shareName)
	backupDir := filepath.Join("/srv/anemone/backups/incoming", backupDirName)
	targetPath := filepath.Join(backupDir, relativePath)

	// Delete file
	if err := os.Remove(targetPath); err != nil {
		if os.IsNotExist(err) {
			// File already doesn't exist - that's OK
			log.Printf("File already deleted: %s", relativePath)
		} else {
			log.Printf("Error deleting file: %v", err)
			http.Error(w, "Failed to delete file", http.StatusInternalServerError)
			return
		}
	}

	log.Printf("Successfully deleted file: %s (user %d, share %s)", relativePath, userID, shareName)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "message": "File deleted"}`)
}

// handleSettings shows the user settings page
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.GetSessionFromContext(r)

	// Get user from database
	user, err := users.GetByID(s.db, session.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	data := struct {
		Lang    string
		Title   string
		Session *auth.Session
		User    *users.User
		Success string
		Error   string
	}{
		Lang:    s.getLang(r),
		Title:   "Settings",
		Session: session,
		User:    user,
		Success: r.URL.Query().Get("success"),
		Error:   r.URL.Query().Get("error"),
	}

	if err := s.templates.ExecuteTemplate(w, "settings.html", data); err != nil {
		log.Printf("Error executing settings template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleSettingsLanguage handles language change
func (s *Server) handleSettingsLanguage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, _ := auth.GetSessionFromContext(r)

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/settings?error=Invalid+form+data", http.StatusSeeOther)
		return
	}

	language := r.FormValue("language")

	// Update user language in database
	if err := users.UpdateUserLanguage(s.db, session.UserID, language); err != nil {
		log.Printf("Error updating user language: %v", err)
		http.Redirect(w, r, "/settings?error=Failed+to+update+language", http.StatusSeeOther)
		return
	}

	// Redirect back to settings with success message
	successMsg := "Language changed successfully"
	if language == "fr" {
		successMsg = "Langue modifiée avec succès"
	}
	http.Redirect(w, r, "/settings?success="+successMsg, http.StatusSeeOther)
}

// handleSettingsPassword handles password change
func (s *Server) handleSettingsPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, _ := auth.GetSessionFromContext(r)

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/settings?error=Invalid+form+data", http.StatusSeeOther)
		return
	}

	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate inputs
	if currentPassword == "" || newPassword == "" || confirmPassword == "" {
		http.Redirect(w, r, "/settings?error=All+fields+are+required", http.StatusSeeOther)
		return
	}

	if newPassword != confirmPassword {
		http.Redirect(w, r, "/settings?error=New+passwords+do+not+match", http.StatusSeeOther)
		return
	}

	if len(newPassword) < 8 {
		http.Redirect(w, r, "/settings?error=Password+must+be+at+least+8+characters", http.StatusSeeOther)
		return
	}

	// Get master key
	var masterKey string
	err := s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		log.Printf("Error getting master key: %v", err)
		http.Redirect(w, r, "/settings?error=System+configuration+error", http.StatusSeeOther)
		return
	}

	// Change password (DB + SMB)
	if err := users.ChangePassword(s.db, session.UserID, currentPassword, newPassword, masterKey); err != nil {
		log.Printf("Error changing password for user %d: %v", session.UserID, err)

		// Check for specific error messages
		errMsg := err.Error()
		if errMsg == "incorrect current password" {
			http.Redirect(w, r, "/settings?error=Incorrect+current+password", http.StatusSeeOther)
		} else if errMsg == "new password must be at least 8 characters" {
			http.Redirect(w, r, "/settings?error=Password+must+be+at+least+8+characters", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/settings?error=Failed+to+change+password", http.StatusSeeOther)
		}
		return
	}

	// Get user to determine language for success message
	user, _ := users.GetByID(s.db, session.UserID)
	successMsg := "Password changed successfully"
	if user != nil && user.Language == "fr" {
		successMsg = "Mot de passe modifié avec succès"
	}

	http.Redirect(w, r, "/settings?success="+successMsg, http.StatusSeeOther)
}

// handleAdminUserQuota handles quota management for a user
func (s *Server) handleAdminUserQuota(w http.ResponseWriter, r *http.Request, userID int, session *auth.Session, lang string) {
	if r.Method == http.MethodGet {
		// Display quota edit form
		user, err := users.GetByID(s.db, userID)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			http.NotFound(w, r)
			return
		}

		// Get quota info
		quotaInfo, err := quota.GetUserQuota(s.db, userID)
		if err != nil {
			log.Printf("Error getting quota info: %v", err)
			http.Error(w, "Failed to get quota information", http.StatusInternalServerError)
			return
		}

		data := struct {
			Lang      string
			Title     string
			Session   *auth.Session
			User      *users.User
			QuotaInfo *quota.QuotaInfo
		}{
			Lang:      lang,
			Title:     i18n.T(lang, "users.quota.title"),
			Session:   session,
			User:      user,
			QuotaInfo: quotaInfo,
		}

		if err := s.templates.ExecuteTemplate(w, "admin_users_quota.html", data); err != nil {
			log.Printf("Error rendering quota template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	} else if r.Method == http.MethodPost {
		// Update quotas
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		quotaBackupGB, err := strconv.Atoi(r.FormValue("quota_backup_gb"))
		if err != nil || quotaBackupGB < 1 {
			http.Error(w, "Invalid backup quota", http.StatusBadRequest)
			return
		}

		quotaDataGB, err := strconv.Atoi(r.FormValue("quota_data_gb"))
		if err != nil || quotaDataGB < 1 {
			http.Error(w, "Invalid data quota", http.StatusBadRequest)
			return
		}

		// Calculate total (backup + data)
		quotaTotalGB := quotaBackupGB + quotaDataGB

		// Update quota in database
		if err := quota.UpdateUserQuota(s.db, userID, quotaTotalGB, quotaBackupGB); err != nil {
			log.Printf("Error updating quota: %v", err)
			http.Error(w, "Failed to update quota", http.StatusInternalServerError)
			return
		}

		// Update Btrfs quotas for user shares
		user, err := users.GetByID(s.db, userID)
		if err == nil {
			backupPath := filepath.Join(s.cfg.SharesDir, user.Username, "backup")
			dataPath := filepath.Join(s.cfg.SharesDir, user.Username, "data")

			// Initialize quota manager
			qm, err := quota.NewQuotaManager(s.cfg.SharesDir)
			if err == nil {
				// Update backup quota
				if err := qm.UpdateQuota(backupPath, quotaBackupGB); err != nil {
					log.Printf("Warning: Failed to update Btrfs quota for backup: %v", err)
				} else {
					log.Printf("Updated Btrfs quota for %s: %dGB", backupPath, quotaBackupGB)
				}

				// Update data quota
				if err := qm.UpdateQuota(dataPath, quotaDataGB); err != nil {
					log.Printf("Warning: Failed to update Btrfs quota for data: %v", err)
				} else {
					log.Printf("Updated Btrfs quota for %s: %dGB", dataPath, quotaDataGB)
				}
			} else {
				log.Printf("Warning: Failed to initialize quota manager: %v", err)
			}
		}

		log.Printf("Admin %s updated quotas for user %d: backup=%dGB, data=%dGB, total=%dGB",
			session.Username, userID, quotaBackupGB, quotaDataGB, quotaTotalGB)

		// Redirect back to users page with success message
		http.Redirect(w, r, "/admin/users?success="+i18n.T(lang, "users.quota.success"), http.StatusSeeOther)

	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAdminSync displays the automatic sync configuration page
func (s *Server) handleAdminSync(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Get sync configuration
	config, err := syncconfig.Get(s.db)
	if err != nil {
		log.Printf("Error getting sync config: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get recent syncs (last 20)
	type RecentSync struct {
		Username    string
		PeerName    string
		StartedAt   time.Time
		Status      string
		FilesSynced int
		BytesSynced int64
	}

	query := `
		SELECT u.username, p.name, sl.started_at, sl.status, sl.files_synced, sl.bytes_synced
		FROM sync_log sl
		JOIN users u ON sl.user_id = u.id
		JOIN peers p ON sl.peer_id = p.id
		ORDER BY sl.started_at DESC
		LIMIT 20
	`

	rows, err := s.db.Query(query)
	if err != nil {
		log.Printf("Error getting recent syncs: %v", err)
		// Continue with empty list
	}

	var recentSyncs []RecentSync
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var rs RecentSync
			if err := rows.Scan(&rs.Username, &rs.PeerName, &rs.StartedAt, &rs.Status, &rs.FilesSynced, &rs.BytesSynced); err != nil {
				log.Printf("Error scanning sync log: %v", err)
				continue
			}
			recentSyncs = append(recentSyncs, rs)
		}
	}

	// Get success/error messages from query params
	successMsg := r.URL.Query().Get("success")
	errorMsg := r.URL.Query().Get("error")

	data := struct {
		Lang        string
		Title       string
		Session     *auth.Session
		Config      *syncconfig.SyncConfig
		RecentSyncs []RecentSync
		Success     string
		Error       string
	}{
		Lang:        lang,
		Title:       "Synchronisation Automatique",
		Session:     session,
		Config:      config,
		RecentSyncs: recentSyncs,
		Success:     successMsg,
		Error:       errorMsg,
	}

	if err := s.templates.ExecuteTemplate(w, "admin_sync.html", data); err != nil {
		log.Printf("Error rendering admin_sync template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleAdminSyncConfig saves the sync configuration
func (s *Server) handleAdminSyncConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/sync?error=Invalid+form+data", http.StatusSeeOther)
		return
	}

	enabled := r.FormValue("enabled") == "on"
	interval := r.FormValue("interval")
	fixedHourStr := r.FormValue("fixed_hour")

	// Validate interval
	validIntervals := map[string]bool{
		"30min": true,
		"1h":    true,
		"2h":    true,
		"6h":    true,
		"fixed": true,
	}
	if !validIntervals[interval] {
		http.Redirect(w, r, "/admin/sync?error=Invalid+interval", http.StatusSeeOther)
		return
	}

	// Parse fixed_hour if interval is "fixed"
	fixedHour := 23
	if interval == "fixed" {
		var err error
		fixedHour, err = strconv.Atoi(fixedHourStr)
		if err != nil || fixedHour < 0 || fixedHour > 23 {
			http.Redirect(w, r, "/admin/sync?error=Invalid+fixed+hour+(must+be+0-23)", http.StatusSeeOther)
			return
		}
	}

	// Update configuration
	config := &syncconfig.SyncConfig{
		Enabled:   enabled,
		Interval:  interval,
		FixedHour: fixedHour,
	}

	if err := syncconfig.Update(s.db, config); err != nil {
		log.Printf("Error updating sync config: %v", err)
		http.Redirect(w, r, "/admin/sync?error=Failed+to+update+configuration", http.StatusSeeOther)
		return
	}

	log.Printf("Admin %s updated sync config: enabled=%v, interval=%s, fixed_hour=%d",
		session.Username, enabled, interval, fixedHour)

	http.Redirect(w, r, "/admin/sync?success=Configuration+enregistrée+avec+succès", http.StatusSeeOther)
}

// handleAdminSyncForce forces immediate synchronization of all users
func (s *Server) handleAdminSyncForce(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	log.Printf("Admin %s triggered forced synchronization of all users", session.Username)

	// Run SyncAllUsers
	successCount, errorCount, lastError := sync.SyncAllUsers(s.db)

	// Update last_sync timestamp
	if err := syncconfig.UpdateLastSync(s.db); err != nil {
		log.Printf("Warning: Failed to update last_sync: %v", err)
	}

	// Redirect with result message
	if errorCount > 0 {
		errorMsg := fmt.Sprintf("Synchronisation partielle : %d réussis, %d échecs. Dernière erreur: %s",
			successCount, errorCount, lastError)
		http.Redirect(w, r, "/admin/sync?error="+errorMsg, http.StatusSeeOther)
	} else if successCount == 0 {
		http.Redirect(w, r, "/admin/sync?error=Aucune+synchronisation+effectuée+(pas+de+partages+activés+ou+pas+de+pairs)", http.StatusSeeOther)
	} else {
		successMsg := fmt.Sprintf("Synchronisation réussie : %d synchronisations effectuées", successCount)
		http.Redirect(w, r, "/admin/sync?success="+successMsg, http.StatusSeeOther)
	}
}

// handleAdminIncoming displays incoming backups from remote peers
func (s *Server) handleAdminIncoming(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Scan incoming backups directory
	backupsDir := filepath.Join(s.cfg.DataDir, "backups", "incoming")
	backups, err := incoming.ScanIncomingBackups(s.db, backupsDir)
	if err != nil {
		log.Printf("Error scanning incoming backups: %v", err)
		http.Error(w, "Failed to scan incoming backups", http.StatusInternalServerError)
		return
	}

	// Calculate total statistics
	var totalFiles int
	var totalSize int64
	for _, backup := range backups {
		totalFiles += backup.FileCount
		totalSize += backup.TotalSize
	}

	data := struct {
		Lang       string
		Session    *auth.Session
		Backups    []*incoming.IncomingBackup
		TotalFiles int
		TotalSize  string
		Error      string
		Success    string
	}{
		Lang:       s.cfg.Language,
		Session:    session,
		Backups:    backups,
		TotalFiles: totalFiles,
		TotalSize:  incoming.FormatBytes(totalSize),
		Error:      r.URL.Query().Get("error"),
		Success:    r.URL.Query().Get("success"),
	}

	if err := s.templates.ExecuteTemplate(w, "admin_incoming.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

// handleAdminIncomingDelete deletes an incoming backup
func (s *Server) handleAdminIncomingDelete(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/incoming?error=Invalid+form+data", http.StatusSeeOther)
		return
	}

	// Get backup path from form
	backupPath := r.FormValue("path")
	if backupPath == "" {
		http.Redirect(w, r, "/admin/incoming?error=Missing+backup+path", http.StatusSeeOther)
		return
	}

	// Security check: ensure path is within data directory
	dataDir := s.cfg.DataDir
	if !strings.HasPrefix(backupPath, dataDir) {
		log.Printf("Security: Attempted to delete path outside data directory: %s", backupPath)
		http.Redirect(w, r, "/admin/incoming?error=Invalid+backup+path", http.StatusSeeOther)
		return
	}

	// Delete the backup
	if err := incoming.DeleteIncomingBackup(backupPath); err != nil {
		log.Printf("Error deleting backup %s: %v", backupPath, err)
		http.Redirect(w, r, "/admin/incoming?error=Failed+to+delete+backup", http.StatusSeeOther)
		return
	}

	log.Printf("Admin %s deleted incoming backup: %s", session.Username, backupPath)
	http.Redirect(w, r, "/admin/incoming?success=Backup+deleted+successfully", http.StatusSeeOther)
}

// handleRestore displays the restore page
func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := struct {
		Lang    string
		Session *auth.Session
	}{
		Lang:    s.cfg.Language,
		Session: session,
	}

	if err := s.templates.ExecuteTemplate(w, "restore.html", data); err != nil {
		log.Printf("Error rendering restore template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleAPIRestoreBackups returns list of available backups from all configured peers
// GET /api/restore/backups
func (s *Server) handleAPIRestoreBackups(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get all configured peers
	allPeers, err := peers.GetAll(s.db)
	if err != nil {
		log.Printf("Error getting peers: %v", err)
		http.Error(w, "Failed to get peers", http.StatusInternalServerError)
		return
	}

	type PeerBackup struct {
		PeerID       int       `json:"peer_id"`
		PeerName     string    `json:"peer_name"`
		PeerAddress  string    `json:"peer_address"`
		ShareName    string    `json:"share_name"`
		FileCount    int       `json:"file_count"`
		TotalSize    int64     `json:"total_size"`
		LastModified time.Time `json:"last_modified"`
	}

	var allBackups []PeerBackup

	// Query each peer for backups
	for _, peer := range allPeers {
		// Skip disabled peers
		if !peer.SyncEnabled {
			continue
		}

		// Build URL
		url := fmt.Sprintf("https://%s:%d/api/sync/list-user-backups?user_id=%d",
			peer.Address, peer.Port, session.UserID)

		// Create HTTP client with TLS skip verify (self-signed certs)
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{
			Transport: tr,
			Timeout:   10 * time.Second,
		}

		// Create request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("Error creating request for peer %s: %v", peer.Name, err)
			continue
		}

		// Add P2P authentication header if peer has password
		if peer.Password != nil && *peer.Password != "" {
			req.Header.Set("X-Sync-Password", *peer.Password)
		}

		// Execute request
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error contacting peer %s: %v", peer.Name, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Peer %s returned status %d", peer.Name, resp.StatusCode)
			continue
		}

		// Parse response
		type BackupInfo struct {
			ShareName    string    `json:"share_name"`
			FileCount    int       `json:"file_count"`
			TotalSize    int64     `json:"total_size"`
			LastModified time.Time `json:"last_modified"`
		}
		var peerBackups []BackupInfo
		if err := json.NewDecoder(resp.Body).Decode(&peerBackups); err != nil {
			log.Printf("Error decoding response from peer %s: %v", peer.Name, err)
			continue
		}

		// Add peer info to each backup
		for _, backup := range peerBackups {
			allBackups = append(allBackups, PeerBackup{
				PeerID:       peer.ID,
				PeerName:     peer.Name,
				PeerAddress:  peer.Address,
				ShareName:    backup.ShareName,
				FileCount:    backup.FileCount,
				TotalSize:    backup.TotalSize,
				LastModified: backup.LastModified,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(allBackups); err != nil {
		log.Printf("Error encoding backups JSON: %v", err)
	}
}

// handleAPIRestoreFiles returns the file tree for a backup from a remote peer
// GET /api/restore/files?peer_id={id}&backup={share_name}
func (s *Server) handleAPIRestoreFiles(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	peerIDStr := r.URL.Query().Get("peer_id")
	shareName := r.URL.Query().Get("backup")
	if peerIDStr == "" || shareName == "" {
		http.Error(w, "Missing peer_id or backup parameter", http.StatusBadRequest)
		return
	}

	peerID, err := strconv.Atoi(peerIDStr)
	if err != nil {
		http.Error(w, "Invalid peer_id", http.StatusBadRequest)
		return
	}

	// Get peer info
	peer, err := peers.GetByID(s.db, peerID)
	if err != nil {
		log.Printf("Error getting peer %d: %v", peerID, err)
		http.Error(w, "Peer not found", http.StatusNotFound)
		return
	}

	// Get user encryption key
	var encryptedKey []byte
	err = s.db.QueryRow("SELECT encryption_key_encrypted FROM users WHERE id = ?", session.UserID).Scan(&encryptedKey)
	if err != nil {
		log.Printf("Error getting user encryption key: %v", err)
		http.Error(w, "Failed to get encryption key", http.StatusInternalServerError)
		return
	}

	// Get master key from database
	var masterKey string
	err = s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		log.Printf("Error reading master key: %v", err)
		http.Error(w, "Failed to read master key", http.StatusInternalServerError)
		return
	}

	// Decrypt user key
	userKey, err := crypto.DecryptKey(string(encryptedKey), masterKey)
	if err != nil {
		log.Printf("Error decrypting user key: %v", err)
		http.Error(w, "Failed to decrypt user key", http.StatusInternalServerError)
		return
	}

	// Download encrypted manifest from peer
	url := fmt.Sprintf("https://%s:%d/api/sync/download-encrypted-manifest?user_id=%d&share_name=%s",
		peer.Address, peer.Port, session.UserID, shareName)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Add P2P authentication
	if peer.Password != nil && *peer.Password != "" {
		req.Header.Set("X-Sync-Password", *peer.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error downloading manifest from peer %s: %v", peer.Name, err)
		http.Error(w, "Failed to contact peer", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Peer %s returned status %d", peer.Name, resp.StatusCode)
		http.Error(w, "Failed to get manifest from peer", http.StatusInternalServerError)
		return
	}

	// Read encrypted manifest
	encryptedManifest, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading manifest response: %v", err)
		http.Error(w, "Failed to read manifest", http.StatusInternalServerError)
		return
	}

	// Decrypt manifest
	var decryptedBuf bytes.Buffer
	err = crypto.DecryptStream(bytes.NewReader(encryptedManifest), &decryptedBuf, userKey)
	if err != nil {
		log.Printf("Error decrypting manifest: %v", err)
		http.Error(w, "Failed to decrypt manifest", http.StatusInternalServerError)
		return
	}

	// Parse manifest
	var manifest sync.SyncManifest
	if err := json.Unmarshal(decryptedBuf.Bytes(), &manifest); err != nil {
		log.Printf("Error parsing manifest: %v", err)
		http.Error(w, "Failed to parse manifest", http.StatusInternalServerError)
		return
	}

	// Build file tree
	fileTree := restore.BuildFileTree(&manifest)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(fileTree); err != nil {
		log.Printf("Error encoding file tree JSON: %v", err)
	}
}

// handleAPIRestoreDownload downloads and decrypts a file from a remote peer
// GET /api/restore/download?peer_id={id}&backup={share_name}&file={file_path}
func (s *Server) handleAPIRestoreDownload(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	peerIDStr := r.URL.Query().Get("peer_id")
	shareName := r.URL.Query().Get("backup")
	filePath := r.URL.Query().Get("file")

	if peerIDStr == "" || shareName == "" || filePath == "" {
		http.Error(w, "Missing peer_id, backup, or file parameter", http.StatusBadRequest)
		return
	}

	peerID, err := strconv.Atoi(peerIDStr)
	if err != nil {
		http.Error(w, "Invalid peer_id", http.StatusBadRequest)
		return
	}

	// Get peer info
	peer, err := peers.GetByID(s.db, peerID)
	if err != nil {
		log.Printf("Error getting peer %d: %v", peerID, err)
		http.Error(w, "Peer not found", http.StatusNotFound)
		return
	}

	// Get user encryption key
	var encryptedKey []byte
	err = s.db.QueryRow("SELECT encryption_key_encrypted FROM users WHERE id = ?", session.UserID).Scan(&encryptedKey)
	if err != nil {
		log.Printf("Error getting user encryption key: %v", err)
		http.Error(w, "Failed to get encryption key", http.StatusInternalServerError)
		return
	}

	// Get master key from database
	var masterKey string
	err = s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		log.Printf("Error reading master key: %v", err)
		http.Error(w, "Failed to read master key", http.StatusInternalServerError)
		return
	}

	// Decrypt user key
	userKey, err := crypto.DecryptKey(string(encryptedKey), masterKey)
	if err != nil {
		log.Printf("Error decrypting user key: %v", err)
		http.Error(w, "Failed to decrypt user key", http.StatusInternalServerError)
		return
	}

	// Download encrypted file from peer (with proper URL encoding)
	baseURL := fmt.Sprintf("https://%s:%d/api/sync/download-encrypted-file", peer.Address, peer.Port)
	fileURL, err := buildURL(baseURL, map[string]string{
		"user_id":    strconv.Itoa(session.UserID),
		"share_name": shareName,
		"path":       filePath,
	})
	if err != nil {
		log.Printf("Error building URL: %v", err)
		http.Error(w, "Failed to build request URL", http.StatusInternalServerError)
		return
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   120 * time.Second, // Longer timeout for large files
	}

	req, err := http.NewRequest("GET", fileURL, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Add P2P authentication
	if peer.Password != nil && *peer.Password != "" {
		req.Header.Set("X-Sync-Password", *peer.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error downloading file from peer %s: %v", peer.Name, err)
		http.Error(w, "Failed to contact peer", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Peer %s returned status %d", peer.Name, resp.StatusCode)
		http.Error(w, "Failed to get file from peer", http.StatusInternalServerError)
		return
	}

	// Set headers for file download (use original filename without .enc)
	fileName := filepath.Base(filePath)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))

	// Decrypt and stream file directly to response
	err = crypto.DecryptStream(resp.Body, w, userKey)
	if err != nil {
		log.Printf("Error decrypting file %s: %v", filePath, err)
		// Can't send error response here as we've already started writing
		return
	}

	log.Printf("User %s downloaded file %s from peer %s backup %s", session.Username, filePath, peer.Name, shareName)
}

// handleAPIRestoreDownloadMultiple downloads and decrypts multiple files/folders from a remote peer as ZIP
// POST /api/restore/download-multiple?peer_id={id}&backup={share_name}
// Form data: paths[] (multiple)
func (s *Server) handleAPIRestoreDownloadMultiple(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	peerIDStr := r.URL.Query().Get("peer_id")
	shareName := r.URL.Query().Get("backup")
	paths := r.Form["paths"]

	if peerIDStr == "" || shareName == "" || len(paths) == 0 {
		http.Error(w, "Missing peer_id, backup, or paths", http.StatusBadRequest)
		return
	}

	peerID, err := strconv.Atoi(peerIDStr)
	if err != nil {
		http.Error(w, "Invalid peer_id", http.StatusBadRequest)
		return
	}

	// Get peer info
	peer, err := peers.GetByID(s.db, peerID)
	if err != nil {
		log.Printf("Error getting peer %d: %v", peerID, err)
		http.Error(w, "Peer not found", http.StatusNotFound)
		return
	}

	// Get user encryption key
	var encryptedKey []byte
	err = s.db.QueryRow("SELECT encryption_key_encrypted FROM users WHERE id = ?", session.UserID).Scan(&encryptedKey)
	if err != nil {
		log.Printf("Error getting user encryption key: %v", err)
		http.Error(w, "Failed to get encryption key", http.StatusInternalServerError)
		return
	}

	// Get master key from database
	var masterKey string
	err = s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		log.Printf("Error reading master key: %v", err)
		http.Error(w, "Failed to read master key", http.StatusInternalServerError)
		return
	}

	// Decrypt user key
	userKey, err := crypto.DecryptKey(string(encryptedKey), masterKey)
	if err != nil {
		log.Printf("Error decrypting user key: %v", err)
		http.Error(w, "Failed to decrypt user key", http.StatusInternalServerError)
		return
	}

	// Download manifest to determine which paths are files vs directories
	baseManifestURL := fmt.Sprintf("https://%s:%d/api/sync/download-encrypted-manifest", peer.Address, peer.Port)
	manifestURL, err := buildURL(baseManifestURL, map[string]string{
		"user_id":    strconv.Itoa(session.UserID),
		"share_name": shareName,
	})
	if err != nil {
		http.Error(w, "Failed to build manifest URL", http.StatusInternalServerError)
		return
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   300 * time.Second, // 5 min timeout for large operations
	}

	manifestReq, err := http.NewRequest("GET", manifestURL, nil)
	if err != nil {
		http.Error(w, "Failed to create manifest request", http.StatusInternalServerError)
		return
	}

	if peer.Password != nil && *peer.Password != "" {
		manifestReq.Header.Set("X-Sync-Password", *peer.Password)
	}

	manifestResp, err := client.Do(manifestReq)
	if err != nil {
		log.Printf("Error downloading manifest from peer %s: %v", peer.Name, err)
		http.Error(w, "Failed to contact peer", http.StatusInternalServerError)
		return
	}
	defer manifestResp.Body.Close()

	if manifestResp.StatusCode != http.StatusOK {
		log.Printf("Peer %s returned status %d for manifest", peer.Name, manifestResp.StatusCode)
		http.Error(w, "Failed to get manifest from peer", http.StatusInternalServerError)
		return
	}

	// Decrypt manifest
	var manifestBuf bytes.Buffer
	err = crypto.DecryptStream(manifestResp.Body, &manifestBuf, userKey)
	if err != nil {
		log.Printf("Error decrypting manifest: %v", err)
		http.Error(w, "Failed to decrypt manifest", http.StatusInternalServerError)
		return
	}

	// Parse manifest
	var manifest sync.SyncManifest
	err = json.Unmarshal(manifestBuf.Bytes(), &manifest)
	if err != nil {
		log.Printf("Error parsing manifest: %v", err)
		http.Error(w, "Failed to parse manifest", http.StatusInternalServerError)
		return
	}

	// Build file tree from manifest
	fileTree := buildFileTreeFromManifest(&manifest)

	// Expand paths: for each path, determine if it's a file or directory
	// and collect all file paths to download
	filesToDownload := make([]string, 0)
	for _, path := range paths {
		node := getNodeAtPath(fileTree, path)
		if node == nil {
			continue // Skip invalid paths
		}

		if node.IsDir {
			// Collect all files in directory recursively
			collectFilesRecursive(node, &filesToDownload)
		} else {
			// It's a file, add directly
			filesToDownload = append(filesToDownload, path)
		}
	}

	if len(filesToDownload) == 0 {
		http.Error(w, "No files to download", http.StatusBadRequest)
		return
	}

	// Set headers for ZIP download
	zipFileName := fmt.Sprintf("restore_%s_%d.zip", shareName, time.Now().Unix())
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFileName))

	// Create ZIP writer
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	// Download and add each file to ZIP
	for _, filePath := range filesToDownload {
		// Download encrypted file from peer
		baseURL := fmt.Sprintf("https://%s:%d/api/sync/download-encrypted-file", peer.Address, peer.Port)
		fileURL, err := buildURL(baseURL, map[string]string{
			"user_id":    strconv.Itoa(session.UserID),
			"share_name": shareName,
			"path":       filePath,
		})
		if err != nil {
			log.Printf("Error building URL for file %s: %v", filePath, err)
			continue
		}

		fileReq, err := http.NewRequest("GET", fileURL, nil)
		if err != nil {
			log.Printf("Error creating request for file %s: %v", filePath, err)
			continue
		}

		if peer.Password != nil && *peer.Password != "" {
			fileReq.Header.Set("X-Sync-Password", *peer.Password)
		}

		fileResp, err := client.Do(fileReq)
		if err != nil {
			log.Printf("Error downloading file %s from peer %s: %v", filePath, peer.Name, err)
			continue
		}

		if fileResp.StatusCode != http.StatusOK {
			log.Printf("Peer %s returned status %d for file %s", peer.Name, fileResp.StatusCode, filePath)
			fileResp.Body.Close()
			continue
		}

		// Decrypt file to a buffer
		var decryptedBuf bytes.Buffer
		err = crypto.DecryptStream(fileResp.Body, &decryptedBuf, userKey)
		fileResp.Body.Close()

		if err != nil {
			log.Printf("Error decrypting file %s: %v", filePath, err)
			continue
		}

		// Add file to ZIP
		// Remove leading slash for ZIP entries
		zipPath := strings.TrimPrefix(filePath, "/")
		zipEntry, err := zipWriter.Create(zipPath)
		if err != nil {
			log.Printf("Error creating ZIP entry for %s: %v", filePath, err)
			continue
		}

		_, err = zipEntry.Write(decryptedBuf.Bytes())
		if err != nil {
			log.Printf("Error writing ZIP entry for %s: %v", filePath, err)
			continue
		}
	}

	log.Printf("User %s downloaded %d files from peer %s backup %s as ZIP", session.Username, len(filesToDownload), peer.Name, shareName)
}

// Helper functions for file tree navigation

type FileTreeNode struct {
	Name     string
	Path     string
	IsDir    bool
	Size     int64
	ModTime  time.Time
	Children map[string]*FileTreeNode
}

func buildFileTreeFromManifest(manifest *sync.SyncManifest) *FileTreeNode {
	root := &FileTreeNode{
		Name:     "/",
		Path:     "/",
		IsDir:    true,
		Children: make(map[string]*FileTreeNode),
	}

	for filePath, file := range manifest.Files {
		parts := strings.Split(strings.Trim(filePath, "/"), "/")
		currentNode := root

		// Create directory nodes
		for i, part := range parts[:len(parts)-1] {
			if _, exists := currentNode.Children[part]; !exists {
				dirPath := "/" + strings.Join(parts[:i+1], "/")
				currentNode.Children[part] = &FileTreeNode{
					Name:     part,
					Path:     dirPath,
					IsDir:    true,
					Children: make(map[string]*FileTreeNode),
				}
			}
			currentNode = currentNode.Children[part]
		}

		// Add file node
		fileName := parts[len(parts)-1]
		currentNode.Children[fileName] = &FileTreeNode{
			Name:    fileName,
			Path:    filePath,
			IsDir:   false,
			Size:    file.Size,
			ModTime: file.ModTime,
		}
	}

	return root
}

func getNodeAtPath(root *FileTreeNode, path string) *FileTreeNode {
	if path == "/" {
		return root
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	currentNode := root

	for _, part := range parts {
		if currentNode.Children == nil {
			return nil
		}
		node, exists := currentNode.Children[part]
		if !exists {
			return nil
		}
		currentNode = node
	}

	return currentNode
}

func collectFilesRecursive(node *FileTreeNode, files *[]string) {
	if !node.IsDir {
		*files = append(*files, node.Path)
		return
	}

	for _, child := range node.Children {
		collectFilesRecursive(child, files)
	}
}

// buildURL constructs a URL with properly encoded query parameters
func buildURL(baseURL string, params map[string]string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	for key, value := range params {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// handleAPISyncListUserBackups lists available backups for a given user on this peer
// GET /api/sync/list-user-backups?user_id=X
// This endpoint is called by the origin server to discover backups stored on this peer
func (s *Server) handleAPISyncListUserBackups(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "Missing user_id parameter", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Scan incoming backups directory for this user
	backupsDir := filepath.Join(s.cfg.DataDir, "backups", "incoming")
	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			// No backups directory yet
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
			return
		}
		log.Printf("Error reading backups directory: %v", err)
		http.Error(w, "Failed to read backups directory", http.StatusInternalServerError)
		return
	}

	type BackupInfo struct {
		ShareName    string    `json:"share_name"`
		FileCount    int       `json:"file_count"`
		TotalSize    int64     `json:"total_size"`
		LastModified time.Time `json:"last_modified"`
	}

	var backups []BackupInfo
	prefix := fmt.Sprintf("%d_", userID)

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}

		// Extract username from directory name: {user_id}_{username} -> backup_{username}
		username := strings.TrimPrefix(entry.Name(), prefix)
		shareName := "backup_" + username
		backupPath := filepath.Join(backupsDir, entry.Name())

		// Get modification time
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Count files and size
		var fileCount int
		var totalSize int64
		filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			fileCount++
			totalSize += info.Size()
			return nil
		})

		backups = append(backups, BackupInfo{
			ShareName:    shareName,
			FileCount:    fileCount,
			TotalSize:    totalSize,
			LastModified: info.ModTime(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backups)
}

// handleAPISyncDownloadEncryptedManifest downloads the encrypted manifest without decrypting it
// GET /api/sync/download-encrypted-manifest?user_id=X&share_name=Y
// Returns the .anemone-manifest.json.enc file as-is (encrypted)
func (s *Server) handleAPISyncDownloadEncryptedManifest(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	shareName := r.URL.Query().Get("share_name")

	if userIDStr == "" || shareName == "" {
		http.Error(w, "Missing user_id or share_name", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Build backup path
	// Convert share name to directory name (e.g., "backup_test" -> "test")
	// Convention: incoming/{user_id}_{username}/ but API uses backup_{username}
	username := shareName
	if strings.HasPrefix(shareName, "backup_") {
		username = strings.TrimPrefix(shareName, "backup_")
	} else if strings.HasPrefix(shareName, "data_") {
		username = strings.TrimPrefix(shareName, "data_")
	}
	backupDir := fmt.Sprintf("%d_%s", userID, username)
	backupPath := filepath.Join(s.cfg.DataDir, "backups", "incoming", backupDir)
	manifestPath := filepath.Join(backupPath, ".anemone-manifest.json.enc")

	// Check if manifest exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		http.Error(w, "Manifest not found", http.StatusNotFound)
		return
	}

	// Read encrypted manifest
	encryptedData, err := os.ReadFile(manifestPath)
	if err != nil {
		log.Printf("Error reading encrypted manifest: %v", err)
		http.Error(w, "Failed to read manifest", http.StatusInternalServerError)
		return
	}

	// Return encrypted manifest as-is
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\".anemone-manifest.json.enc\"")
	w.WriteHeader(http.StatusOK)
	w.Write(encryptedData)

	log.Printf("Sent encrypted manifest for user %d share %s", userID, shareName)
}

// handleAPISyncDownloadEncryptedFile downloads an encrypted file without decrypting it
// GET /api/sync/download-encrypted-file?user_id=X&share_name=Y&path=Z
// Returns the encrypted file as-is (with .enc extension)
func (s *Server) handleAPISyncDownloadEncryptedFile(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	shareName := r.URL.Query().Get("share_name")
	filePath := r.URL.Query().Get("path")

	if userIDStr == "" || shareName == "" || filePath == "" {
		http.Error(w, "Missing user_id, share_name, or path", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Build backup path
	// Convert share name to directory name (e.g., "backup_test" -> "test")
	// Convention: incoming/{user_id}_{username}/ but API uses backup_{username}
	username := shareName
	if strings.HasPrefix(shareName, "backup_") {
		username = strings.TrimPrefix(shareName, "backup_")
	} else if strings.HasPrefix(shareName, "data_") {
		username = strings.TrimPrefix(shareName, "data_")
	}
	backupDir := fmt.Sprintf("%d_%s", userID, username)
	backupPath := filepath.Join(s.cfg.DataDir, "backups", "incoming", backupDir)

	// Build encrypted file path
	encryptedFilePath := filepath.Join(backupPath, filePath+".enc")

	// Security check: ensure path is within backup directory
	absBackupPath, err := filepath.Abs(backupPath)
	if err != nil {
		http.Error(w, "Invalid backup path", http.StatusBadRequest)
		return
	}
	absFilePath, err := filepath.Abs(encryptedFilePath)
	if err != nil {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(absFilePath, absBackupPath) {
		log.Printf("Security: Attempted path traversal: %s", filePath)
		http.Error(w, "Invalid file path", http.StatusForbidden)
		return
	}

	// Check if file exists
	fileInfo, err := os.Stat(encryptedFilePath)
	if os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error accessing file: %v", err)
		http.Error(w, "Failed to access file", http.StatusInternalServerError)
		return
	}

	// Open encrypted file
	encryptedFile, err := os.Open(encryptedFilePath)
	if err != nil {
		log.Printf("Error opening encrypted file: %v", err)
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		return
	}
	defer encryptedFile.Close()

	// Return encrypted file as-is
	fileName := filepath.Base(filePath) + ".enc"
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	w.WriteHeader(http.StatusOK)

	// Stream the encrypted file
	io.Copy(w, encryptedFile)

	log.Printf("Sent encrypted file %s for user %d share %s", filePath, userID, shareName)
}

// handleAdminBackupExport handles server configuration export
func (s *Server) handleAdminBackupExport(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	if r.Method == "GET" {
		// Display export form
		data := struct {
			Lang    string
			Title   string
			Session *auth.Session
		}{
			Lang:    lang,
			Title:   i18n.T(lang, "backup.export.title"),
			Session: session,
		}
		if err := s.templates.ExecuteTemplate(w, "admin_backup_export.html", data); err != nil {
			log.Printf("Error rendering backup export template: %v", err)
		}
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get passphrase from form
	passphrase := r.FormValue("passphrase")
	if passphrase == "" {
		http.Error(w, "Passphrase is required", http.StatusBadRequest)
		return
	}

	// Confirm passphrase
	passphraseConfirm := r.FormValue("passphrase_confirm")
	if passphrase != passphraseConfirm {
		http.Error(w, "Passphrases do not match", http.StatusBadRequest)
		return
	}

	// Get server name (optional)
	serverName := r.FormValue("server_name")
	if serverName == "" {
		serverName = "Anemone Server"
	}

	// Export configuration
	backup, err := backupPkg.ExportConfiguration(s.db, serverName)
	if err != nil {
		log.Printf("Error exporting configuration: %v", err)
		http.Error(w, "Failed to export configuration", http.StatusInternalServerError)
		return
	}

	// Encrypt backup
	encryptedData, err := backupPkg.EncryptBackup(backup, passphrase)
	if err != nil {
		log.Printf("Error encrypting backup: %v", err)
		http.Error(w, "Failed to encrypt backup", http.StatusInternalServerError)
		return
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("anemone_backup_%s.enc", timestamp)

	// Send encrypted file as download
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(encryptedData)))
	w.WriteHeader(http.StatusOK)
	w.Write(encryptedData)

	log.Printf("Admin exported server configuration (backup size: %d bytes)", len(encryptedData))
}

// handleAdminBackup displays the list of server backups
func (s *Server) handleAdminBackup(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Get backup directory path
	backupDir := filepath.Join(s.cfg.DataDir, "backups", "server")

	// List all backups
	backupFiles, err := serverbackup.ListBackups(backupDir)
	if err != nil {
		log.Printf("Error listing backups: %v", err)
		http.Error(w, "Failed to list backups", http.StatusInternalServerError)
		return
	}

	// Format backups for template
	type BackupInfo struct {
		Filename      string
		FormattedDate string
		FormattedSize string
	}

	backups := make([]BackupInfo, 0, len(backupFiles))
	for _, bf := range backupFiles {
		// Format size in KB or MB
		var sizeStr string
		if bf.Size < 1024*1024 {
			sizeStr = fmt.Sprintf("%.1f KB", float64(bf.Size)/1024)
		} else {
			sizeStr = fmt.Sprintf("%.2f MB", float64(bf.Size)/(1024*1024))
		}

		backups = append(backups, BackupInfo{
			Filename:      bf.Filename,
			FormattedDate: bf.CreatedAt.Format("02/01/2006 15:04:05"),
			FormattedSize: sizeStr,
		})
	}

	data := struct {
		Lang    string
		Session *auth.Session
		Backups []BackupInfo
	}{
		Lang:    lang,
		Session: session,
		Backups: backups,
	}

	if err := s.templates.ExecuteTemplate(w, "admin_backup.html", data); err != nil {
		log.Printf("Error rendering backup template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleAdminBackupCreate creates a manual server backup
func (s *Server) handleAdminBackupCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get backup directory path
	backupDir := filepath.Join(s.cfg.DataDir, "backups", "server")

	// Create backup
	backupPath, err := serverbackup.CreateServerBackup(s.db, backupDir)
	if err != nil {
		log.Printf("Error creating manual backup: %v", err)
		http.Error(w, "Failed to create backup", http.StatusInternalServerError)
		return
	}

	log.Printf("Manual server backup created: %s", backupPath)

	// Redirect back to backup list
	http.Redirect(w, r, "/admin/backup", http.StatusSeeOther)
}

// handleAdminBackupDownload downloads a backup re-encrypted with user passphrase
func (s *Server) handleAdminBackupDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get parameters from form
	filename := r.FormValue("filename")
	passphrase := r.FormValue("passphrase")

	if filename == "" || passphrase == "" {
		http.Error(w, "Missing filename or passphrase", http.StatusBadRequest)
		return
	}

	// Validate passphrase length
	if len(passphrase) < 12 {
		http.Error(w, "Passphrase must be at least 12 characters", http.StatusBadRequest)
		return
	}

	// Get backup directory path
	backupDir := filepath.Join(s.cfg.DataDir, "backups", "server")
	backupPath := filepath.Join(backupDir, filename)

	// Check if file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		http.Error(w, "Backup file not found", http.StatusNotFound)
		return
	}

	// Re-encrypt backup with user passphrase
	reEncryptedData, err := serverbackup.ReEncryptBackup(s.db, backupPath, passphrase)
	if err != nil {
		log.Printf("Error re-encrypting backup: %v", err)
		http.Error(w, "Failed to prepare backup for download", http.StatusInternalServerError)
		return
	}

	// Send as download
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(reEncryptedData)))
	w.WriteHeader(http.StatusOK)
	w.Write(reEncryptedData)

	log.Printf("Admin downloaded backup %s (re-encrypted, size: %d bytes)", filename, len(reEncryptedData))
}

// handleAdminBackupDelete deletes a server backup file
func (s *Server) handleAdminBackupDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get filename from form
	filename := r.FormValue("filename")
	if filename == "" {
		http.Error(w, "Missing filename", http.StatusBadRequest)
		return
	}

	// Security: prevent path traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Get backup directory path
	backupDir := filepath.Join(s.cfg.DataDir, "backups", "server")
	backupPath := filepath.Join(backupDir, filename)

	// Check if file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		http.Error(w, "Backup file not found", http.StatusNotFound)
		return
	}

	// Delete the backup file
	if err := os.Remove(backupPath); err != nil {
		log.Printf("Error deleting backup %s: %v", filename, err)
		http.Error(w, "Failed to delete backup", http.StatusInternalServerError)
		return
	}

	log.Printf("Admin deleted backup %s", filename)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Backup deleted successfully"))
}

// handleRestoreWarning displays the restore warning page
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

			if peer.Password != nil && *peer.Password != "" {
				req.Header.Set("X-Sync-Password", *peer.Password)
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

	data := struct {
		Lang              string
		Title             string
		Session           *auth.Session
		RestoreDate       string
		AvailableBackups  []BackupInfo
	}{
		Lang:             lang,
		Title:            "Server Restored",
		Session:          session,
		RestoreDate:      restoreDate,
		AvailableBackups: availableBackups,
	}

	if err := s.templates.ExecuteTemplate(w, "restore_warning.html", data); err != nil {
		log.Printf("Error rendering restore warning template: %v", err)
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
		log.Printf("Error updating restore_acknowledged: %v", err)
		http.Error(w, "Failed to acknowledge restore", http.StatusInternalServerError)
		return
	}

	log.Printf("User %s acknowledged server restore (manual restore)", session.Username)

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

	// Get peer ID and share name from form
	peerIDStr := r.FormValue("peer_id")
	shareName := r.FormValue("share_name")

	if peerIDStr == "" || shareName == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Missing peer_id or share_name",
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

	log.Printf("User %s starting bulk restore from peer %d share %s", session.Username, peerID, shareName)

	// Start bulk restore in background
	go func() {
		// Note: We can't use progressChan in a simple HTTP request/response
		// For now, we'll just do the restore and mark as complete
		err := bulkrestore.BulkRestoreFromPeer(s.db, session.UserID, peerID, shareName, s.cfg.DataDir, nil)
		if err != nil {
			log.Printf("Bulk restore failed for user %s: %v", session.Username, err)
		} else {
			// Mark restore as completed
			s.db.Exec("UPDATE users SET restore_acknowledged = 1, restore_completed = 1 WHERE id = ?", session.UserID)
			log.Printf("Bulk restore completed successfully for user %s", session.Username)
		}
	}()

	// Return immediate response (restore runs in background)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Bulk restore started in background",
	})
}

// handleAdminRestoreUsers displays all users and their available backups for restoration
// GET /admin/restore-users
func (s *Server) handleAdminRestoreUsers(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Get all users (except admin)
	rows, err := s.db.Query("SELECT id, username FROM users WHERE is_admin = 0 ORDER BY username")
	if err != nil {
		log.Printf("Error getting users: %v", err)
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type UserBackup struct {
		UserID       int
		Username     string
		PeerID       int
		PeerName     string
		ShareName    string
		FileCount    int
		TotalSize    int64
		LastModified time.Time
	}

	var allBackups []UserBackup

	// For each user, check available backups on all peers
	for rows.Next() {
		var userID int
		var username string
		if err := rows.Scan(&userID, &username); err != nil {
			continue
		}

		// Get all peers
		allPeers, err := peers.GetAll(s.db)
		if err != nil {
			log.Printf("Error getting peers: %v", err)
			continue
		}

		// Query each peer for this user's backups
		// Note: We query ALL peers, even disabled ones, because we want to list
		// available backups for restoration (peers are disabled after server restore)
		for _, peer := range allPeers {
			// Build URL
			url := fmt.Sprintf("https://%s:%d/api/sync/list-user-backups?user_id=%d",
				peer.Address, peer.Port, userID)

			// Create HTTP client
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client := &http.Client{
				Transport: tr,
				Timeout:   10 * time.Second,
			}

			// Create request
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				continue
			}

			// Add P2P authentication header
			if peer.Password != nil && *peer.Password != "" {
				req.Header.Set("X-Sync-Password", *peer.Password)
			}

			// Execute request
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				continue
			}

			// Parse response
			type BackupInfo struct {
				ShareName    string    `json:"share_name"`
				FileCount    int       `json:"file_count"`
				TotalSize    int64     `json:"total_size"`
				LastModified time.Time `json:"last_modified"`
			}
			var peerBackups []BackupInfo
			if err := json.NewDecoder(resp.Body).Decode(&peerBackups); err != nil {
				continue
			}

			// Add to results
			for _, backup := range peerBackups {
				allBackups = append(allBackups, UserBackup{
					UserID:       userID,
					Username:     username,
					PeerID:       peer.ID,
					PeerName:     peer.Name,
					ShareName:    backup.ShareName,
					FileCount:    backup.FileCount,
					TotalSize:    backup.TotalSize,
					LastModified: backup.LastModified,
				})
			}
		}
	}

	// Render template
	data := map[string]interface{}{
		"Session": session,
		"Lang":    lang,
		"Backups": allBackups,
	}

	if err := s.templates.ExecuteTemplate(w, "admin_restore_users.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleAdminRestoreUsersRestore handles bulk restoration of a user's files
// POST /admin/restore-users/restore
func (s *Server) handleAdminRestoreUsersRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get parameters
	userIDStr := r.FormValue("user_id")
	peerIDStr := r.FormValue("peer_id")
	shareName := r.FormValue("share_name")

	if userIDStr == "" || peerIDStr == "" || shareName == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Missing parameters",
		})
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid user_id",
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

	// Get username for logging
	var username string
	err = s.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "User not found",
		})
		return
	}

	log.Printf("Admin %s starting bulk restore for user %s (id %d) from peer %d share %s",
		session.Username, username, userID, peerID, shareName)

	// Start bulk restore in background
	go func() {
		err := bulkrestore.BulkRestoreFromPeer(s.db, userID, peerID, shareName, s.cfg.DataDir, nil)
		if err != nil {
			log.Printf("Admin bulk restore failed for user %s: %v", username, err)
		} else {
			log.Printf("Admin bulk restore completed successfully for user %s", username)
		}
	}()

	// Return immediate response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Bulk restore started in background for user " + username,
	})
}
