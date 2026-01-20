// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/juste-un-gars/anemone/internal/backup"
	"github.com/juste-un-gars/anemone/internal/i18n"
	"github.com/juste-un-gars/anemone/internal/setup"
)

// SetupWizardServer wraps setup wizard handlers
type SetupWizardServer struct {
	manager   *setup.Manager
	dataDir   string
	templates *template.Template
	lang      string

	// Pending restore data (temporary storage during wizard flow)
	pendingRestoreMu     sync.RWMutex
	pendingRestoreBackup *backup.ServerBackup
}

// NewSetupWizardServer creates a new setup wizard server
func NewSetupWizardServer(dataDir, lang string) *SetupWizardServer {
	manager := setup.NewManager(dataDir)
	manager.LoadState()

	// Create translator instance for template functions
	translator, err := i18n.New()
	if err != nil {
		log.Printf("Warning: Failed to create translator: %v", err)
	}

	// Load only setup wizard template with i18n support
	funcMap := translator.FuncMap()
	templates := template.Must(template.New("").Funcs(funcMap).ParseFiles(filepath.Join("web", "templates", "setup_wizard.html")))

	return &SetupWizardServer{
		manager:   manager,
		dataDir:   dataDir,
		templates: templates,
		lang:      lang,
	}
}

// IsSetupMode returns whether setup mode is active
func (s *SetupWizardServer) IsSetupMode() bool {
	return s.manager.IsActive()
}

// GetManager returns the setup manager
func (s *SetupWizardServer) GetManager() *setup.Manager {
	return s.manager
}

// SetupWizardData holds data for the setup wizard template
type SetupWizardData struct {
	Lang        string
	Title       string
	CurrentStep int
	State       setup.SetupStateView
}

// handleWizard serves the setup wizard page
func (s *SetupWizardServer) handleWizard(w http.ResponseWriter, r *http.Request) {
	// Get current state
	state := s.manager.GetState()

	// Determine current step based on state
	currentStep := 0 // Mode selection
	if state.StorageConfigured {
		currentStep = 4 // Admin
	}
	if state.AdminCreated {
		currentStep = 5 // Summary
	}
	// If finalized, show the final success step (step 6) with restart message
	if state.Finalized {
		currentStep = 6
	}

	// Get language from query param or default
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = s.lang
	}
	if lang == "" {
		lang = "fr"
	}

	data := SetupWizardData{
		Lang:        lang,
		Title:       i18n.T(lang, "setup_wizard.title"),
		CurrentStep: currentStep,
		State:       state,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "setup_wizard.html", data); err != nil {
		log.Printf("Error rendering setup wizard template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleStorageOptions returns available storage options
func (s *SetupWizardServer) handleStorageOptions(w http.ResponseWriter, r *http.Request) {
	options, err := setup.GetStorageOptions()
	if err != nil {
		log.Printf("Error getting storage options: %v", err)
		http.Error(w, "Failed to get storage options", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

// handleDisks returns available disks for ZFS pool creation
func (s *SetupWizardServer) handleDisks(w http.ResponseWriter, r *http.Request) {
	disks, err := setup.GetAvailableDisks()
	if err != nil {
		log.Printf("Error getting available disks: %v", err)
		http.Error(w, "Failed to get available disks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(disks)
}

// handleStorageConfig saves the storage configuration
func (s *SetupWizardServer) handleStorageConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config setup.SetupConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate configuration
	if err := setup.ValidateStorageConfig(config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Apply storage configuration based on type
	var err error
	switch config.StorageType {
	case "default":
		config.DataDir = "/srv/anemone"
		config.SharesDir = "/srv/anemone/shares"
		config.IncomingDir = "/srv/anemone/backups/incoming"
		err = setup.SetupDefaultStorage(config.DataDir, "")

	case "zfs_existing":
		err = setup.SetupExistingZFS(config.ZFSPoolName, "anemone", "")
		if err == nil {
			config.DataDir = "/" + config.ZFSPoolName + "/anemone"
			config.SharesDir = config.DataDir + "/shares"
			config.IncomingDir = config.DataDir + "/backups/incoming"
		}

	case "zfs_new":
		opts := setup.ZFSSetupOptions{
			PoolName:   config.ZFSPoolName,
			Devices:    config.ZFSDevices,
			RaidLevel:  config.ZFSRaidLevel,
			Mountpoint: config.ZFSMountpoint,
		}
		err = setup.SetupZFSStorage(opts)
		if err == nil {
			config.DataDir = config.ZFSMountpoint
			if config.DataDir == "" {
				config.DataDir = "/" + config.ZFSPoolName
			}
			config.SharesDir = config.DataDir + "/shares"
			config.IncomingDir = config.DataDir + "/backups/incoming"
		}

	case "custom":
		err = setup.SetupCustomStorage(config.DataDir, config.SharesDir, config.IncomingDir, "")

	default:
		http.Error(w, "Unknown storage type", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Error setting up storage: %v", err)
		http.Error(w, "Failed to setup storage: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save configuration
	if err := s.manager.SetStorageConfig(config); err != nil {
		log.Printf("Error saving storage config: %v", err)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"config":  config,
	})
}

// handleAdmin creates the admin account
func (s *SetupWizardServer) handleAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check that storage is configured first
	state := s.manager.GetState()
	if !state.StorageConfigured {
		http.Error(w, "Storage must be configured first", http.StatusBadRequest)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	if len(req.Username) < 3 {
		http.Error(w, "Username must be at least 3 characters", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Finalize setup (creates DB, admin user, etc.)
	result, err := setup.FinalizeSetup(setup.FinalizeOptions{
		DataDir:       state.Config.DataDir,
		SharesDir:     state.Config.SharesDir,
		IncomingDir:   state.Config.IncomingDir,
		AdminUsername: req.Username,
		AdminPassword: req.Password,
		AdminEmail:    req.Email,
	})

	if err != nil {
		log.Printf("Error finalizing setup: %v", err)
		http.Error(w, "Failed to create admin account: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Mark admin as created
	if err := s.manager.SetAdminCreated(req.Username); err != nil {
		log.Printf("Error marking admin created: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"username":       result.AdminUsername,
		"encryption_key": result.EncryptionKey,
		"sync_password":  result.SyncPassword,
	})
}

// handleFinalize completes the setup process
func (s *SetupWizardServer) handleFinalize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state := s.manager.GetState()
	if !state.StorageConfigured || !state.AdminCreated {
		http.Error(w, "Setup steps incomplete", http.StatusBadRequest)
		return
	}

	// Mark as finalized
	if err := s.manager.Finalize(); err != nil {
		log.Printf("Error finalizing setup: %v", err)
		http.Error(w, "Failed to finalize setup", http.StatusInternalServerError)
		return
	}

	// Clean up setup state file
	if err := s.manager.Cleanup(); err != nil {
		log.Printf("Warning: Failed to cleanup setup state: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"redirect": "/login",
	})
}

// handleState returns the current setup state
func (s *SetupWizardServer) handleState(w http.ResponseWriter, r *http.Request) {
	state := s.manager.GetState()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// SetupWizardMiddleware redirects to setup wizard if setup is needed
func SetupWizardMiddleware(wizardServer *SetupWizardServer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow setup routes and static files
		if strings.HasPrefix(r.URL.Path, "/setup/wizard") ||
			strings.HasPrefix(r.URL.Path, "/static") {
			next.ServeHTTP(w, r)
			return
		}

		// Check if setup wizard is active
		if wizardServer.IsSetupMode() {
			http.Redirect(w, r, "/setup/wizard", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleRestoreValidate validates and decrypts an uploaded backup
func (s *SetupWizardServer) handleRestoreValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 100MB)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get passphrase
	passphrase := r.FormValue("passphrase")
	if passphrase == "" {
		http.Error(w, "Passphrase is required", http.StatusBadRequest)
		return
	}

	// Get uploaded file
	file, _, err := r.FormFile("backup")
	if err != nil {
		http.Error(w, "Failed to get backup file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file content
	encryptedData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read backup file: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate and decrypt backup
	result, serverBackup, err := setup.ValidateBackup(encryptedData, passphrase)
	if err != nil {
		log.Printf("Backup validation failed: %v", err)
		// Return the result with error info (not http error)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	// Store backup for later execution
	s.pendingRestoreMu.Lock()
	s.pendingRestoreBackup = serverBackup
	s.pendingRestoreMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleRestoreExecute executes the restoration from pending backup
func (s *SetupWizardServer) handleRestoreExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check for pending backup
	s.pendingRestoreMu.RLock()
	serverBackup := s.pendingRestoreBackup
	s.pendingRestoreMu.RUnlock()

	if serverBackup == nil {
		http.Error(w, "No backup pending for restoration. Please validate a backup first.", http.StatusBadRequest)
		return
	}

	// Parse request for storage configuration
	var req struct {
		DataDir     string `json:"data_dir"`
		SharesDir   string `json:"shares_dir"`
		IncomingDir string `json:"incoming_dir"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Use defaults if not provided
	if req.DataDir == "" {
		req.DataDir = "/srv/anemone"
	}
	if req.SharesDir == "" {
		req.SharesDir = filepath.Join(req.DataDir, "shares")
	}
	if req.IncomingDir == "" {
		req.IncomingDir = filepath.Join(req.DataDir, "backups", "incoming")
	}

	// Execute restore
	opts := setup.RestoreOptions{
		DataDir:     req.DataDir,
		SharesDir:   req.SharesDir,
		IncomingDir: req.IncomingDir,
	}

	if err := setup.ExecuteRestore(serverBackup, opts); err != nil {
		log.Printf("Restore failed: %v", err)
		http.Error(w, "Restore failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Clear pending backup
	s.pendingRestoreMu.Lock()
	s.pendingRestoreBackup = nil
	s.pendingRestoreMu.Unlock()

	// Mark setup as finalized
	if err := s.manager.Finalize(); err != nil {
		log.Printf("Warning: Failed to finalize setup after restore: %v", err)
	}

	// Clean up setup state file
	if err := s.manager.Cleanup(); err != nil {
		log.Printf("Warning: Failed to cleanup setup state: %v", err)
	}

	log.Printf("Server restored successfully from backup: %s", serverBackup.ServerName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"server_name": serverBackup.ServerName,
		"users_count": len(serverBackup.Users),
		"peers_count": len(serverBackup.Peers),
		"redirect":    "/login",
	})
}

// RegisterWizardRoutes registers setup wizard routes
func (s *SetupWizardServer) RegisterWizardRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/setup/wizard", s.handleWizard)
	mux.HandleFunc("/setup/wizard/state", s.handleState)
	mux.HandleFunc("/setup/wizard/storage/options", s.handleStorageOptions)
	mux.HandleFunc("/setup/wizard/storage/disks", s.handleDisks)
	mux.HandleFunc("/setup/wizard/storage/config", s.handleStorageConfig)
	mux.HandleFunc("/setup/wizard/admin", s.handleAdmin)
	mux.HandleFunc("/setup/wizard/finalize", s.handleFinalize)

	// Restore endpoints
	mux.HandleFunc("/setup/wizard/restore/validate", s.handleRestoreValidate)
	mux.HandleFunc("/setup/wizard/restore/execute", s.handleRestoreExecute)
}
