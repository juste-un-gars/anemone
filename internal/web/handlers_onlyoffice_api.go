// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// This file handles OnlyOffice API endpoints for document editing.
// - /api/oo/download: serves files to the OnlyOffice container (server-to-server)
// - /api/oo/callback: receives edited files from OnlyOffice after save
package web

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/onlyoffice"
	"github.com/juste-un-gars/anemone/internal/shares"
)

// handleOODownload serves a file to the OnlyOffice container.
// Authentication is via JWT token (not session cookie) since this is server-to-server.
// GET /api/oo/download?token=JWT
func (s *Server) handleOODownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	claims, err := onlyoffice.VerifyFileToken(s.cfg.OnlyOfficeSecret, tokenStr)
	if err != nil {
		logger.Warn("OO download: invalid token", "error", err)
		http.Error(w, "Invalid token", http.StatusForbidden)
		return
	}

	// Resolve the file path using share ownership validation
	absPath, err := s.resolveSharePathByUserID(claims.UserID, claims.ShareName, claims.FilePath)
	if err != nil {
		logger.Warn("OO download: path resolution failed", "error", err)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	logger.Info("OO download: serving file", "path", absPath, "user", claims.UserID)
	http.ServeFile(w, r, absPath)
}

// ooCallbackRequest represents the OnlyOffice callback payload.
type ooCallbackRequest struct {
	Key    string `json:"key"`
	Status int    `json:"status"`
	URL    string `json:"url"`
}

// handleOOCallback receives edited files from the OnlyOffice container.
// OnlyOffice calls this when a document is saved (status 2) or force-saved (status 6).
// POST /api/oo/callback
func (s *Server) handleOOCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify JWT from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		// Also check the request body for token (OnlyOffice sends it both ways)
		// For now, if no auth header, check if JWT validation is needed
		logger.Warn("OO callback: missing Authorization header")
	} else {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if _, err := onlyoffice.VerifyCallbackToken(s.cfg.OnlyOfficeSecret, tokenStr); err != nil {
			logger.Warn("OO callback: invalid token", "error", err)
			http.Error(w, "Invalid token", http.StatusForbidden)
			return
		}
	}

	var body ooCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		logger.Warn("OO callback: invalid JSON", "error", err)
		respondJSON(w, map[string]int{"error": 1})
		return
	}

	logger.Info("OO callback received", "key", body.Key, "status", body.Status)

	// Status 2 = document saved, Status 6 = force save
	if body.Status == 2 || body.Status == 6 {
		if err := s.downloadAndSaveOOFile(body.Key, body.URL); err != nil {
			logger.Error("OO callback: failed to save file", "key", body.Key, "error", err)
			respondJSON(w, map[string]int{"error": 1})
			return
		}
	}

	// Always respond with error: 0 (success acknowledgement)
	respondJSON(w, map[string]int{"error": 0})
}

// downloadAndSaveOOFile downloads the edited file from OnlyOffice and saves it.
// The document key format is: userID-shareName-filePath (encoded by the editor handler).
// For non-native formats (md, txt, odt, csv...), OnlyOffice saves in its native
// format (docx/xlsx/pptx). We use the Conversion API to convert back to the original format.
func (s *Server) downloadAndSaveOOFile(docKey, downloadURL string) error {
	// Decode base64url-encoded document key
	decoded, err := base64.URLEncoding.DecodeString(docKey)
	if err != nil {
		return fmt.Errorf("invalid base64 document key: %w", err)
	}
	// Key format: "userID:shareName:relativePath:modtime"
	parts := strings.SplitN(string(decoded), ":", 4)
	if len(parts) < 3 {
		return fmt.Errorf("invalid document key: %s", docKey)
	}

	var userID int
	if _, err := fmt.Sscanf(parts[0], "%d", &userID); err != nil {
		return fmt.Errorf("invalid user ID in key: %w", err)
	}
	shareName := parts[1]
	relPath := parts[2]

	// Resolve target path
	absPath, err := s.resolveSharePathByUserID(userID, shareName, relPath)
	if err != nil {
		return fmt.Errorf("path resolution failed: %w", err)
	}

	// Check if the original format needs conversion.
	// OO always saves in native format (docx/xlsx/pptx); for other formats
	// we must convert back via the Conversion API.
	origExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(relPath), "."))
	actualDownloadURL := downloadURL

	if !isNativeOOFormat(origExt) {
		nativeFmt := nativeOOFormat(origExt)
		logger.Info("OO conversion needed", "from", nativeFmt, "to", origExt, "file", relPath)

		convertedURL, err := s.convertViaOO(downloadURL, nativeFmt, origExt, docKey)
		if err != nil {
			logger.Error("OO conversion failed, saving native format", "error", err)
			// Fall through: save native format as fallback (better than losing data)
		} else {
			actualDownloadURL = convertedURL
		}
	}

	// Download the (possibly converted) file from OnlyOffice.
	// Use InsecureSkipVerify because OO serves HTTPS with Anemone's self-signed cert.
	ooClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := ooClient.Get(actualDownloadURL)
	if err != nil {
		return fmt.Errorf("failed to download from OO: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OO download returned status %d", resp.StatusCode)
	}

	// Write to a temp file first, then rename (atomic save).
	// Try writing next to the target file; if permission denied (share owned
	// by another user), fall back to system temp dir + sudo mv.
	tmpFile := absPath + ".oo-tmp"
	f, err := os.Create(tmpFile)
	useSudo := false
	if err != nil {
		// Permission denied — use system temp dir + sudo mv
		f, err = os.CreateTemp("", "anemone-oo-save-*")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		tmpFile = f.Name()
		useSudo = true
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmpFile)
		return fmt.Errorf("failed to write file: %w", err)
	}
	f.Close()

	if useSudo {
		cmd := exec.Command("sudo", "/usr/bin/mv", tmpFile, absPath)
		if err := cmd.Run(); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("failed to move file (sudo): %w", err)
		}
	} else {
		if err := os.Rename(tmpFile, absPath); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("failed to replace file: %w", err)
		}
	}

	logger.Info("OO file saved", "path", absPath, "user", userID)
	return nil
}

// isNativeOOFormat returns true if the extension is a native OnlyOffice format
// that doesn't need conversion after save (OO saves in these formats natively).
func isNativeOOFormat(ext string) bool {
	switch ext {
	case "docx", "xlsx", "pptx":
		return true
	}
	return false
}

// nativeOOFormat returns the native OO format that a given extension gets
// converted to internally. E.g. "md" → "docx", "csv" → "xlsx", "odp" → "pptx".
func nativeOOFormat(ext string) string {
	switch ooDocumentType(ext) {
	case "cell":
		return "xlsx"
	case "slide":
		return "pptx"
	default:
		return "docx"
	}
}

// convertViaOO calls the OnlyOffice Conversion API to convert a file from
// one format to another. Returns the URL of the converted file.
func (s *Server) convertViaOO(fileURL, fromExt, toExt, key string) (string, error) {
	convPayload := map[string]interface{}{
		"async":      false,
		"filetype":   fromExt,
		"key":        fmt.Sprintf("conv_%d", time.Now().UnixNano()),
		"outputtype": toExt,
		"url":        fileURL,
	}

	// Sign the conversion request with JWT
	token, err := onlyoffice.SignEditorConfig(s.cfg.OnlyOfficeSecret, convPayload)
	if err != nil {
		return "", fmt.Errorf("failed to sign conversion request: %w", err)
	}
	convPayload["token"] = token

	body, _ := json.Marshal(convPayload)

	// OO container serves HTTPS; ensure we use https:// for the conversion API
	ooBase := strings.TrimRight(s.cfg.OnlyOfficeURL, "/")
	ooBase = strings.Replace(ooBase, "http://", "https://", 1)
	convURL := ooBase + "/ConvertService.ashx"
	req, err := http.NewRequest("POST", convURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create conversion request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	ooClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := ooClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("conversion API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("conversion API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		FileURL string `json:"fileUrl"`
		Error   int    `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse conversion response: %w", err)
	}
	if result.Error != 0 {
		return "", fmt.Errorf("conversion error code: %d", result.Error)
	}
	if result.FileURL == "" {
		return "", fmt.Errorf("conversion returned empty fileUrl")
	}

	logger.Info("OO conversion done", "from", fromExt, "to", toExt, "resultURL", result.FileURL)
	return result.FileURL, nil
}

// resolveSharePathByUserID resolves a file path using user ID instead of session.
// Used by OnlyOffice endpoints where authentication is via JWT, not session cookie.
func (s *Server) resolveSharePathByUserID(userID int, shareName, relPath string) (string, error) {
	userShares, err := shares.GetByUser(s.db, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user shares: %w", err)
	}

	var targetShare *shares.Share
	for _, sh := range userShares {
		if sh.Name == shareName {
			targetShare = sh
			break
		}
	}
	if targetShare == nil {
		return "", fmt.Errorf("share not found")
	}

	if relPath == "" || relPath == "/" {
		relPath = "."
	}
	relPath = filepath.Clean(relPath)

	if isPathTraversal(relPath) {
		return "", fmt.Errorf("invalid path")
	}

	absPath := filepath.Join(targetShare.Path, relPath)

	// Verify the path is within the share
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	shareReal, _ := filepath.EvalSymlinks(targetShare.Path)
	if !strings.HasPrefix(realPath, shareReal) {
		return "", fmt.Errorf("path outside share")
	}

	return absPath, nil
}

// respondJSON writes a JSON response.
func respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// resolveSharePathBySession is a helper that wraps resolveSharePath for use with session.
// Used by the editor handler to resolve file paths.
func (s *Server) resolveSharePathBySession(session *auth.Session, shareName, relPath string) (string, error) {
	absPath, _, err := s.resolveSharePath(session, shareName, relPath)
	return absPath, err
}

// handleFilesEdit serves the OnlyOffice editor page (GET /files/edit).
// Opens a document for editing using the OnlyOffice Document Server.
func (s *Server) handleFilesEdit(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	if !s.cfg.OnlyOfficeEnabled {
		http.Error(w, "OnlyOffice is not enabled", http.StatusServiceUnavailable)
		return
	}

	shareName := r.URL.Query().Get("share")
	relPath := r.URL.Query().Get("path")
	if shareName == "" || relPath == "" {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}

	absPath, _, err := s.resolveSharePath(session, shareName, relPath)
	if err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if info.IsDir() {
		http.Error(w, "Cannot edit directory", http.StatusBadRequest)
		return
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(absPath), "."))
	docType := ooDocumentType(ext)
	if docType == "" {
		http.Error(w, "Unsupported file type", http.StatusBadRequest)
		return
	}

	// Document key unique per file version (forces re-download on external changes)
	// Base64url-encoded because OnlyOffice only accepts [0-9-.a-zA-Z_=] in keys
	rawKey := fmt.Sprintf("%d:%s:%s:%d", session.UserID, shareName, relPath, info.ModTime().Unix())
	docKey := base64.URLEncoding.EncodeToString([]byte(rawKey))

	fileToken, err := onlyoffice.SignFileToken(s.cfg.OnlyOfficeSecret, session.UserID, shareName, relPath)
	if err != nil {
		logger.Error("Failed to sign file token", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Build URLs reachable from the OnlyOffice container via Docker bridge network.
	// Uses HTTP (not HTTPS) to avoid self-signed certificate issues.
	// host.docker.internal resolves to the host machine from inside the container.
	baseURL := fmt.Sprintf("http://host.docker.internal:%s", s.cfg.Port)
	downloadURL := fmt.Sprintf("%s/api/oo/download?token=%s", baseURL, fileToken)
	callbackURL := fmt.Sprintf("%s/api/oo/callback", baseURL)
	logger.Info("OO editor: URLs generated", "downloadURL", downloadURL, "callbackURL", callbackURL, "secret_len", len(s.cfg.OnlyOfficeSecret))

	// Back URL points to parent directory in file browser
	parentDir := filepath.Dir(relPath)
	backURL := "/files?share=" + url.QueryEscape(shareName)
	if parentDir != "." && parentDir != "" {
		backURL += "&path=" + url.QueryEscape(parentDir)
	}

	editorConfig := map[string]interface{}{
		"document": map[string]interface{}{
			"fileType": ext,
			"key":      docKey,
			"title":    filepath.Base(absPath),
			"url":      downloadURL,
			"permissions": map[string]interface{}{
				"edit":     true,
				"download": true,
			},
		},
		"documentType": docType,
		"editorConfig": map[string]interface{}{
			"callbackUrl": callbackURL,
			"lang":        lang,
			"mode":        "edit",
			"user": map[string]interface{}{
				"id":   fmt.Sprintf("%d", session.UserID),
				"name": session.Username,
			},
			"customization": map[string]interface{}{
				"goback": map[string]interface{}{
					"url": backURL,
				},
			},
		},
	}

	configToken, err := onlyoffice.SignEditorConfig(s.cfg.OnlyOfficeSecret, editorConfig)
	if err != nil {
		logger.Error("Failed to sign editor config", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	editorConfig["token"] = configToken

	configJSON, _ := json.Marshal(editorConfig)

	// Build OO external URL: browser connects directly to OO via HTTPS on its own port.
	// This avoids the subpath reverse proxy which can't handle WebSocket/Socket.IO.
	ooURL, _ := url.Parse(s.cfg.OnlyOfficeURL)
	ooPort := ooURL.Port()
	if ooPort == "" {
		ooPort = "9980"
	}
	browserHost, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		browserHost = r.Host // no port in Host header
	}
	ooExternalURL := fmt.Sprintf("https://%s:%s", browserHost, ooPort)

	// Override CSP to allow loading OnlyOffice resources from its dedicated port
	ooOrigin := ooExternalURL
	w.Header().Set("Content-Security-Policy", fmt.Sprintf(
		"default-src 'self' %[1]s; script-src 'self' 'unsafe-inline' 'unsafe-eval' %[1]s; style-src 'self' 'unsafe-inline'; img-src 'self' data: %[1]s; font-src 'self' %[1]s; connect-src 'self' %[1]s wss://%[2]s:%[3]s; frame-src 'self' %[1]s; frame-ancestors 'self'",
		ooOrigin, browserHost, ooPort))

	data := struct {
		Lang       string
		Title      string
		FileName   string
		ConfigJSON template.JS
		BackURL    string
		OOURL      string
	}{
		Lang:       lang,
		Title:      filepath.Base(absPath),
		FileName:   filepath.Base(absPath),
		ConfigJSON: template.JS(configJSON),
		BackURL:    backURL,
		OOURL:      ooExternalURL,
	}

	tmpl := template.Must(
		template.New("v2_editor.html").Funcs(s.funcMap).ParseFiles(
			filepath.Join("web", "templates", "v2", "v2_editor.html"),
		),
	)
	if err := tmpl.Execute(w, data); err != nil {
		logger.Error("Error rendering editor template", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

// ooDocumentType returns the OnlyOffice document type for a file extension.
// Returns "word", "cell", "slide", or "" if unsupported.
// Only includes formats that support roundtrip (can be converted back via the
// Conversion API after editing). Legacy formats (doc, xls, ppt) and formats
// without output support (md, xml, etc.) are excluded.
func ooDocumentType(ext string) string {
	switch ext {
	case "docx", "docm", "dotx", "dotm",
		"odt", "ott", "rtf", "txt",
		"epub", "fb2", "htm", "html":
		return "word"
	case "xlsx", "xlsm", "xltx",
		"ods", "ots", "csv":
		return "cell"
	case "pptx", "pptm", "ppsx", "ppsm", "potx", "potm",
		"odp", "otp":
		return "slide"
	default:
		return ""
	}
}
