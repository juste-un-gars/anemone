// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package web provides HTTP handlers for the Anemone web interface.
//
// This file contains restore page and API handlers.
// For restore warning handlers, see handlers_restore_warning.go.
package web

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/crypto"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/peers"
	"github.com/juste-un-gars/anemone/internal/restore"
	"github.com/juste-un-gars/anemone/internal/sync"
)

func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)
	data := V2TemplateData{
		Lang:       lang,
		Title:      i18n.T(lang, "v2.nav.restore"),
		ActivePage: "restore",
		Session:    session,
	}

	tmpl := s.loadV2UserPage("v2_restore.html", s.funcMap)
	if err := tmpl.ExecuteTemplate(w, "v2_base_user", data); err != nil {
		logger.Info("Error rendering v2 restore template", "error", err)
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

	// Get current server name to filter backups
	currentServerName, err := sync.GetServerName(s.db)
	if err != nil {
		logger.Info("Error getting server name", "error", err)
		http.Error(w, "Failed to get server name", http.StatusInternalServerError)
		return
	}

	// Get master key for password decryption
	var masterKey string
	if err := s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey); err != nil {
		logger.Info("Error getting master key", "error", err)
		http.Error(w, "System configuration error", http.StatusInternalServerError)
		return
	}

	// Get all configured peers
	allPeers, err := peers.GetAll(s.db)
	if err != nil {
		logger.Info("Error getting peers", "error", err)
		http.Error(w, "Failed to get peers", http.StatusInternalServerError)
		return
	}

	type PeerBackup struct {
		PeerID       int       `json:"peer_id"`
		PeerName     string    `json:"peer_name"`
		PeerAddress  string    `json:"peer_address"`
		SourceServer string    `json:"source_server"`
		ShareName    string    `json:"share_name"`
		FileCount    int       `json:"file_count"`
		TotalSize    int64     `json:"total_size"`
		LastModified time.Time `json:"last_modified"`
	}

	allBackups := make([]PeerBackup, 0)

	// Query each peer for backups
	// Note: We query ALL peers, even disabled ones, because we want to list
	// available backups for restoration (peers are disabled after server restore)
	for _, peer := range allPeers {
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
			logger.Info("Error creating request for peer", "name", peer.Name, "error", err)
			continue
		}

		// Decrypt and add P2P authentication header if peer has password
		if peer.Password != nil && len(*peer.Password) > 0 {
			peerPassword, err := peers.DecryptPeerPassword(peer.Password, masterKey)
			if err != nil {
				logger.Info("Error decrypting password for peer", "name", peer.Name, "error", err)
				continue
			}
			req.Header.Set("X-Sync-Password", peerPassword)
		}

		// Execute request
		resp, err := client.Do(req)
		if err != nil {
			logger.Info("Error contacting peer", "name", peer.Name, "error", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			logger.Info("Peer returned status", "name", peer.Name, "status_code", resp.StatusCode)
			continue
		}

		// Parse response
		type BackupInfo struct {
			SourceServer string    `json:"source_server"`
			ShareName    string    `json:"share_name"`
			FileCount    int       `json:"file_count"`
			TotalSize    int64     `json:"total_size"`
			LastModified time.Time `json:"last_modified"`
		}
		var peerBackups []BackupInfo
		if err := json.NewDecoder(resp.Body).Decode(&peerBackups); err != nil {
			logger.Info("Error decoding response from peer", "name", peer.Name, "error", err)
			continue
		}

		// Add peer info to each backup (filter by current server name)
		for _, backup := range peerBackups {
			// Only show backups from the current server
			if backup.SourceServer == currentServerName {
				allBackups = append(allBackups, PeerBackup{
					PeerID:       peer.ID,
					PeerName:     peer.Name,
					PeerAddress:  peer.Address,
					SourceServer: backup.SourceServer,
					ShareName:    backup.ShareName,
					FileCount:    backup.FileCount,
					TotalSize:    backup.TotalSize,
					LastModified: backup.LastModified,
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(allBackups); err != nil {
		logger.Info("Error encoding backups JSON", "error", err)
	}
}

// handleAPIRestoreFiles returns the file tree for a backup from a remote peer
// GET /api/restore/files?peer_id={id}&backup={share_name}&source_server={name}
func (s *Server) handleAPIRestoreFiles(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	peerIDStr := r.URL.Query().Get("peer_id")
	shareName := r.URL.Query().Get("backup")
	sourceServer := r.URL.Query().Get("source_server")
	if peerIDStr == "" || shareName == "" || sourceServer == "" {
		http.Error(w, "Missing peer_id, backup, or source_server parameter", http.StatusBadRequest)
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
		logger.Info("Error getting peer", "peer_id", peerID, "error", err)
		http.Error(w, "Peer not found", http.StatusNotFound)
		return
	}

	// Get user encryption key
	var encryptedKey []byte
	err = s.db.QueryRow("SELECT encryption_key_encrypted FROM users WHERE id = ?", session.UserID).Scan(&encryptedKey)
	if err != nil {
		logger.Info("Error getting user encryption key", "error", err)
		http.Error(w, "Failed to get encryption key", http.StatusInternalServerError)
		return
	}

	// Get master key from database
	var masterKey string
	err = s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		logger.Info("Error reading master key", "error", err)
		http.Error(w, "Failed to read master key", http.StatusInternalServerError)
		return
	}

	// Decrypt user key
	userKey, err := crypto.DecryptKey(string(encryptedKey), masterKey)
	if err != nil {
		logger.Info("Error decrypting user key", "error", err)
		http.Error(w, "Failed to decrypt user key", http.StatusInternalServerError)
		return
	}

	// Download encrypted manifest from peer
	url := fmt.Sprintf("https://%s:%d/api/sync/download-encrypted-manifest?user_id=%d&share_name=%s&source_server=%s",
		peer.Address, peer.Port, session.UserID, shareName, sourceServer)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Info("Error creating request", "error", err)
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Decrypt and add P2P authentication
	if peer.Password != nil && len(*peer.Password) > 0 {
		peerPassword, err := peers.DecryptPeerPassword(peer.Password, masterKey)
		if err != nil {
			logger.Info("Error decrypting peer password", "error", err)
			http.Error(w, "Failed to decrypt peer password", http.StatusInternalServerError)
			return
		}
		req.Header.Set("X-Sync-Password", peerPassword)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Info("Error downloading manifest from peer", "name", peer.Name, "error", err)
		http.Error(w, "Failed to contact peer", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Info("Peer returned status", "name", peer.Name, "status_code", resp.StatusCode)
		http.Error(w, "Failed to get manifest from peer", http.StatusInternalServerError)
		return
	}

	// Read encrypted manifest
	encryptedManifest, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Info("Error reading manifest response", "error", err)
		http.Error(w, "Failed to read manifest", http.StatusInternalServerError)
		return
	}

	// Decrypt manifest
	var decryptedBuf bytes.Buffer
	err = crypto.DecryptStream(bytes.NewReader(encryptedManifest), &decryptedBuf, userKey)
	if err != nil {
		logger.Info("Error decrypting manifest", "error", err)
		http.Error(w, "Failed to decrypt manifest", http.StatusInternalServerError)
		return
	}

	// Parse manifest
	var manifest sync.SyncManifest
	if err := json.Unmarshal(decryptedBuf.Bytes(), &manifest); err != nil {
		logger.Info("Error parsing manifest", "error", err)
		http.Error(w, "Failed to parse manifest", http.StatusInternalServerError)
		return
	}

	// Build file tree
	fileTree := restore.BuildFileTree(&manifest)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(fileTree); err != nil {
		logger.Info("Error encoding file tree JSON", "error", err)
	}
}

// handleAPIRestoreDownload downloads and decrypts a file from a remote peer
// GET /api/restore/download?peer_id={id}&backup={share_name}&file={file_path}&source_server={name}
func (s *Server) handleAPIRestoreDownload(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	peerIDStr := r.URL.Query().Get("peer_id")
	shareName := r.URL.Query().Get("backup")
	filePath := r.URL.Query().Get("file")
	sourceServer := r.URL.Query().Get("source_server")

	if peerIDStr == "" || shareName == "" || filePath == "" || sourceServer == "" {
		http.Error(w, "Missing peer_id, backup, file, or source_server parameter", http.StatusBadRequest)
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
		logger.Info("Error getting peer", "peer_id", peerID, "error", err)
		http.Error(w, "Peer not found", http.StatusNotFound)
		return
	}

	// Get user encryption key
	var encryptedKey []byte
	err = s.db.QueryRow("SELECT encryption_key_encrypted FROM users WHERE id = ?", session.UserID).Scan(&encryptedKey)
	if err != nil {
		logger.Info("Error getting user encryption key", "error", err)
		http.Error(w, "Failed to get encryption key", http.StatusInternalServerError)
		return
	}

	// Get master key from database
	var masterKey string
	err = s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		logger.Info("Error reading master key", "error", err)
		http.Error(w, "Failed to read master key", http.StatusInternalServerError)
		return
	}

	// Decrypt user key
	userKey, err := crypto.DecryptKey(string(encryptedKey), masterKey)
	if err != nil {
		logger.Info("Error decrypting user key", "error", err)
		http.Error(w, "Failed to decrypt user key", http.StatusInternalServerError)
		return
	}

	// Download encrypted file from peer (with proper URL encoding)
	baseURL := fmt.Sprintf("https://%s:%d/api/sync/download-encrypted-file", peer.Address, peer.Port)
	fileURL, err := buildURL(baseURL, map[string]string{
		"user_id":       strconv.Itoa(session.UserID),
		"share_name":    shareName,
		"path":          filePath,
		"source_server": sourceServer,
	})
	if err != nil {
		logger.Info("Error building URL", "error", err)
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
		logger.Info("Error creating request", "error", err)
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Decrypt and add P2P authentication
	if peer.Password != nil && len(*peer.Password) > 0 {
		peerPassword, err := peers.DecryptPeerPassword(peer.Password, masterKey)
		if err != nil {
			logger.Info("Error decrypting peer password", "error", err)
			http.Error(w, "Failed to decrypt peer password", http.StatusInternalServerError)
			return
		}
		req.Header.Set("X-Sync-Password", peerPassword)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Info("Error downloading file from peer", "name", peer.Name, "error", err)
		http.Error(w, "Failed to contact peer", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Info("Peer returned status", "name", peer.Name, "status_code", resp.StatusCode)
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
		logger.Info("Error decrypting file", "file_path", filePath, "error", err)
		// Can't send error response here as we've already started writing
		return
	}

	logger.Info("User downloaded file from peer backup", "username", session.Username, "file_path", filePath, "name", peer.Name, "share_name", shareName)
}

// handleAPIRestoreDownloadMultiple downloads and decrypts multiple files/folders from a remote peer as ZIP
// POST /api/restore/download-multiple?peer_id={id}&backup={share_name}&source_server={name}
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
	sourceServer := r.URL.Query().Get("source_server")
	paths := r.Form["paths"]

	if peerIDStr == "" || shareName == "" || sourceServer == "" || len(paths) == 0 {
		http.Error(w, "Missing peer_id, backup, source_server, or paths", http.StatusBadRequest)
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
		logger.Info("Error getting peer", "peer_id", peerID, "error", err)
		http.Error(w, "Peer not found", http.StatusNotFound)
		return
	}

	// Get user encryption key
	var encryptedKey []byte
	err = s.db.QueryRow("SELECT encryption_key_encrypted FROM users WHERE id = ?", session.UserID).Scan(&encryptedKey)
	if err != nil {
		logger.Info("Error getting user encryption key", "error", err)
		http.Error(w, "Failed to get encryption key", http.StatusInternalServerError)
		return
	}

	// Get master key from database
	var masterKey string
	err = s.db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		logger.Info("Error reading master key", "error", err)
		http.Error(w, "Failed to read master key", http.StatusInternalServerError)
		return
	}

	// Decrypt user key
	userKey, err := crypto.DecryptKey(string(encryptedKey), masterKey)
	if err != nil {
		logger.Info("Error decrypting user key", "error", err)
		http.Error(w, "Failed to decrypt user key", http.StatusInternalServerError)
		return
	}

	// Download manifest to determine which paths are files vs directories
	baseManifestURL := fmt.Sprintf("https://%s:%d/api/sync/download-encrypted-manifest", peer.Address, peer.Port)
	manifestURL, err := buildURL(baseManifestURL, map[string]string{
		"user_id":       strconv.Itoa(session.UserID),
		"share_name":    shareName,
		"source_server": sourceServer,
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

	// Decrypt and add P2P authentication
	if peer.Password != nil && len(*peer.Password) > 0 {
		peerPassword, err := peers.DecryptPeerPassword(peer.Password, masterKey)
		if err != nil {
			logger.Info("Error decrypting peer password", "error", err)
			http.Error(w, "Failed to decrypt peer password", http.StatusInternalServerError)
			return
		}
		manifestReq.Header.Set("X-Sync-Password", peerPassword)
	}

	manifestResp, err := client.Do(manifestReq)
	if err != nil {
		logger.Info("Error downloading manifest from peer", "name", peer.Name, "error", err)
		http.Error(w, "Failed to contact peer", http.StatusInternalServerError)
		return
	}
	defer manifestResp.Body.Close()

	if manifestResp.StatusCode != http.StatusOK {
		logger.Info("Peer returned status for manifest", "name", peer.Name, "status_code", manifestResp.StatusCode)
		http.Error(w, "Failed to get manifest from peer", http.StatusInternalServerError)
		return
	}

	// Decrypt manifest
	var manifestBuf bytes.Buffer
	err = crypto.DecryptStream(manifestResp.Body, &manifestBuf, userKey)
	if err != nil {
		logger.Info("Error decrypting manifest", "error", err)
		http.Error(w, "Failed to decrypt manifest", http.StatusInternalServerError)
		return
	}

	// Parse manifest
	var manifest sync.SyncManifest
	err = json.Unmarshal(manifestBuf.Bytes(), &manifest)
	if err != nil {
		logger.Info("Error parsing manifest", "error", err)
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
			"user_id":       strconv.Itoa(session.UserID),
			"share_name":    shareName,
			"path":          filePath,
			"source_server": sourceServer,
		})
		if err != nil {
			logger.Info("Error building URL for file", "file_path", filePath, "error", err)
			continue
		}

		fileReq, err := http.NewRequest("GET", fileURL, nil)
		if err != nil {
			logger.Info("Error creating request for file", "file_path", filePath, "error", err)
			continue
		}

		// Decrypt and add P2P authentication
		if peer.Password != nil && len(*peer.Password) > 0 {
			peerPassword, err := peers.DecryptPeerPassword(peer.Password, masterKey)
			if err != nil {
				logger.Info("Error decrypting peer password", "error", err)
				continue
			}
			fileReq.Header.Set("X-Sync-Password", peerPassword)
		}

		fileResp, err := client.Do(fileReq)
		if err != nil {
			logger.Info("Error downloading file from peer", "file_path", filePath, "name", peer.Name, "error", err)
			continue
		}

		if fileResp.StatusCode != http.StatusOK {
			logger.Info("Peer returned status for file", "name", peer.Name, "status_code", fileResp.StatusCode, "file_path", filePath)
			fileResp.Body.Close()
			continue
		}

		// Decrypt file to a buffer
		var decryptedBuf bytes.Buffer
		err = crypto.DecryptStream(fileResp.Body, &decryptedBuf, userKey)
		fileResp.Body.Close()

		if err != nil {
			logger.Info("Error decrypting file", "file_path", filePath, "error", err)
			continue
		}

		// Add file to ZIP
		// Remove leading slash for ZIP entries
		zipPath := strings.TrimPrefix(filePath, "/")
		zipEntry, err := zipWriter.Create(zipPath)
		if err != nil {
			logger.Info("Error creating ZIP entry for", "file_path", filePath, "error", err)
			continue
		}

		_, err = zipEntry.Write(decryptedBuf.Bytes())
		if err != nil {
			logger.Info("Error writing ZIP entry for", "file_path", filePath, "error", err)
			continue
		}
	}

	logger.Info("User downloaded files from peer backup as ZIP", "username", session.Username, "files_to_download", len(filesToDownload), "name", peer.Name, "share_name", shareName)
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
