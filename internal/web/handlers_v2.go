// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// This file provides HTTP handlers for the v2 UI prototype.
package web

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"sort"
	"time"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/incoming"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/rclone"
	"github.com/juste-un-gars/anemone/internal/serverbackup"
	"github.com/juste-un-gars/anemone/internal/syncconfig"
	"github.com/juste-un-gars/anemone/internal/updater"
	"github.com/juste-un-gars/anemone/internal/usbbackup"
)

// V2TemplateData holds common data for all v2 templates.
type V2TemplateData struct {
	Lang       string
	Title      string
	ActivePage string
	Session    *auth.Session
}

// V2DashboardData holds data for the v2 dashboard page.
type V2DashboardData struct {
	V2TemplateData
	Stats          *DashboardStats
	RecentActivity []V2Activity
	UpdateInfo     *updater.UpdateInfo
}

// V2Activity represents a recent activity item on the dashboard.
type V2Activity struct {
	Description string
	Time        string
	Status      string // "success", "error", "warning"
}

// V2BackupsData holds data for the v2 backups consolidated page.
type V2BackupsData struct {
	V2TemplateData

	// USB tab
	USBBackups []V2USBBackup
	USBDrives  []V2Drive

	// Cloud tab
	RcloneConfigs    []V2RcloneConfig
	SSHKeyExists     bool
	SSHKeyPublicKey  string
	SSHKeyRelPath    string

	// P2P Sync tab
	SyncEnabled  bool
	SyncInterval string
	RecentSyncs  []V2SyncEntry

	// Incoming tab
	IncomingBackups []V2IncomingBackup
	IncomingCount   int
	IncomingSize    string

	// Server backup tab
	ServerBackups []V2ServerBackup

	// Recent tab
	RecentBackups []V2RecentBackup

	// UI state
	ActiveTab string // "recent", "usb", "cloud", "p2p", "incoming", "server"
	Flash     string // message text
	FlashType string // "success", "error", "info"
}

// V2USBBackup holds USB backup display data.
type V2USBBackup struct {
	ID         int
	Name       string
	DevicePath string
	IsMounted  bool
	LastSync   string
	LastStatus string
}

// V2Drive holds available drive display data.
type V2Drive struct {
	Name string
	Size string
}

// V2RcloneConfig holds cloud backup display data.
type V2RcloneConfig struct {
	ID           int
	Name         string
	ProviderType string
	Destination  string
	RemotePath   string
	Enabled      bool
	Encrypted    bool
	LastSync     string
	LastStatus   string
}

// V2SyncEntry holds a recent P2P sync log entry.
type V2SyncEntry struct {
	PeerName    string
	Username    string
	Status      string
	FilesSynced int
	BytesSynced string
	CompletedAt string
}

// V2IncomingBackup holds incoming backup display data.
type V2IncomingBackup struct {
	SourceServer       string
	UserID             int
	ShareName          string
	FileCount          int
	TotalSizeFormatted string
	Path               string
}

// V2ServerBackup holds server backup display data.
type V2ServerBackup struct {
	Name          string
	SizeFormatted string
	Date          string
}

// V2RecentBackup holds a consolidated recent backup entry across all types.
type V2RecentBackup struct {
	Type     string // "usb", "cloud", "p2p", "server"
	Name     string
	Status   string // "success", "error", ""
	Details  string
	Time     string
	SortTime time.Time
}

// loadV2Page parses the v2 base layout with a specific page template.
func (s *Server) loadV2Page(page string, funcMap template.FuncMap) *template.Template {
	base := filepath.Join("web", "templates", "v2", "v2_base.html")
	pageFile := filepath.Join("web", "templates", "v2", page)
	return template.Must(
		template.New("v2_base.html").Funcs(funcMap).ParseFiles(base, pageFile),
	)
}

// loadV2UserPage parses the v2 user base layout with a specific page template.
func (s *Server) loadV2UserPage(page string, funcMap template.FuncMap) *template.Template {
	base := filepath.Join("web", "templates", "v2", "v2_base_user.html")
	pageFile := filepath.Join("web", "templates", "v2", page)
	return template.Must(
		template.New("v2_base_user.html").Funcs(funcMap).ParseFiles(base, pageFile),
	)
}

// handleAdminBackups serves the consolidated backups page (USB, Cloud, P2P, Incoming, Server).
func (s *Server) handleAdminBackups(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	lang := s.getLang(r)

	data := V2BackupsData{
		V2TemplateData: V2TemplateData{
			Lang:       lang,
			Title:      i18n.T(lang, "v2.nav.backups"),
			ActivePage: "backups",
			Session:    session,
		},
	}

	// USB backups
	data.USBBackups, data.USBDrives = s.getV2USBData(lang)

	// Cloud/Rclone backups
	data.RcloneConfigs, data.SSHKeyExists, data.SSHKeyPublicKey, data.SSHKeyRelPath = s.getV2RcloneData(lang)

	// P2P sync
	data.SyncEnabled, data.SyncInterval, data.RecentSyncs = s.getV2SyncData(lang)

	// Incoming backups
	data.IncomingBackups, data.IncomingCount, data.IncomingSize = s.getV2IncomingData()

	// Server backups
	data.ServerBackups = s.getV2ServerBackupData()

	// Recent backups (consolidated)
	data.RecentBackups = s.getV2RecentBackups(lang, 10)

	// Active tab from query params
	q := r.URL.Query()
	data.ActiveTab = q.Get("tab")
	if data.ActiveTab == "" {
		data.ActiveTab = "recent"
	}

	// Flash notifications from query params
	switch {
	case q.Get("syncing") != "":
		data.Flash = i18n.T(lang, "rclone.sync_started")
		data.FlashType = "info"
	case q.Get("test_success") != "":
		data.Flash = i18n.T(lang, "rclone.test_success")
		data.FlashType = "success"
	case q.Get("test_error") != "":
		data.Flash = q.Get("test_error")
		data.FlashType = "error"
	case q.Get("deleted") != "":
		data.Flash = i18n.T(lang, "rclone.deleted")
		data.FlashType = "success"
	case q.Get("updated") != "":
		data.Flash = i18n.T(lang, "rclone.updated")
		data.FlashType = "success"
	case q.Get("success") != "":
		data.Flash = i18n.T(lang, "rclone.created")
		data.FlashType = "success"
	case q.Get("error") != "":
		data.Flash = q.Get("error")
		data.FlashType = "error"
	}

	tmpl := s.loadV2Page("v2_backups.html", s.funcMap)
	if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
		logger.Info("Error rendering v2 backups: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// getRecentActivity returns recent sync log entries for the dashboard.
func (s *Server) getRecentActivity(lang string, limit int) []V2Activity {
	rows, err := s.db.Query(`
		SELECT sl.status, sl.started_at, sl.completed_at,
		       COALESCE(u.username, '?') AS username,
		       COALESCE(p.name, '?') AS peer_name
		FROM sync_log sl
		LEFT JOIN users u ON sl.user_id = u.id
		LEFT JOIN peers p ON sl.peer_id = p.id
		ORDER BY sl.started_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		logger.Info("Error querying recent activity: %v", err)
		return nil
	}
	defer rows.Close()

	var activities []V2Activity
	for rows.Next() {
		var status, username, peerName string
		var startedAt time.Time
		var completedAt sql.NullTime
		if err := rows.Scan(&status, &startedAt, &completedAt, &username, &peerName); err != nil {
			continue
		}

		desc := fmt.Sprintf("Sync %s → %s", peerName, username)
		if status == "success" {
			desc += " ✓"
		} else if status == "error" {
			desc += " ✗"
		}

		activities = append(activities, V2Activity{
			Description: desc,
			Time:        formatTimeAgo(startedAt, lang),
			Status:      status,
		})
	}
	return activities
}

// getV2USBData retrieves USB backup data for the v2 backups page.
func (s *Server) getV2USBData(lang string) ([]V2USBBackup, []V2Drive) {
	backups, err := usbbackup.GetAll(s.db)
	if err != nil {
		logger.Info("Error getting USB backups: %v", err)
		return nil, nil
	}

	var v2backups []V2USBBackup
	for _, b := range backups {
		lastSync := i18n.T(lang, "v2.backups.never")
		if b.LastSync != nil {
			lastSync = formatTimeAgo(*b.LastSync, lang)
		}
		v2backups = append(v2backups, V2USBBackup{
			ID:         b.ID,
			Name:       b.Name,
			DevicePath: b.MountPath,
			IsMounted:  b.IsMounted(),
			LastSync:   lastSync,
			LastStatus: b.LastStatus,
		})
	}

	drives, err := usbbackup.DetectDrives()
	if err != nil {
		logger.Info("Error detecting drives: %v", err)
		return v2backups, nil
	}
	var v2drives []V2Drive
	for _, d := range drives {
		v2drives = append(v2drives, V2Drive{
			Name: d.Label,
			Size: formatBytes(d.TotalBytes),
		})
	}
	return v2backups, v2drives
}

// getV2RcloneData retrieves rclone cloud backup data.
func (s *Server) getV2RcloneData(lang string) ([]V2RcloneConfig, bool, string, string) {
	backups, err := rclone.GetAll(s.db)
	if err != nil {
		logger.Info("Error getting rclone backups: %v", err)
		return nil, false, "", ""
	}

	var configs []V2RcloneConfig
	for _, b := range backups {
		lastSync := i18n.T(lang, "v2.backups.never")
		if b.LastSync != nil {
			lastSync = formatTimeAgo(*b.LastSync, lang)
		}
		encrypted := false
		if v, ok := b.ProviderConfig["crypt_password"]; ok && v != "" {
			encrypted = true
		}
		configs = append(configs, V2RcloneConfig{
			ID:           b.ID,
			Name:         b.Name,
			ProviderType: b.ProviderType,
			Destination:  b.DisplayHost(),
			RemotePath:   b.RemotePath,
			Enabled:      b.Enabled,
			Encrypted:    encrypted,
			LastSync:     lastSync,
			LastStatus:   b.LastStatus,
		})
	}

	keyInfo, err := rclone.GetSSHKeyInfo(s.cfg.DataDir)
	sshExists := err == nil && keyInfo != nil && keyInfo.Exists
	var pubKey, relPath string
	if sshExists {
		pubKey = keyInfo.PublicKey
		relPath = keyInfo.RelativePath
	}

	return configs, sshExists, pubKey, relPath
}

// getV2SyncData retrieves P2P sync configuration and recent syncs.
func (s *Server) getV2SyncData(lang string) (bool, string, []V2SyncEntry) {
	cfg, err := syncconfig.Get(s.db)
	if err != nil {
		logger.Info("Error getting sync config: %v", err)
		return false, "", nil
	}

	interval := cfg.Interval
	if cfg.Interval == "fixed" {
		interval = fmt.Sprintf("%02d:00", cfg.FixedHour)
	}

	// Recent syncs
	rows, err := s.db.Query(`
		SELECT COALESCE(u.username, '?'), COALESCE(p.name, '?'),
		       sl.status, sl.files_synced, sl.bytes_synced, sl.completed_at
		FROM sync_log sl
		LEFT JOIN users u ON sl.user_id = u.id
		LEFT JOIN peers p ON sl.peer_id = p.id
		ORDER BY sl.started_at DESC
		LIMIT 20
	`)
	if err != nil {
		logger.Info("Error querying sync log: %v", err)
		return cfg.Enabled, interval, nil
	}
	defer rows.Close()

	var syncs []V2SyncEntry
	for rows.Next() {
		var username, peerName, status string
		var filesSynced int
		var bytesSynced int64
		var completedAt sql.NullTime
		if err := rows.Scan(&username, &peerName, &status, &filesSynced, &bytesSynced, &completedAt); err != nil {
			continue
		}
		completedStr := "-"
		if completedAt.Valid {
			completedStr = formatTimeAgo(completedAt.Time, lang)
		}
		syncs = append(syncs, V2SyncEntry{
			PeerName:    peerName,
			Username:    username,
			Status:      status,
			FilesSynced: filesSynced,
			BytesSynced: formatBytes(bytesSynced),
			CompletedAt: completedStr,
		})
	}
	return cfg.Enabled, interval, syncs
}

// getV2IncomingData retrieves incoming backup data.
func (s *Server) getV2IncomingData() ([]V2IncomingBackup, int, string) {
	backups, err := incoming.ScanIncomingBackups(s.db, s.cfg.IncomingDir)
	if err != nil {
		logger.Info("Error scanning incoming backups: %v", err)
		return nil, 0, "0 B"
	}

	var result []V2IncomingBackup
	var totalBytes int64
	for _, b := range backups {
		totalBytes += b.TotalSize
		result = append(result, V2IncomingBackup{
			SourceServer:       b.SourceServer,
			UserID:             b.UserID,
			ShareName:          b.ShareName,
			FileCount:          b.FileCount,
			TotalSizeFormatted: incoming.FormatBytes(b.TotalSize),
			Path:               b.Path,
		})
	}
	return result, len(backups), incoming.FormatBytes(totalBytes)
}

// getV2ServerBackupData retrieves server backup data.
func (s *Server) getV2ServerBackupData() []V2ServerBackup {
	backupDir := filepath.Join(s.cfg.DataDir, "backups", "server")
	files, err := serverbackup.ListBackups(backupDir)
	if err != nil {
		logger.Info("Error listing server backups: %v", err)
		return nil
	}

	var result []V2ServerBackup
	for _, f := range files {
		result = append(result, V2ServerBackup{
			Name:          f.Filename,
			SizeFormatted: formatBytes(f.Size),
			Date:          f.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	return result
}

// getV2RecentBackups returns the most recent backups across all types.
func (s *Server) getV2RecentBackups(lang string, limit int) []V2RecentBackup {
	var all []V2RecentBackup

	// USB backups
	usbRows, err := s.db.Query(`
		SELECT name, last_sync, last_status
		FROM usb_backups WHERE last_sync IS NOT NULL
		ORDER BY last_sync DESC LIMIT 10
	`)
	if err == nil {
		defer usbRows.Close()
		for usbRows.Next() {
			var name, status string
			var lastSync time.Time
			if err := usbRows.Scan(&name, &lastSync, &status); err != nil {
				continue
			}
			all = append(all, V2RecentBackup{
				Type:     "usb",
				Name:     name,
				Status:   status,
				Time:     formatTimeAgo(lastSync, lang),
				SortTime: lastSync,
			})
		}
	}

	// Cloud (rclone) backups
	cloudRows, err := s.db.Query(`
		SELECT name, provider_type, last_sync, last_status
		FROM rclone_backups WHERE last_sync IS NOT NULL
		ORDER BY last_sync DESC LIMIT 10
	`)
	if err == nil {
		defer cloudRows.Close()
		for cloudRows.Next() {
			var name, provider, status string
			var lastSync time.Time
			if err := cloudRows.Scan(&name, &provider, &lastSync, &status); err != nil {
				continue
			}
			all = append(all, V2RecentBackup{
				Type:     "cloud",
				Name:     name,
				Status:   status,
				Details:  provider,
				Time:     formatTimeAgo(lastSync, lang),
				SortTime: lastSync,
			})
		}
	}

	// P2P sync log
	p2pRows, err := s.db.Query(`
		SELECT sl.completed_at, sl.status,
		       COALESCE(u.username, '?') AS username,
		       COALESCE(p.name, '?') AS peer_name
		FROM sync_log sl
		LEFT JOIN users u ON sl.user_id = u.id
		LEFT JOIN peers p ON sl.peer_id = p.id
		WHERE sl.completed_at IS NOT NULL
		ORDER BY sl.started_at DESC LIMIT 10
	`)
	if err == nil {
		defer p2pRows.Close()
		for p2pRows.Next() {
			var completedAt time.Time
			var status, username, peerName string
			if err := p2pRows.Scan(&completedAt, &status, &username, &peerName); err != nil {
				continue
			}
			all = append(all, V2RecentBackup{
				Type:     "p2p",
				Name:     peerName + " → " + username,
				Status:   status,
				Time:     formatTimeAgo(completedAt, lang),
				SortTime: completedAt,
			})
		}
	}

	// Server backups (filesystem)
	backupDir := filepath.Join(s.cfg.DataDir, "backups", "server")
	files, err := serverbackup.ListBackups(backupDir)
	if err == nil {
		for i, f := range files {
			if i >= 10 {
				break
			}
			all = append(all, V2RecentBackup{
				Type:     "server",
				Name:     f.Filename,
				Status:   "success",
				Details:  formatBytes(f.Size),
				Time:     formatTimeAgo(f.CreatedAt, lang),
				SortTime: f.CreatedAt,
			})
		}
	}

	// Sort by date descending
	sort.Slice(all, func(i, j int) bool {
		return all[i].SortTime.After(all[j].SortTime)
	})

	// Keep only the top N
	if len(all) > limit {
		all = all[:limit]
	}
	return all
}

// formatTimeAgo returns a human-readable relative time string.
func formatTimeAgo(t time.Time, lang string) string {
	diff := time.Since(t)
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
