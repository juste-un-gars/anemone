// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/sysconfig"
)

// handleAdminLogs displays the logs management page.
func (s *Server) handleAdminLogs(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Get current log level from DB
	currentLevel, err := sysconfig.GetLogLevel(s.db)
	if err != nil {
		logger.Error("Error getting log level", "error", err)
		currentLevel = "warn"
	}

	// List log files
	logFiles, err := logger.ListLogFiles(s.cfg.LogDir, "anemone")
	if err != nil {
		logger.Error("Error listing log files", "error", err)
		logFiles = nil
	}

	data := struct {
		Lang         string
		Session      *auth.Session
		CurrentLevel string
		LogFiles     []logger.LogFileEntry
		Success      string
		Error        string
	}{
		Lang:         lang,
		Session:      session,
		CurrentLevel: currentLevel,
		LogFiles:     logFiles,
	}

	if err := s.templates.ExecuteTemplate(w, "admin_logs.html", data); err != nil {
		logger.Error("Error rendering admin logs template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleAdminLogsLevel handles POST to change the log level.
func (s *Server) handleAdminLogsLevel(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/logs", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)
	level := r.FormValue("level")

	// Validate level
	switch level {
	case "debug", "info", "warn", "error":
		// Valid
	default:
		s.renderLogsPage(w, session, lang, "", "Invalid log level")
		return
	}

	// Save to database
	if err := sysconfig.SetLogLevel(s.db, level); err != nil {
		logger.Error("Error setting log level", "error", err)
		s.renderLogsPage(w, session, lang, "", "Failed to save log level")
		return
	}

	// Apply immediately to running logger
	logger.SetLevel(logger.ParseLevel(level))

	logger.Info("Admin changed log level", "admin", session.Username, "level", level)
	s.renderLogsPage(w, session, lang, "Log level updated to "+strings.ToUpper(level), "")
}

// handleAdminLogsDownload serves a log file for download.
func (s *Server) handleAdminLogsDownload(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	filename := r.URL.Query().Get("file")
	if filename == "" {
		http.Error(w, "Missing file parameter", http.StatusBadRequest)
		return
	}

	// Security: only allow downloading from log directory with expected pattern
	// Prevent path traversal attacks
	cleanName := filepath.Base(filename)
	if !strings.HasPrefix(cleanName, "anemone-") || !strings.HasSuffix(cleanName, ".log") {
		http.Error(w, "Invalid file name", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(s.cfg.LogDir, cleanName)

	// Verify file exists and is in the log directory
	absLogDir, _ := filepath.Abs(s.cfg.LogDir)
	absFilePath, _ := filepath.Abs(filePath)
	if !strings.HasPrefix(absFilePath, absLogDir) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check file exists
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if info.IsDir() {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}

	// Serve file
	w.Header().Set("Content-Disposition", "attachment; filename=\""+cleanName+"\"")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	http.ServeFile(w, r, filePath)
}

// renderLogsPage is a helper to render the logs page with messages.
func (s *Server) renderLogsPage(w http.ResponseWriter, session *auth.Session, lang, success, errMsg string) {
	currentLevel, err := sysconfig.GetLogLevel(s.db)
	if err != nil {
		currentLevel = "warn"
	}

	logFiles, _ := logger.ListLogFiles(s.cfg.LogDir, "anemone")

	data := struct {
		Lang         string
		Session      *auth.Session
		CurrentLevel string
		LogFiles     []logger.LogFileEntry
		Success      string
		Error        string
	}{
		Lang:         lang,
		Session:      session,
		CurrentLevel: currentLevel,
		LogFiles:     logFiles,
		Success:      success,
		Error:        errMsg,
	}

	if err := s.templates.ExecuteTemplate(w, "admin_logs.html", data); err != nil {
		logger.Error("Error rendering admin logs template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
