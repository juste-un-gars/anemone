// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// This file provides HTTP handlers for the web file browser.
package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/shares"
)

// decodeJSON reads and decodes a JSON request body.
func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// jsonSuccess writes a success JSON response.
func jsonSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"success":true}`)
}

// jsonError writes an error JSON response.
func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp, _ := json.Marshal(map[string]interface{}{"success": false, "message": message})
	w.Write(resp)
}

// FileEntry represents a file or directory in the browser.
type FileEntry struct {
	Name     string
	Path     string // relative path within share
	IsDir    bool
	Size     int64
	ModTime  time.Time
	MimeType string // category: image, video, audio, document, archive, code, text, other
}

// BreadcrumbItem represents one segment of the navigation breadcrumb.
type BreadcrumbItem struct {
	Name string
	Path string
}

// resolveSharePath validates that the user owns the share and the relative path is safe.
// Returns the absolute filesystem path and the share, or an error.
func (s *Server) resolveSharePath(session *auth.Session, shareName, relPath string) (string, *shares.Share, error) {
	if shareName == "" {
		return "", nil, fmt.Errorf("missing share name")
	}

	// Get user's shares
	userShares, err := shares.GetByUser(s.db, session.UserID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get user shares: %w", err)
	}

	// Find the matching share
	var targetShare *shares.Share
	for _, sh := range userShares {
		if sh.Name == shareName {
			targetShare = sh
			break
		}
	}
	if targetShare == nil {
		return "", nil, fmt.Errorf("share not found")
	}

	// Normalize relative path
	if relPath == "" || relPath == "/" {
		relPath = "."
	}
	relPath = filepath.Clean(relPath)

	// Check for path traversal
	if isPathTraversal(relPath) {
		return "", nil, fmt.Errorf("invalid path")
	}

	// Build absolute path
	absPath := filepath.Join(targetShare.Path, relPath)

	// Resolve symlinks and verify the path is still within the share
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If path doesn't exist yet (for mkdir), check the parent
		if os.IsNotExist(err) {
			parentReal, err2 := filepath.EvalSymlinks(filepath.Dir(absPath))
			if err2 != nil {
				return "", nil, fmt.Errorf("invalid path")
			}
			shareReal, _ := filepath.EvalSymlinks(targetShare.Path)
			if !strings.HasPrefix(parentReal, shareReal) {
				return "", nil, fmt.Errorf("path outside share")
			}
			return absPath, targetShare, nil
		}
		return "", nil, fmt.Errorf("invalid path")
	}

	shareReal, _ := filepath.EvalSymlinks(targetShare.Path)
	if !strings.HasPrefix(realPath, shareReal) {
		return "", nil, fmt.Errorf("path outside share")
	}

	return absPath, targetShare, nil
}

// listDirectory reads directory entries, filters dotfiles, and sorts (dirs first, then alpha).
func listDirectory(dirPath, basePath string) ([]FileEntry, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []FileEntry
	for _, entry := range entries {
		name := entry.Name()
		// Skip dotfiles/dotdirs
		if strings.HasPrefix(name, ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		relPath, _ := filepath.Rel(basePath, filepath.Join(dirPath, name))

		fe := FileEntry{
			Name:    name,
			Path:    relPath,
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}
		if !entry.IsDir() {
			fe.MimeType = detectMimeType(name)
		}
		files = append(files, fe)
	}

	// Sort: directories first, then alphabetical (case-insensitive)
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	return files, nil
}

// buildBreadcrumb creates breadcrumb items from a relative path.
func buildBreadcrumb(relPath string) []BreadcrumbItem {
	if relPath == "" || relPath == "." || relPath == "/" {
		return nil
	}

	parts := strings.Split(filepath.ToSlash(relPath), "/")
	var crumbs []BreadcrumbItem
	for i, part := range parts {
		if part == "" || part == "." {
			continue
		}
		path := strings.Join(parts[:i+1], "/")
		crumbs = append(crumbs, BreadcrumbItem{Name: part, Path: path})
	}
	return crumbs
}

// detectMimeType returns a category string based on file extension.
func detectMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".ico", ".tiff":
		return "image"
	case ".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm":
		return "video"
	case ".mp3", ".flac", ".wav", ".ogg", ".aac", ".wma", ".m4a":
		return "audio"
	case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".odt", ".ods", ".odp":
		return "document"
	case ".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar", ".zst":
		return "archive"
	case ".go", ".py", ".js", ".ts", ".html", ".css", ".json", ".xml", ".yaml", ".yml", ".sh", ".sql", ".c", ".cpp", ".java", ".rs":
		return "code"
	case ".txt", ".md", ".csv", ".log", ".ini", ".conf", ".cfg":
		return "text"
	default:
		return "other"
	}
}

// isValidFileName checks that a filename is safe for filesystem operations.
func isValidFileName(name string) bool {
	if name == "" || len(name) > 255 {
		return false
	}
	if name == "." || name == ".." {
		return false
	}
	if strings.HasPrefix(name, ".") {
		return false
	}
	if strings.ContainsAny(name, "/\\\x00") {
		return false
	}
	return true
}

// handleFiles serves the file browser page (GET /files).
func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	lang := s.getLang(r)

	// Get user's shares
	userShares, err := shares.GetByUser(s.db, session.UserID)
	if err != nil {
		logger.Info("Error getting shares: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	shareName := r.URL.Query().Get("share")
	relPath := r.URL.Query().Get("path")

	// Auto-select if only one share
	if shareName == "" && len(userShares) == 1 {
		shareName = userShares[0].Name
	}

	var files []FileEntry
	var breadcrumb []BreadcrumbItem
	var currentShare string

	if shareName != "" {
		absPath, share, err := s.resolveSharePath(session, shareName, relPath)
		if err != nil {
			logger.Info("Error resolving share path: %v", err)
			http.Error(w, i18n.T(lang, "files.error.access_denied"), http.StatusForbidden)
			return
		}

		// Check if path exists and is a directory
		info, err := os.Stat(absPath)
		if err != nil || !info.IsDir() {
			http.Error(w, i18n.T(lang, "files.error.not_found"), http.StatusNotFound)
			return
		}

		files, err = listDirectory(absPath, share.Path)
		if err != nil {
			logger.Info("Error listing directory: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		breadcrumb = buildBreadcrumb(relPath)
		currentShare = shareName
	}

	data := struct {
		V2TemplateData
		Shares       []*shares.Share
		Files        []FileEntry
		Breadcrumb   []BreadcrumbItem
		CurrentShare string
		CurrentPath  string
		OOEnabled    bool
	}{
		V2TemplateData: V2TemplateData{
			Lang:       lang,
			Title:      i18n.T(lang, "files.title"),
			ActivePage: "files",
			Session:    session,
		},
		Shares:       userShares,
		Files:        files,
		Breadcrumb:   breadcrumb,
		CurrentShare: currentShare,
		CurrentPath:  relPath,
		OOEnabled:    s.cfg.OnlyOfficeEnabled,
	}

	tmpl := s.loadV2UserPage("v2_files.html", s.funcMap)
	if err := tmpl.ExecuteTemplate(w, "v2_base_user", data); err != nil {
		logger.Info("Error rendering v2 files template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleFilesDownload serves file downloads (GET /api/files/download).
func (s *Server) handleFilesDownload(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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
		logger.Info("Download denied for user %d: %v", session.UserID, err)
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if info.IsDir() {
		http.Error(w, "Cannot download directory", http.StatusBadRequest)
		return
	}

	logger.Info("User %s downloading file: %s from share %s", session.Username, relPath, shareName)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(absPath)))
	http.ServeFile(w, r, absPath)
}

// handleFilesUpload handles file uploads (POST /api/files/upload).
func (s *Server) handleFilesUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Limit upload size (2 GB)
	r.Body = http.MaxBytesReader(w, r.Body, 2<<30)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, "File too large or invalid upload", http.StatusBadRequest)
		return
	}

	shareName := r.FormValue("share")
	relPath := r.FormValue("path")

	destDir, _, err := s.resolveSharePath(session, shareName, relPath)
	if err != nil {
		jsonError(w, "Access denied", http.StatusForbidden)
		return
	}

	// Verify destination is a directory
	info, err := os.Stat(destDir)
	if err != nil || !info.IsDir() {
		jsonError(w, "Destination not found", http.StatusNotFound)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		jsonError(w, "No files provided", http.StatusBadRequest)
		return
	}

	count := 0
	for _, fh := range files {
		name := filepath.Base(fh.Filename)
		if !isValidFileName(name) {
			continue
		}

		src, err := fh.Open()
		if err != nil {
			continue
		}

		destFile := filepath.Join(destDir, name)

		// Write to destination — try direct first, fallback to tmp + sudo mv
		dst, err := os.Create(destFile)
		if err != nil {
			// Permission denied: write to temp file, then sudo mv
			tmpFile, err2 := os.CreateTemp("", "anemone-upload-*")
			if err2 != nil {
				src.Close()
				logger.Info("Error creating temp file for %s: %v", destFile, err2)
				continue
			}
			if _, err2 := io.Copy(tmpFile, src); err2 != nil {
				src.Close()
				tmpFile.Close()
				os.Remove(tmpFile.Name())
				logger.Info("Error writing temp file for %s: %v", destFile, err2)
				continue
			}
			src.Close()
			tmpFile.Close()
			cmd := exec.Command("sudo", "/usr/bin/mv", tmpFile.Name(), destFile)
			if err2 := cmd.Run(); err2 != nil {
				os.Remove(tmpFile.Name())
				logger.Info("Error moving upload %s: %v", destFile, err2)
				continue
			}
			count++
			continue
		}
		if _, err := io.Copy(dst, src); err != nil {
			src.Close()
			dst.Close()
			os.Remove(destFile)
			logger.Info("Error writing file %s: %v", destFile, err)
			continue
		}
		src.Close()
		dst.Close()

		count++
	}

	logger.Info("User %s uploaded %d file(s) to share %s path %s", session.Username, count, shareName, relPath)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success":true,"count":%d}`, count)
}

// handleFilesMkdir creates a new directory (POST /api/files/mkdir).
func (s *Server) handleFilesMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Share string `json:"share"`
		Path  string `json:"path"`
		Name  string `json:"name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if !isValidFileName(req.Name) {
		jsonError(w, "Invalid folder name", http.StatusBadRequest)
		return
	}

	// Resolve parent directory
	parentPath, _, err := s.resolveSharePath(session, req.Share, req.Path)
	if err != nil {
		jsonError(w, "Access denied", http.StatusForbidden)
		return
	}

	newDir := filepath.Join(parentPath, req.Name)

	// Check it doesn't already exist
	if _, err := os.Stat(newDir); err == nil {
		jsonError(w, "Already exists", http.StatusConflict)
		return
	}

	// Create directory — try direct first, fallback to sudo
	if err := os.Mkdir(newDir, 0755); err != nil {
		cmd := exec.Command("sudo", "/bin/mkdir", "-m", "0755", newDir)
		if err2 := cmd.Run(); err2 != nil {
			logger.Info("Error creating directory %s: %v (sudo: %v)", newDir, err, err2)
			jsonError(w, "Failed to create folder", http.StatusInternalServerError)
			return
		}
	}

	logger.Info("User %s created folder: %s in share %s", session.Username, req.Name, req.Share)
	jsonSuccess(w)
}

// handleFilesRename renames a file or directory (POST /api/files/rename).
func (s *Server) handleFilesRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Share   string `json:"share"`
		Path    string `json:"path"`
		NewName string `json:"new_name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if !isValidFileName(req.NewName) {
		jsonError(w, "Invalid file name", http.StatusBadRequest)
		return
	}

	// Resolve source path
	srcPath, _, err := s.resolveSharePath(session, req.Share, req.Path)
	if err != nil {
		jsonError(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check source exists
	if _, err := os.Stat(srcPath); err != nil {
		jsonError(w, "File not found", http.StatusNotFound)
		return
	}

	// Build destination (same parent, new name)
	dstPath := filepath.Join(filepath.Dir(srcPath), req.NewName)

	// Check destination doesn't exist
	if _, err := os.Stat(dstPath); err == nil {
		jsonError(w, "A file with this name already exists", http.StatusConflict)
		return
	}

	// Rename — try direct first, fallback to sudo
	if err := os.Rename(srcPath, dstPath); err == nil {
		logger.Info("User %s renamed %s to %s in share %s", session.Username, req.Path, req.NewName, req.Share)
		jsonSuccess(w)
		return
	}
	cmd := exec.Command("sudo", "/usr/bin/mv", srcPath, dstPath)
	if err := cmd.Run(); err != nil {
		logger.Info("Error renaming %s to %s: %v", srcPath, dstPath, err)
		jsonError(w, "Failed to rename", http.StatusInternalServerError)
		return
	}

	logger.Info("User %s renamed %s to %s in share %s", session.Username, req.Path, req.NewName, req.Share)
	jsonSuccess(w)
}

// handleFilesDelete moves a file/directory to the Samba trash (POST /api/files/delete).
func (s *Server) handleFilesDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Share string `json:"share"`
		Path  string `json:"path"`
	}
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Path == "" || req.Path == "." || req.Path == "/" {
		jsonError(w, "Cannot delete root", http.StatusBadRequest)
		return
	}

	absPath, share, err := s.resolveSharePath(session, req.Share, req.Path)
	if err != nil {
		jsonError(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check file exists
	if _, err := os.Stat(absPath); err != nil {
		jsonError(w, "File not found", http.StatusNotFound)
		return
	}

	// Move to .trash/{username}/ (same pattern as Samba recycle)
	trashBase := filepath.Join(share.Path, ".trash", session.Username)

	// Ensure trash directory exists
	os.MkdirAll(trashBase, 0755)

	// Build trash destination preserving relative path structure (keeptree)
	relPath := filepath.Clean(req.Path)
	trashDest := filepath.Join(trashBase, relPath)

	// If destination already exists, add timestamp suffix
	if _, err := os.Stat(trashDest); err == nil {
		ext := filepath.Ext(trashDest)
		base := trashDest[:len(trashDest)-len(ext)]
		timestamp := time.Now().Format("20060102-150405")
		trashDest = fmt.Sprintf("%s.deleted-%s%s", base, timestamp, ext)
	}

	// Ensure parent directory in trash exists
	trashParent := filepath.Dir(trashDest)
	os.MkdirAll(trashParent, 0755)

	// Move to trash — try direct first, fallback to sudo
	err = os.Rename(absPath, trashDest)
	if err != nil {
		cmd := exec.Command("sudo", "/usr/bin/mv", absPath, trashDest)
		err = cmd.Run()
	}
	if err != nil {
		logger.Info("Error moving %s to trash: %v", absPath, err)
		jsonError(w, "Failed to delete", http.StatusInternalServerError)
		return
	}

	logger.Info("User %s deleted %s from share %s (moved to trash)", session.Username, req.Path, req.Share)
	jsonSuccess(w)
}
