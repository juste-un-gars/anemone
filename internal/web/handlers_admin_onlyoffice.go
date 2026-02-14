// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// This file handles OnlyOffice Document Server administration.
// Manages Docker container lifecycle: pull, start, stop, restart, remove.
package web

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/onlyoffice"
)

// handleAdminOnlyOffice displays the OnlyOffice admin page.
func (s *Server) handleAdminOnlyOffice(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

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
func (s *Server) handleAdminOnlyOfficeStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/onlyoffice", http.StatusSeeOther)
		return
	}

	secret := s.cfg.OnlyOfficeSecret
	if secret == "" {
		session, _ := auth.GetSessionFromContext(r)
		s.renderOnlyOfficePage(w, session, s.getLang(r), "", "ANEMONE_OO_SECRET is not configured")
		return
	}

	if err := onlyoffice.StartContainer(secret, s.cfg.OnlyOfficeURL); err != nil {
		logger.Error("Failed to start OnlyOffice container", "error", err)
		session, _ := auth.GetSessionFromContext(r)
		s.renderOnlyOfficePage(w, session, s.getLang(r), "", err.Error())
		return
	}

	session, _ := auth.GetSessionFromContext(r)
	s.renderOnlyOfficePage(w, session, s.getLang(r), "Container started", "")
}

// handleAdminOnlyOfficeStop stops the OnlyOffice container.
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

	session, _ := auth.GetSessionFromContext(r)
	s.renderOnlyOfficePage(w, session, s.getLang(r), "Container removed", "")
}

// onlyOfficeProxy creates a reverse proxy handler for /onlyoffice/* requests.
// Strips the /onlyoffice prefix and forwards to the OnlyOffice container.
// No auth required — security is handled by JWT tokens in the editor config.
func (s *Server) onlyOfficeProxy() http.Handler {
	target, err := url.Parse(s.cfg.OnlyOfficeURL)
	if err != nil {
		logger.Error("Invalid OnlyOffice URL", "url", s.cfg.OnlyOfficeURL, "error", err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "OnlyOffice not configured", http.StatusBadGateway)
		})
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Custom director to strip /onlyoffice prefix and set Host header
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

	return proxy
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
