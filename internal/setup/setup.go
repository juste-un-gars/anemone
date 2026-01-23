// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package setup handles the initial setup wizard for Anemone.
// It detects when setup is needed and manages the setup flow.
package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// SetupState represents the current state of the setup wizard (internal, with mutex)
type SetupState struct {
	mu sync.RWMutex `json:"-"`

	// Whether setup mode is active
	Active bool `json:"active"`

	// Completed steps
	StorageConfigured bool `json:"storage_configured"`
	AdminCreated      bool `json:"admin_created"`
	Finalized         bool `json:"finalized"`

	// Configuration collected during setup
	Config SetupConfig `json:"config"`
}

// SetupStateView represents the setup state for external use (no mutex)
type SetupStateView struct {
	Active            bool        `json:"active"`
	StorageConfigured bool        `json:"storage_configured"`
	AdminCreated      bool        `json:"admin_created"`
	Finalized         bool        `json:"finalized"`
	Config            SetupConfig `json:"config"`
}

// SetupConfig holds the configuration collected during setup
type SetupConfig struct {
	// Storage configuration
	StorageType string `json:"storage_type"` // "default", "zfs", "custom"
	DataDir     string `json:"data_dir"`
	SharesDir   string `json:"shares_dir"`
	IncomingDir string `json:"incoming_dir"`

	// ZFS configuration (if StorageType == "zfs")
	ZFSPoolName   string   `json:"zfs_pool_name,omitempty"`
	ZFSDevices    []string `json:"zfs_devices,omitempty"`
	ZFSRaidLevel  string   `json:"zfs_raid_level,omitempty"` // "single", "mirror", "raidz", "raidz2"
	ZFSMountpoint string   `json:"zfs_mountpoint,omitempty"`

	// Separate incoming storage (optional)
	SeparateIncoming    bool   `json:"separate_incoming"`
	IncomingStorageType string `json:"incoming_storage_type,omitempty"` // "same", "zfs", "custom"
	IncomingZFSPool     string `json:"incoming_zfs_pool,omitempty"`

	// Admin account
	AdminUsername string `json:"admin_username,omitempty"`
	// Note: password is not stored in state, only used during setup
}

// Manager handles setup state and operations
type Manager struct {
	state    *SetupState
	stateDir string
}

// NewManager creates a new setup manager
func NewManager(dataDir string) *Manager {
	return &Manager{
		state: &SetupState{
			Active: false,
			Config: SetupConfig{
				StorageType: "default",
				DataDir:     dataDir,
			},
		},
		stateDir: dataDir,
	}
}

// IsSetupNeeded checks if setup is required
// Setup is needed if:
// 1. Database doesn't exist (first run)
// 2. ANEMONE_SETUP_MODE=true environment variable
// 3. Setup state file exists and is not finalized
// 4. .needs-setup marker file exists (e.g., after OS reinstall with existing data)
func IsSetupNeeded(dataDir string) bool {
	// Check environment variable
	if os.Getenv("ANEMONE_SETUP_MODE") == "true" {
		return true
	}

	// Check for .needs-setup marker file (created by cleanup script or manually)
	markerPath := filepath.Join(dataDir, ".needs-setup")
	if _, err := os.Stat(markerPath); err == nil {
		return true
	}

	// Check if database exists
	dbPath := filepath.Join(dataDir, "db", "anemone.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return true
	}

	// Check if setup state file exists and is not finalized
	statePath := filepath.Join(dataDir, ".setup-state.json")
	if _, err := os.Stat(statePath); err == nil {
		state, err := loadState(statePath)
		if err == nil && state.Active && !state.Finalized {
			return true
		}
	}

	return false
}

// Start activates setup mode
func (m *Manager) Start() error {
	m.state.mu.Lock()
	defer m.state.mu.Unlock()

	m.state.Active = true
	return m.saveState()
}

// GetState returns the current setup state (thread-safe copy without mutex)
func (m *Manager) GetState() SetupStateView {
	m.state.mu.RLock()
	defer m.state.mu.RUnlock()

	// Return a copy without the mutex
	return SetupStateView{
		Active:            m.state.Active,
		StorageConfigured: m.state.StorageConfigured,
		AdminCreated:      m.state.AdminCreated,
		Finalized:         m.state.Finalized,
		Config:            m.state.Config,
	}
}

// IsActive returns whether setup mode is active
func (m *Manager) IsActive() bool {
	m.state.mu.RLock()
	defer m.state.mu.RUnlock()
	return m.state.Active
}

// SetStorageConfig saves the storage configuration
func (m *Manager) SetStorageConfig(config SetupConfig) error {
	m.state.mu.Lock()
	defer m.state.mu.Unlock()

	m.state.Config = config
	m.state.StorageConfigured = true
	return m.saveState()
}

// SetAdminCreated marks admin account as created
func (m *Manager) SetAdminCreated(username string) error {
	m.state.mu.Lock()
	defer m.state.mu.Unlock()

	m.state.Config.AdminUsername = username
	m.state.AdminCreated = true
	return m.saveState()
}

// Finalize completes the setup process
func (m *Manager) Finalize() error {
	m.state.mu.Lock()
	defer m.state.mu.Unlock()

	m.state.Finalized = true
	m.state.Active = false
	return m.saveState()
}

// Cleanup removes setup files after successful setup
func (m *Manager) Cleanup() error {
	// Remove setup state file
	statePath := filepath.Join(m.stateDir, ".setup-state.json")
	os.Remove(statePath) // Ignore error if doesn't exist

	// Remove .needs-setup marker file
	markerPath := filepath.Join(m.stateDir, ".needs-setup")
	os.Remove(markerPath) // Ignore error if doesn't exist

	return nil
}

// LoadState loads the setup state from disk
func (m *Manager) LoadState() error {
	statePath := filepath.Join(m.stateDir, ".setup-state.json")
	state, err := loadState(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No state file, use defaults
		}
		return err
	}

	m.state.mu.Lock()
	defer m.state.mu.Unlock()
	m.state = state
	return nil
}

// saveState saves the current state to disk
func (m *Manager) saveState() error {
	statePath := filepath.Join(m.stateDir, ".setup-state.json")

	// Ensure directory exists
	if err := os.MkdirAll(m.stateDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0600)
}

// loadState loads state from a file
func loadState(path string) (*SetupState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state SetupState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}
