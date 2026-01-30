// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"encoding/json"
	"github.com/juste-un-gars/anemone/internal/logger"
	"net/http"
	"strings"

	"github.com/juste-un-gars/anemone/internal/adminverify"
	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/storage"
)

func (s *Server) handleAdminStorage(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	lang := s.getLang(r)

	// Get storage overview
	overview, err := storage.GetStorageOverview()
	if err != nil {
		logger.Info("Error getting storage overview: %v", err)
		// Continue with empty overview
		overview = &storage.StorageOverview{}
	}

	data := map[string]interface{}{
		"Title":          i18n.T(lang, "storage_management"),
		"Session":        session,
		"Overview":       overview,
		"Disks":          overview.Disks,
		"Pools":          overview.Pools,
		"SMARTAvailable": overview.SMARTAvailable,
		"ZFSAvailable":   overview.ZFSAvailable,
		"Lang":           lang,
		"ActivePage":     "storage",
	}

	if err := s.templates.ExecuteTemplate(w, "admin_storage.html", data); err != nil {
		logger.Info("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleAdminStorageAPI provides JSON API for storage data
func (s *Server) handleAdminStorageAPI(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	overview, err := storage.GetStorageOverview()
	if err != nil {
		logger.Info("Error getting storage overview: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(overview)
}

// handleAdminStoragePoolScrub handles starting/stopping ZFS pool scrub
func (s *Server) handleAdminStoragePoolScrub(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract pool name from URL path: /api/admin/storage/pool/{name}/scrub
	path := r.URL.Path
	parts := splitPath(path)
	if len(parts) < 6 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	poolName := parts[4] // /api/admin/storage/pool/{name}/scrub

	// Check action parameter (start or stop)
	action := r.URL.Query().Get("action")
	if action == "" {
		action = "start"
	}

	var err error
	if action == "stop" {
		err = storage.StopScrub(poolName)
	} else {
		err = storage.StartScrub(poolName)
	}

	if err != nil {
		logger.Info("Error %sing scrub on pool %s: %v", action, poolName, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"action":  action,
		"pool":    poolName,
	})
}

// handleAdminStorageDiskSMART returns detailed SMART info for a disk
func (s *Server) handleAdminStorageDiskSMART(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract disk name from URL path: /api/admin/storage/disk/{name}/smart
	path := r.URL.Path
	parts := splitPath(path)
	if len(parts) < 6 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	diskName := parts[4] // /api/admin/storage/disk/{name}/smart

	// Build device path
	devicePath := "/dev/" + diskName

	smartInfo, err := storage.GetSMARTInfo(devicePath)
	if err != nil {
		logger.Info("Error getting SMART info for %s: %v", devicePath, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(smartInfo)
}

// splitPath splits URL path into parts
func splitPath(path string) []string {
	var parts []string
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// getClientIP extracts client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for reverse proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	parts := strings.Split(r.RemoteAddr, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return r.RemoteAddr
}

// handleAdminVerifyPassword verifies admin password and returns a token
func (s *Server) handleAdminVerifyPassword(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	ip := getClientIP(r)
	verifier := adminverify.GetVerifier()

	token, err := verifier.VerifyPassword(s.db, session.UserID, req.Password, ip)
	if err != nil {
		logger.Info("Password verification failed for user %d from %s: %v", session.UserID, ip, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":              err.Error(),
			"remaining_attempts": verifier.GetRemainingAttempts(ip),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   token,
	})
}

// validateVerificationToken validates a verification token from request
func (s *Server) validateVerificationToken(r *http.Request, session *auth.Session) error {
	token := r.Header.Get("X-Verification-Token")
	if token == "" {
		// Also check in JSON body
		return adminverify.GetVerifier().ValidateToken("", session.UserID)
	}
	return adminverify.GetVerifier().ValidateToken(token, session.UserID)
}

// --- ZFS Pool Handlers ---

// handleAdminStoragePoolCreate creates a new ZFS pool
func (s *Server) handleAdminStoragePoolCreate(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req storage.PoolCreateOptions
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	// Verify password token for this destructive operation
	if err := s.validateVerificationToken(r, session); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Password verification required"})
		return
	}

	if err := storage.CreatePool(req); err != nil {
		logger.Info("Error creating pool: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"pool":    req.Name,
	})
}

// handleAdminStoragePoolDestroy destroys a ZFS pool
func (s *Server) handleAdminStoragePoolDestroy(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract pool name from URL path
	parts := splitPath(r.URL.Path)
	if len(parts) < 5 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	poolName := parts[4]

	// Verify password token
	if err := s.validateVerificationToken(r, session); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Password verification required"})
		return
	}

	force := r.URL.Query().Get("force") == "true"

	if err := storage.DestroyPool(poolName, force); err != nil {
		logger.Info("Error destroying pool %s: %v", poolName, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"pool":    poolName,
	})
}

// handleAdminStoragePoolExport exports a ZFS pool
func (s *Server) handleAdminStoragePoolExport(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := splitPath(r.URL.Path)
	if len(parts) < 6 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	poolName := parts[4]

	// Verify password token
	if err := s.validateVerificationToken(r, session); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Password verification required"})
		return
	}

	force := r.URL.Query().Get("force") == "true"

	if err := storage.ExportPool(poolName, force); err != nil {
		logger.Info("Error exporting pool %s: %v", poolName, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"pool":    poolName,
	})
}

// handleAdminStoragePoolsImportable lists importable pools
func (s *Server) handleAdminStoragePoolsImportable(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	pools, err := storage.ListImportablePools()
	if err != nil {
		logger.Info("Error listing importable pools: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pools)
}

// handleAdminStoragePoolImport imports a ZFS pool
func (s *Server) handleAdminStoragePoolImport(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name    string `json:"name"`
		Force   bool   `json:"force"`
		AltRoot string `json:"alt_root"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	// Verify password token
	if err := s.validateVerificationToken(r, session); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Password verification required"})
		return
	}

	if err := storage.ImportPool(req.Name, req.Force, req.AltRoot); err != nil {
		logger.Info("Error importing pool %s: %v", req.Name, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"pool":    req.Name,
	})
}

// handleAdminStoragePoolAddVDev adds a vdev to a pool
func (s *Server) handleAdminStoragePoolAddVDev(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := splitPath(r.URL.Path)
	if len(parts) < 6 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	poolName := parts[4]

	var req struct {
		VDevType string   `json:"vdev_type"`
		Disks    []string `json:"disks"`
		Force    bool     `json:"force"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	// Verify password token
	if err := s.validateVerificationToken(r, session); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Password verification required"})
		return
	}

	opts := storage.AddVDevOptions{
		PoolName: poolName,
		VDevType: req.VDevType,
		Disks:    req.Disks,
		Force:    req.Force,
	}

	if err := storage.AddVDev(opts); err != nil {
		logger.Info("Error adding vdev to pool %s: %v", poolName, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"pool":    poolName,
	})
}

// handleAdminStoragePoolReplace replaces a disk in a pool
func (s *Server) handleAdminStoragePoolReplace(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := splitPath(r.URL.Path)
	if len(parts) < 6 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	poolName := parts[4]

	var req struct {
		OldDisk string `json:"old_disk"`
		NewDisk string `json:"new_disk"`
		Force   bool   `json:"force"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	// Verify password token
	if err := s.validateVerificationToken(r, session); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Password verification required"})
		return
	}

	opts := storage.ReplaceOptions{
		PoolName: poolName,
		OldDisk:  req.OldDisk,
		NewDisk:  req.NewDisk,
		Force:    req.Force,
	}

	if err := storage.ReplaceDisk(opts); err != nil {
		logger.Info("Error replacing disk in pool %s: %v", poolName, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"pool":    poolName,
	})
}

// --- ZFS Dataset Handlers ---

// handleAdminStorageDatasetCreate creates a new dataset
func (s *Server) handleAdminStorageDatasetCreate(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req storage.DatasetCreateOptions
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	// Verify password token
	if err := s.validateVerificationToken(r, session); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Password verification required"})
		return
	}

	if err := storage.CreateDataset(req); err != nil {
		logger.Info("Error creating dataset: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"dataset": req.Name,
	})
}

// handleAdminStorageDatasetDelete deletes a dataset
func (s *Server) handleAdminStorageDatasetDelete(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Dataset name is URL encoded in the path
	name := r.URL.Query().Get("name")
	if name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Dataset name required"})
		return
	}

	// Verify password token
	if err := s.validateVerificationToken(r, session); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Password verification required"})
		return
	}

	recursive := r.URL.Query().Get("recursive") == "true"
	force := r.URL.Query().Get("force") == "true"

	if err := storage.DeleteDataset(name, recursive, force); err != nil {
		logger.Info("Error deleting dataset %s: %v", name, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"dataset": name,
	})
}

// handleAdminStorageDatasetUpdate updates dataset properties
func (s *Server) handleAdminStorageDatasetUpdate(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name       string `json:"name"`
		Property   string `json:"property"`
		Value      string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	// Verify password token
	if err := s.validateVerificationToken(r, session); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Password verification required"})
		return
	}

	if err := storage.SetDatasetProperty(req.Name, req.Property, req.Value); err != nil {
		logger.Info("Error setting property on dataset %s: %v", req.Name, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"dataset":  req.Name,
		"property": req.Property,
		"value":    req.Value,
	})
}

// handleAdminStorageDatasetList lists datasets in a pool
func (s *Server) handleAdminStorageDatasetList(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	parent := r.URL.Query().Get("parent")
	if parent == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Parent pool/dataset required"})
		return
	}

	datasets, err := storage.ListDatasets(parent)
	if err != nil {
		logger.Info("Error listing datasets: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(datasets)
}

// --- ZFS Snapshot Handlers ---

// handleAdminStorageSnapshotCreate creates a snapshot
func (s *Server) handleAdminStorageSnapshotCreate(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req storage.SnapshotCreateOptions
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	// No password verification needed for creating snapshots (non-destructive)
	_ = session

	if err := storage.CreateSnapshot(req); err != nil {
		logger.Info("Error creating snapshot: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"snapshot": req.Dataset + "@" + req.Name,
	})
}

// handleAdminStorageSnapshotDelete deletes a snapshot
func (s *Server) handleAdminStorageSnapshotDelete(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Snapshot name required"})
		return
	}

	// Verify password token
	if err := s.validateVerificationToken(r, session); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Password verification required"})
		return
	}

	recursive := r.URL.Query().Get("recursive") == "true"
	force := r.URL.Query().Get("force") == "true"

	if err := storage.DeleteSnapshot(name, recursive, force); err != nil {
		logger.Info("Error deleting snapshot %s: %v", name, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"snapshot": name,
	})
}

// handleAdminStorageSnapshotList lists snapshots
func (s *Server) handleAdminStorageSnapshotList(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	dataset := r.URL.Query().Get("dataset")

	var snapshots []storage.Snapshot
	var err error

	if dataset != "" {
		snapshots, err = storage.ListSnapshots(dataset)
	} else {
		snapshots, err = storage.ListAllSnapshots()
	}

	if err != nil {
		logger.Info("Error listing snapshots: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshots)
}

// handleAdminStorageSnapshotRollback rolls back to a snapshot
func (s *Server) handleAdminStorageSnapshotRollback(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req storage.RollbackOptions
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	// Verify password token
	if err := s.validateVerificationToken(r, session); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Password verification required"})
		return
	}

	if err := storage.Rollback(req); err != nil {
		logger.Info("Error rolling back to snapshot %s: %v", req.Snapshot, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"snapshot": req.Snapshot,
	})
}

// --- Disk Handlers ---

// handleAdminStorageDisksAvailable lists available disks
func (s *Server) handleAdminStorageDisksAvailable(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	disks, err := storage.GetAvailableDisks()
	if err != nil {
		logger.Info("Error listing available disks: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(disks)
}

// handleAdminStorageDiskFormat formats a disk
func (s *Server) handleAdminStorageDiskFormat(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req storage.FormatOptions
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	// Verify password token
	if err := s.validateVerificationToken(r, session); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Password verification required"})
		return
	}

	if err := storage.FormatDisk(req); err != nil {
		logger.Info("Error formatting disk %s: %v", req.Device, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Mount the disk if requested
	var mountError string
	var fstabError string
	if req.Mount && req.MountPath != "" {
		if err := storage.MountDisk(req.Device, req.MountPath, req.SharedAccess); err != nil {
			logger.Info("Warning: Failed to mount disk %s at %s: %v", req.Device, req.MountPath, err)
			mountError = err.Error()
		} else {
			logger.Info("Mounted disk %s at %s (shared: %v)", req.Device, req.MountPath, req.SharedAccess)

			// Add to fstab if persistent mount requested
			if req.Persistent {
				if err := storage.AddToFstab(req.Device, req.MountPath, req.SharedAccess); err != nil {
					logger.Info("Warning: Failed to add to fstab: %v", err)
					fstabError = err.Error()
				} else {
					logger.Info("Added to fstab: %s at %s", req.Device, req.MountPath)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"success":    true,
		"device":     req.Device,
		"mounted":    req.Mount && mountError == "",
		"mount_path": req.MountPath,
	}
	if mountError != "" {
		response["mount_error"] = mountError
	}
	if fstabError != "" {
		response["fstab_error"] = fstabError
	}
	json.NewEncoder(w).Encode(response)
}

// handleAdminStorageDiskUnmount unmounts and optionally ejects a disk
func (s *Server) handleAdminStorageDiskUnmount(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MountPath string `json:"mount_path"`
		Eject     bool   `json:"eject"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	if err := storage.UnmountDisk(req.MountPath, req.Eject); err != nil {
		logger.Info("Error unmounting disk at %s: %v", req.MountPath, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	action := "unmounted"
	if req.Eject {
		action = "ejected"
	}
	logger.Info("Disk %s at %s", action, req.MountPath)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"mount_path": req.MountPath,
		"ejected":    req.Eject,
	})
}

// handleAdminStorageDiskMount mounts an already formatted disk
func (s *Server) handleAdminStorageDiskMount(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.GetSessionFromContext(r); !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Device       string `json:"device"`
		MountPath    string `json:"mount_path"`
		Persistent   bool   `json:"persistent"`
		SharedAccess bool   `json:"shared_access"` // If true, all users can read/write
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	if req.Device == "" || req.MountPath == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Device and mount_path are required"})
		return
	}

	if err := storage.MountDisk(req.Device, req.MountPath, req.SharedAccess); err != nil {
		logger.Info("Error mounting disk %s at %s: %v", req.Device, req.MountPath, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Add to fstab if persistent mount requested
	if req.Persistent {
		if err := storage.AddToFstab(req.Device, req.MountPath, req.SharedAccess); err != nil {
			// Mount succeeded, but fstab failed - log warning but don't fail the request
			logger.Info("Warning: Mounted disk but failed to add to fstab: %v", err)
		}
	}

	logger.Info("Mounted disk %s at %s (persistent: %v, shared: %v)", req.Device, req.MountPath, req.Persistent, req.SharedAccess)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"device":     req.Device,
		"mount_path": req.MountPath,
	})
}

// handleAdminStorageDiskWipe wipes a disk
func (s *Server) handleAdminStorageDiskWipe(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req storage.WipeOptions
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	// Verify password token
	if err := s.validateVerificationToken(r, session); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Password verification required"})
		return
	}

	if err := storage.WipeDisk(req); err != nil {
		logger.Info("Error wiping disk %s: %v", req.Device, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"device":  req.Device,
	})
}
