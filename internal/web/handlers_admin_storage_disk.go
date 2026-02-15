// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// This file contains disk-related handlers: format, mount, unmount, wipe.

package web

import (
	"encoding/json"
	"net/http"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/storage"
)

// handleAdminStorageDisksAvailable lists available disks
func (s *Server) handleAdminStorageDisksAvailable(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	disks, err := storage.GetAvailableDisks()
	if err != nil {
		logger.Info("Error listing available disks", "error", err)
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
		logger.Info("Error formatting disk", "device", req.Device, "error", err)
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
			logger.Info("Warning: Failed to mount disk at", "device", req.Device, "mount_path", req.MountPath, "error", err)
			mountError = err.Error()
		} else {
			logger.Info("Mounted disk", "device", req.Device, "mount_path", req.MountPath, "shared_access", req.SharedAccess)

			// Add to fstab if persistent mount requested
			if req.Persistent {
				if err := storage.AddToFstab(req.Device, req.MountPath, req.SharedAccess); err != nil {
					logger.Info("Warning: Failed to add to fstab", "error", err)
					fstabError = err.Error()
				} else {
					logger.Info("Added to fstab: at", "device", req.Device, "mount_path", req.MountPath)
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
		logger.Info("Error unmounting disk at", "mount_path", req.MountPath, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	action := "unmounted"
	if req.Eject {
		action = "ejected"
	}
	logger.Info("Disk at", "action", action, "mount_path", req.MountPath)

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
		logger.Info("Error mounting disk at", "device", req.Device, "mount_path", req.MountPath, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Add to fstab if persistent mount requested
	if req.Persistent {
		if err := storage.AddToFstab(req.Device, req.MountPath, req.SharedAccess); err != nil {
			// Mount succeeded, but fstab failed - log warning but don't fail the request
			logger.Info("Warning: Mounted disk but failed to add to fstab", "error", err)
		}
	}

	logger.Info("Mounted disk", "device", req.Device, "mount_path", req.MountPath, "persistent", req.Persistent, "shared_access", req.SharedAccess)
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
		logger.Info("Error wiping disk", "device", req.Device, "error", err)
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
