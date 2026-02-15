// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"fmt"
	"net/http"
	"time"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/logger"
)

// LockedIPInfo holds display info for a locked IP
type LockedIPInfo struct {
	IP        string
	Remaining string
}

// LockedUserInfo holds display info for a locked user
type LockedUserInfo struct {
	Username  string
	Remaining string
}

func formatDuration(d time.Duration) string {
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if m > 0 {
		return fmt.Sprintf("%d min %d s", m, s)
	}
	return fmt.Sprintf("%d s", s)
}

func (s *Server) handleAdminSecurity(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)
	rl := auth.GetLoginRateLimiter()

	// Get locked IPs
	lockedIPsRaw := rl.GetLockedIPs()
	var lockedIPs []LockedIPInfo
	for ip, remaining := range lockedIPsRaw {
		lockedIPs = append(lockedIPs, LockedIPInfo{
			IP:        ip,
			Remaining: formatDuration(remaining),
		})
	}

	// Get locked users
	lockedUsersRaw := rl.GetLockedUsersWithDuration()
	var lockedUsers []LockedUserInfo
	for username, remaining := range lockedUsersRaw {
		lockedUsers = append(lockedUsers, LockedUserInfo{
			Username:  username,
			Remaining: formatDuration(remaining),
		})
	}

	// Check for success message
	success := r.URL.Query().Get("success")

	data := struct {
		V2TemplateData
		LockedIPs   []LockedIPInfo
		LockedUsers []LockedUserInfo
		Success     string
	}{
		V2TemplateData: V2TemplateData{
			Lang:       lang,
			Title:      i18n.T(lang, "security.title"),
			ActivePage: "security",
			Session:    session,
		},
		LockedIPs:   lockedIPs,
		LockedUsers: lockedUsers,
		Success:     success,
	}

	tmpl := s.loadV2Page("v2_security.html", s.funcMap)
	if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
		logger.Info("Error rendering security template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (s *Server) handleAdminSecurityUnlockIP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := r.FormValue("ip")
	if ip == "" {
		http.Redirect(w, r, "/admin/security", http.StatusSeeOther)
		return
	}

	rl := auth.GetLoginRateLimiter()
	rl.ClearIPLockout(ip)

	logger.Info("Admin unlocked IP", "ip", ip)
	http.Redirect(w, r, "/admin/security?success=ip", http.StatusSeeOther)
}

func (s *Server) handleAdminSecurityUnlockUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	if username == "" {
		http.Redirect(w, r, "/admin/security", http.StatusSeeOther)
		return
	}

	rl := auth.GetLoginRateLimiter()
	rl.ClearUserLockout(username)

	logger.Info("Admin unlocked user", "username", username)
	http.Redirect(w, r, "/admin/security?success=user", http.StatusSeeOther)
}
