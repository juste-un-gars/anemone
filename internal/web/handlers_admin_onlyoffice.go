// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// This file handles OnlyOffice Document Server administration.
// Manages Docker container lifecycle: pull, start, stop, restart, remove.
// Configuration is stored in database (system_config table) for web UI management.
package web

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/onlyoffice"
	"github.com/juste-un-gars/anemone/internal/sysconfig"
)

// loadOnlyOfficeConfig loads OnlyOffice settings from DB into cfg.
// Environment variables take precedence over DB values.
func (s *Server) loadOnlyOfficeConfig() {
	if os.Getenv("ANEMONE_OO_SECRET") == "" {
		if secret, err := sysconfig.GetOnlyOfficeSetting(s.db, "secret"); err == nil && secret != "" {
			s.cfg.OnlyOfficeSecret = secret
		}
	}
	if os.Getenv("ANEMONE_OO_URL") == "" {
		if ooURL, err := sysconfig.GetOnlyOfficeSetting(s.db, "url"); err == nil && ooURL != "" {
			s.cfg.OnlyOfficeURL = ooURL
		}
	}
	if os.Getenv("ANEMONE_OO_ENABLED") == "" {
		if enabled, err := sysconfig.GetOnlyOfficeSetting(s.db, "enabled"); err == nil && enabled != "" {
			s.cfg.OnlyOfficeEnabled = enabled == "true"
		}
	}
}

// handleAdminOnlyOffice displays the OnlyOffice admin page.
func (s *Server) handleAdminOnlyOffice(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	s.loadOnlyOfficeConfig()
	lang := s.getLang(r)
	s.renderOnlyOfficePage(w, session, lang, "", "")
}

// handleAdminOnlyOfficePull pulls the OnlyOffice Docker image.
func (s *Server) handleAdminOnlyOfficePull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/onlyoffice", http.StatusSeeOther)
		return
	}

	if err := onlyoffice.PullImage(); err != nil {
		logger.Error("Failed to pull OnlyOffice image", "error", err)
		session, _ := auth.GetSessionFromContext(r)
		s.renderOnlyOfficePage(w, session, s.getLang(r), "", err.Error())
		return
	}

	session, _ := auth.GetSessionFromContext(r)
	s.renderOnlyOfficePage(w, session, s.getLang(r), "Image pulled successfully", "")
}

// handleAdminOnlyOfficeStart starts the OnlyOffice container.
// Auto-generates a JWT secret if none exists and enables OnlyOffice.
func (s *Server) handleAdminOnlyOfficeStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/onlyoffice", http.StatusSeeOther)
		return
	}

	s.loadOnlyOfficeConfig()

	// Auto-generate secret if not configured
	if s.cfg.OnlyOfficeSecret == "" {
		secret, err := sysconfig.GenerateOnlyOfficeSecret()
		if err != nil {
			logger.Error("Failed to generate OnlyOffice secret", "error", err)
			session, _ := auth.GetSessionFromContext(r)
			s.renderOnlyOfficePage(w, session, s.getLang(r), "", "Failed to generate secret: "+err.Error())
			return
		}
		if err := sysconfig.SetOnlyOfficeSetting(s.db, "secret", secret); err != nil {
			logger.Error("Failed to save OnlyOffice secret", "error", err)
			session, _ := auth.GetSessionFromContext(r)
			s.renderOnlyOfficePage(w, session, s.getLang(r), "", "Failed to save secret: "+err.Error())
			return
		}
		s.cfg.OnlyOfficeSecret = secret
		logger.Info("OnlyOffice JWT secret auto-generated and saved to database")
	}

	if err := onlyoffice.StartContainer(s.cfg.OnlyOfficeSecret, s.cfg.OnlyOfficeURL); err != nil {
		logger.Error("Failed to start OnlyOffice container", "error", err)
		session, _ := auth.GetSessionFromContext(r)
		s.renderOnlyOfficePage(w, session, s.getLang(r), "", err.Error())
		return
	}

	// Auto-enable OnlyOffice
	s.cfg.OnlyOfficeEnabled = true
	if err := sysconfig.SetOnlyOfficeSetting(s.db, "enabled", "true"); err != nil {
		logger.Warn("Failed to save OnlyOffice enabled state", "error", err)
	}
	// Save URL to DB so it persists across restarts
	if err := sysconfig.SetOnlyOfficeSetting(s.db, "url", s.cfg.OnlyOfficeURL); err != nil {
		logger.Warn("Failed to save OnlyOffice URL", "error", err)
	}

	logger.Info("OnlyOffice started and enabled")
	session, _ := auth.GetSessionFromContext(r)
	s.renderOnlyOfficePage(w, session, s.getLang(r), "Container started", "")
}

// handleAdminOnlyOfficeStop stops the OnlyOffice container and disables editing.
func (s *Server) handleAdminOnlyOfficeStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/onlyoffice", http.StatusSeeOther)
		return
	}

	if err := onlyoffice.StopContainer(); err != nil {
		logger.Error("Failed to stop OnlyOffice container", "error", err)
		session, _ := auth.GetSessionFromContext(r)
		s.renderOnlyOfficePage(w, session, s.getLang(r), "", err.Error())
		return
	}

	// Disable OnlyOffice when stopped
	s.cfg.OnlyOfficeEnabled = false
	if err := sysconfig.SetOnlyOfficeSetting(s.db, "enabled", "false"); err != nil {
		logger.Warn("Failed to save OnlyOffice enabled state", "error", err)
	}

	session, _ := auth.GetSessionFromContext(r)
	s.renderOnlyOfficePage(w, session, s.getLang(r), "Container stopped", "")
}

// handleAdminOnlyOfficeRestart restarts the OnlyOffice container.
func (s *Server) handleAdminOnlyOfficeRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/onlyoffice", http.StatusSeeOther)
		return
	}

	if err := onlyoffice.RestartContainer(); err != nil {
		logger.Error("Failed to restart OnlyOffice container", "error", err)
		session, _ := auth.GetSessionFromContext(r)
		s.renderOnlyOfficePage(w, session, s.getLang(r), "", err.Error())
		return
	}

	session, _ := auth.GetSessionFromContext(r)
	s.renderOnlyOfficePage(w, session, s.getLang(r), "Container restarted", "")
}

// handleAdminOnlyOfficeRemove stops and removes the OnlyOffice container.
func (s *Server) handleAdminOnlyOfficeRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/onlyoffice", http.StatusSeeOther)
		return
	}

	if err := onlyoffice.RemoveContainer(); err != nil {
		logger.Error("Failed to remove OnlyOffice container", "error", err)
		session, _ := auth.GetSessionFromContext(r)
		s.renderOnlyOfficePage(w, session, s.getLang(r), "", err.Error())
		return
	}

	// Disable OnlyOffice when removed
	s.cfg.OnlyOfficeEnabled = false
	if err := sysconfig.SetOnlyOfficeSetting(s.db, "enabled", "false"); err != nil {
		logger.Warn("Failed to save OnlyOffice enabled state", "error", err)
	}

	session, _ := auth.GetSessionFromContext(r)
	s.renderOnlyOfficePage(w, session, s.getLang(r), "Container removed", "")
}

// onlyOfficeProxyDynamic returns a handler that checks if OnlyOffice is enabled
// at request time, then proxies to the container. This allows enabling/disabling
// OnlyOffice from the admin UI without restarting Anemone.
func (s *Server) onlyOfficeProxyDynamic() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.cfg.OnlyOfficeEnabled {
			http.Error(w, "OnlyOffice is not enabled", http.StatusServiceUnavailable)
			return
		}

		target, err := url.Parse(s.cfg.OnlyOfficeURL)
		if err != nil {
			http.Error(w, "OnlyOffice not configured", http.StatusBadGateway)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.URL.Path = strings.TrimPrefix(req.URL.Path, "/onlyoffice")
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
			req.Host = target.Host
		}
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Warn("OnlyOffice proxy error", "path", r.URL.Path, "error", err)
			http.Error(w, "OnlyOffice unavailable", http.StatusBadGateway)
		}

		proxy.ServeHTTP(w, r)
	})
}

// renderOnlyOfficePage renders the OnlyOffice admin page.
func (s *Server) renderOnlyOfficePage(w http.ResponseWriter, session *auth.Session, lang, success, errorMsg string) {
	dockerInstalled := onlyoffice.IsDockerInstalled()
	imagePresent := onlyoffice.IsImagePresent()
	containerStatus := onlyoffice.ContainerStatus()

	data := struct {
		V2TemplateData
		DockerInstalled bool
		ImagePresent    bool
		ContainerStatus string
		OOEnabled       bool
		OOURL           string
		OOSecretSet     bool
		Success         string
		Error           string
	}{
		V2TemplateData: V2TemplateData{
			Lang:       lang,
			Title:      "OnlyOffice",
			ActivePage: "onlyoffice",
			Session:    session,
		},
		DockerInstalled: dockerInstalled,
		ImagePresent:    imagePresent,
		ContainerStatus: containerStatus,
		OOEnabled:       s.cfg.OnlyOfficeEnabled,
		OOURL:           s.cfg.OnlyOfficeURL,
		OOSecretSet:     s.cfg.OnlyOfficeSecret != "",
		Success:         success,
		Error:           errorMsg,
	}

	tmpl := s.loadV2Page("v2_onlyoffice.html", s.funcMap)
	if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
		logger.Error("Error rendering OnlyOffice template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
