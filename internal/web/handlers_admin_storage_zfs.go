// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// This file contains ZFS-related handlers: pools, datasets, and snapshots.

package web

import (
	"encoding/json"
	"net/http"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/storage"
)

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
		logger.Info("Error creating pool", "error", err)
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
		logger.Info("Error destroying pool", "pool_name", poolName, "error", err)
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
		logger.Info("Error exporting pool", "pool_name", poolName, "error", err)
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
		logger.Info("Error listing importable pools", "error", err)
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
		logger.Info("Error importing pool", "name", req.Name, "error", err)
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
		logger.Info("Error adding vdev to pool", "pool_name", poolName, "error", err)
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
		logger.Info("Error replacing disk in pool", "pool_name", poolName, "error", err)
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
		logger.Info("Error creating dataset", "error", err)
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
		logger.Info("Error deleting dataset", "name", name, "error", err)
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
		Name     string `json:"name"`
		Property string `json:"property"`
		Value    string `json:"value"`
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
		logger.Info("Error setting property on dataset", "name", req.Name, "error", err)
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
		logger.Info("Error listing datasets", "error", err)
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
		logger.Info("Error creating snapshot", "error", err)
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
		logger.Info("Error deleting snapshot", "name", name, "error", err)
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
		logger.Info("Error listing snapshots", "error", err)
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
		logger.Info("Error rolling back to snapshot", "snapshot", req.Snapshot, "error", err)
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
