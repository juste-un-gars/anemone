// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package web provides HTTP handlers and routing for the web administration interface.
package web

import (
	"database/sql"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/config"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/incoming"
	"github.com/juste-un-gars/anemone/internal/quota"
	"github.com/juste-un-gars/anemone/internal/shares"
	"github.com/juste-un-gars/anemone/internal/sync"
	"github.com/juste-un-gars/anemone/internal/syncauth"
	"github.com/juste-un-gars/anemone/internal/trash"
	"github.com/juste-un-gars/anemone/internal/updater"
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
	SyncPassword  string // Generated sync authentication password (setup only)
	Error         string
	Session       *auth.Session
	Stats         *DashboardStats
	Users         []*users.User
	UpdateInfo    *updater.UpdateInfo // Update notification
	Data          map[string]interface{} // Generic data for templates
}

// DashboardStats holds dashboard statistics
type DashboardStats struct {
	UserCount       int
	StorageUsed     string
	PeerStorageUsed string
	StorageQuota    string
	StoragePercent  int
	LastBackup      string
	PeerCount       int
	TrashCount      int
	QuotaInfo       *quota.QuotaInfo
}

// isPathTraversal checks if a path contains actual path traversal attempts.
// This is more accurate than simply checking for ".." anywhere in the string,
// as it allows legitimate filenames like "file...txt" while blocking real
// path traversal attacks like "../../../etc/passwd".
func isPathTraversal(path string) bool {
	// Clean the path to normalize it (removes redundant separators, resolves . and ..)
	cleaned := filepath.Clean(path)

	// Check if the cleaned path starts with .. (traverses up)
	if strings.HasPrefix(cleaned, "..") {
		return true
	}

	// Check if path contains /../ or \..\  (traversal segments)
	if strings.Contains(cleaned, string(filepath.Separator)+".."+string(filepath.Separator)) {
		return true
	}
	if strings.Contains(cleaned, "/..") || strings.Contains(cleaned, `\..`) {
		return true
	}

	return false
}

// NewRouter creates and configures the HTTP router
func NewRouter(db *sql.DB, cfg *config.Config) http.Handler {
	// Load language from database if setup is completed
	var dbLang string
	err := db.QueryRow("SELECT value FROM system_config WHERE key = 'language'").Scan(&dbLang)
	if err == nil && dbLang != "" {
		// Use language from database (set during setup)
		cfg.Language = dbLang
	}

	// Initialize i18n
	if err := i18n.Init(cfg.Language); err != nil {
		log.Printf("Warning: Failed to initialize i18n: %v", err)
	}

	// Create translator instance
	translator, err := i18n.New()
	if err != nil {
		log.Printf("Warning: Failed to create translator: %v", err)
	}

	// Start with translator's FuncMap (includes T with parameter support)
	funcMap := translator.FuncMap()

	// Add additional template functions
	funcMap["ServerName"] = func() string {
		serverName, err := sync.GetServerName(db)
		if err != nil {
			return "Anemone Server"
		}
		return serverName
	}
	funcMap["divf"] = func(a, b int64) float64 {
		return float64(a) / float64(b)
	}
	funcMap["FormatBytes"] = func(bytes int64) string {
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
	funcMap["FormatTime"] = func(t time.Time, lang string) string {
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
	}
	funcMap["derefInt"] = func(ptr *int) int {
		if ptr == nil {
			return 0
		}
		return *ptr
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
	mux.HandleFunc("/admin/settings/trash", auth.RequireAdmin(server.handleAdminSettingsTrash))

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

	// Admin routes - Storage management
	mux.HandleFunc("/admin/storage", auth.RequireAdmin(server.handleAdminStorage))
	mux.HandleFunc("/admin/storage/api", auth.RequireAdmin(server.handleAdminStorageAPI))
	mux.HandleFunc("/api/admin/storage/pool/", auth.RequireAdmin(server.handleAdminStoragePoolScrub))
	mux.HandleFunc("/api/admin/storage/disk/", auth.RequireAdmin(server.handleAdminStorageDiskSMART))

	// Admin routes - Password verification for destructive operations
	mux.HandleFunc("/api/admin/verify-password", auth.RequireAdmin(server.handleAdminVerifyPassword))

	// Admin routes - ZFS Pool management
	mux.HandleFunc("/api/admin/storage/pool", auth.RequireAdmin(server.handleAdminStoragePoolCreate))
	mux.HandleFunc("/api/admin/storage/pool-destroy/", auth.RequireAdmin(server.handleAdminStoragePoolDestroy))
	mux.HandleFunc("/api/admin/storage/pool-export/", auth.RequireAdmin(server.handleAdminStoragePoolExport))
	mux.HandleFunc("/api/admin/storage/pool-vdev/", auth.RequireAdmin(server.handleAdminStoragePoolAddVDev))
	mux.HandleFunc("/api/admin/storage/pool-replace/", auth.RequireAdmin(server.handleAdminStoragePoolReplace))
	mux.HandleFunc("/api/admin/storage/pools/importable", auth.RequireAdmin(server.handleAdminStoragePoolsImportable))
	mux.HandleFunc("/api/admin/storage/pools/import", auth.RequireAdmin(server.handleAdminStoragePoolImport))

	// Admin routes - ZFS Dataset management
	mux.HandleFunc("/api/admin/storage/dataset", auth.RequireAdmin(server.handleAdminStorageDatasetCreate))
	mux.HandleFunc("/api/admin/storage/dataset-delete", auth.RequireAdmin(server.handleAdminStorageDatasetDelete))
	mux.HandleFunc("/api/admin/storage/dataset-update", auth.RequireAdmin(server.handleAdminStorageDatasetUpdate))
	mux.HandleFunc("/api/admin/storage/datasets", auth.RequireAdmin(server.handleAdminStorageDatasetList))

	// Admin routes - ZFS Snapshot management
	mux.HandleFunc("/api/admin/storage/snapshot", auth.RequireAdmin(server.handleAdminStorageSnapshotCreate))
	mux.HandleFunc("/api/admin/storage/snapshot-delete", auth.RequireAdmin(server.handleAdminStorageSnapshotDelete))
	mux.HandleFunc("/api/admin/storage/snapshot-rollback", auth.RequireAdmin(server.handleAdminStorageSnapshotRollback))
	mux.HandleFunc("/api/admin/storage/snapshots", auth.RequireAdmin(server.handleAdminStorageSnapshotList))

	// Admin routes - Disk management
	mux.HandleFunc("/api/admin/storage/disks/available", auth.RequireAdmin(server.handleAdminStorageDisksAvailable))
	mux.HandleFunc("/api/admin/storage/disk/format", auth.RequireAdmin(server.handleAdminStorageDiskFormat))
	mux.HandleFunc("/api/admin/storage/disk/wipe", auth.RequireAdmin(server.handleAdminStorageDiskWipe))

	// Admin routes - Restore all users (after server restoration)
	mux.HandleFunc("/admin/restore-users", auth.RequireAdmin(server.handleAdminRestoreUsers))
	mux.HandleFunc("/admin/restore-users/restore", auth.RequireAdmin(server.handleAdminRestoreUsersRestore))

	// Admin routes - System updates
	mux.HandleFunc("/admin/system/update", auth.RequireAdmin(server.handleAdminSystemUpdate))
	mux.HandleFunc("/admin/system/update/check", auth.RequireAdmin(server.handleAdminSystemUpdateCheck))
	mux.HandleFunc("/admin/system/update/install", auth.RequireAdmin(server.handleAdminSystemUpdateInstall))

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
	mux.HandleFunc("/api/sync/source-info", server.syncAuthMiddleware(server.handleAPISyncSourceInfo)) // PUT
	mux.HandleFunc("/api/sync/file", server.syncAuthMiddleware(server.handleAPISyncFile))               // POST/DELETE
	mux.HandleFunc("/api/sync/list-physical-files", server.syncAuthMiddleware(server.handleAPISyncListPhysicalFiles)) // GET

	// API routes - Remote restore (protected by password authentication)
	mux.HandleFunc("/api/sync/list-user-backups", server.syncAuthMiddleware(server.handleAPISyncListUserBackups))
	mux.HandleFunc("/api/sync/download-encrypted-manifest", server.syncAuthMiddleware(server.handleAPISyncDownloadEncryptedManifest))
	mux.HandleFunc("/api/sync/download-encrypted-file", server.syncAuthMiddleware(server.handleAPISyncDownloadEncryptedFile))

	// API routes - User management (protected by password authentication)
	mux.HandleFunc("/api/sync/delete-user-backup", server.syncAuthMiddleware(server.handleAPISyncDeleteUserBackup))

	// Apply security headers middleware to all routes
	return securityHeadersMiddleware(mux)
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

// securityHeadersMiddleware adds security headers to all HTTP responses
// Protects against XSS, clickjacking, MIME sniffing, and enforces HTTPS
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HSTS - Force HTTPS for 1 year (31536000 seconds)
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Prevent MIME sniffing (force browser to respect Content-Type)
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking (don't allow embedding in iframes)
		w.Header().Set("X-Frame-Options", "DENY")

		// XSS Protection (legacy but still useful)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Content Security Policy - Restrict resource loading
		// default-src 'self' - Only load resources from same origin
		// style-src 'self' 'unsafe-inline' - Allow inline styles (needed for some UI)
		// script-src 'self' - Only execute scripts from same origin
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline' https://cdn.tailwindcss.com; script-src 'self' 'unsafe-inline' https://cdn.tailwindcss.com https://unpkg.com; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'")

		// Referrer Policy - Don't leak referrer to external sites
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy - Disable unnecessary browser features
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}

// getServerName gets the server name from system_config
func (s *Server) getServerName() string {
	serverName, err := sync.GetServerName(s.db)
	if err != nil {
		return "Anemone Server" // Fallback
	}
	return serverName
}

// getLang gets language from user preference (DB), query param, or config
func (s *Server) getLang(r *http.Request) string {
	lang := ""
	session, isLoggedIn := auth.GetSessionFromContext(r)

	// Priority 1: Query parameter (e.g., ?lang=en) - if user is logged in, save it to DB
	if l := r.URL.Query().Get("lang"); l != "" {
		lang = l
		// If user is logged in and changes language via URL, persist it
		if isLoggedIn && (l == "fr" || l == "en") {
			users.UpdateUserLanguage(s.db, session.UserID, l)
		}
	}

	// Priority 2: User language preference from database (if logged in)
	if lang == "" && isLoggedIn {
		user, err := users.GetByID(s.db, session.UserID)
		if err == nil && user.Language != "" {
			lang = user.Language
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


// tWithParams translates a key with placeholder replacement
func tWithParams(lang, key string, args ...interface{}) string {
	translation := i18n.T(lang, key)
	// Support for placeholder replacement (e.g., {{minutes}})
	if len(args) > 0 {
		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				placeholder := fmt.Sprintf("{{%v}}", args[i])
				value := fmt.Sprintf("%v", args[i+1])
				translation = strings.ReplaceAll(translation, placeholder, value)
			}
		}
	}
	return translation
}

// getDashboardStats retrieves dashboard statistics
func (s *Server) getDashboardStats(session *auth.Session, lang string) *DashboardStats {
	stats := &DashboardStats{
		StorageUsed:     "0 B",
		PeerStorageUsed: "0 B",
		StorageQuota:    "∞",
		StoragePercent:  0,
		LastBackup:      i18n.T(lang, "dashboard.user.last_backup.never"),
		UserCount:       0,
		PeerCount:       0,
		TrashCount:      0,
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

		// Calculate storage used by peers (incoming backups)
		incomingDir := filepath.Join(s.cfg.DataDir, "backups", "incoming")
		incomingBackups, err := incoming.ScanIncomingBackups(s.db, incomingDir)
		if err == nil {
			var totalPeerBytes int64
			for _, backup := range incomingBackups {
				totalPeerBytes += backup.TotalSize
			}
			stats.PeerStorageUsed = incoming.FormatBytes(totalPeerBytes)
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
				stats.LastBackup = tWithParams(lang, "dashboard.user.last_backup.minutes_ago", "minutes", int(duration.Minutes()))
			} else if duration < 24*time.Hour {
				stats.LastBackup = tWithParams(lang, "dashboard.user.last_backup.hours_ago", "hours", int(duration.Hours()))
			} else {
				stats.LastBackup = tWithParams(lang, "dashboard.user.last_backup.days_ago", "days", int(duration.Hours()/24))
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
				stats.LastBackup = tWithParams(lang, "dashboard.user.last_backup.minutes_ago", "minutes", int(duration.Minutes()))
			} else if duration < 24*time.Hour {
				stats.LastBackup = tWithParams(lang, "dashboard.user.last_backup.hours_ago", "hours", int(duration.Hours()))
			} else {
				stats.LastBackup = tWithParams(lang, "dashboard.user.last_backup.days_ago", "days", int(duration.Hours()/24))
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
