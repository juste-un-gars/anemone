// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/juste-un-gars/anemone/internal/storage"
)

// StorageOption represents a storage configuration option
type StorageOption struct {
	Type        string `json:"type"`        // "default", "zfs_existing", "zfs_new", "custom"
	Name        string `json:"name"`        // Display name
	Description string `json:"description"` // Description for UI
	Path        string `json:"path"`        // Path (for existing options)
}

// DiskInfo represents information about an available disk
type DiskInfo struct {
	Name       string `json:"name"`        // e.g., "sda", "nvme0n1"
	Path       string `json:"path"`        // e.g., "/dev/sda"
	Size       int64  `json:"size"`        // Size in bytes
	SizeHuman  string `json:"size_human"`  // Human-readable size
	Model      string `json:"model"`       // Disk model
	Serial     string `json:"serial"`      // Serial number
	Type       string `json:"type"`        // "hdd", "ssd", "nvme"
	InUse      bool   `json:"in_use"`      // Whether disk is in use
	Partitions int    `json:"partitions"`  // Number of partitions
}

// GetStorageOptions returns available storage options for the setup wizard
func GetStorageOptions() ([]StorageOption, error) {
	options := []StorageOption{
		{
			Type:        "default",
			Name:        "Répertoire par défaut",
			Description: "/srv/anemone - Utilise le système de fichiers existant",
			Path:        "/srv/anemone",
		},
	}

	// Check for existing ZFS pools
	pools, err := storage.ListZFSPools()
	if err == nil && len(pools) > 0 {
		for _, pool := range pools {
			// Get mountpoint from root dataset
			mountpoint, _ := storage.GetDatasetProperty(pool.Name, "mountpoint")
			if mountpoint == "" {
				mountpoint = "/" + pool.Name
			}
			options = append(options, StorageOption{
				Type:        "zfs_existing",
				Name:        fmt.Sprintf("Pool ZFS: %s", pool.Name),
				Description: fmt.Sprintf("Utiliser le pool existant (%s, %s disponible)", pool.Health, pool.FreeHuman),
				Path:        mountpoint,
			})
		}
	}

	// Option to create new ZFS pool
	options = append(options, StorageOption{
		Type:        "zfs_new",
		Name:        "Nouveau pool ZFS",
		Description: "Créer un nouveau pool ZFS avec vos disques",
	})

	// Custom path option
	options = append(options, StorageOption{
		Type:        "custom",
		Name:        "Chemin personnalisé",
		Description: "Spécifier un chemin personnalisé pour les données",
	})

	return options, nil
}

// GetAvailableDisks returns disks available for ZFS pool creation
func GetAvailableDisks() ([]DiskInfo, error) {
	disks, err := storage.ListDisks()
	if err != nil {
		return nil, err
	}

	var result []DiskInfo
	for _, d := range disks {
		// Determine if disk is in use (has partitions)
		inUse := len(d.Partitions) > 0

		result = append(result, DiskInfo{
			Name:       d.Name,
			Path:       d.Path,
			Size:       int64(d.Size),
			SizeHuman:  d.SizeHuman,
			Model:      d.Model,
			Serial:     d.Serial,
			Type:       string(d.Type),
			InUse:      inUse,
			Partitions: len(d.Partitions),
		})
	}

	return result, nil
}

// SetupDefaultStorage creates the default directory structure
func SetupDefaultStorage(dataDir string, owner string) error {
	// Create main directories
	dirs := []string{
		dataDir,
		filepath.Join(dataDir, "db"),
		filepath.Join(dataDir, "shares"),
		filepath.Join(dataDir, "backups", "incoming"),
		filepath.Join(dataDir, "certs"),
		filepath.Join(dataDir, "smb"),
	}

	for _, dir := range dirs {
		if err := createDirectoryWithSudo(dir); err != nil {
			return err
		}
	}

	// Set ownership if specified
	if owner != "" {
		if err := storage.FixMountpointOwnership(dataDir, owner); err != nil {
			return fmt.Errorf("failed to set ownership: %w", err)
		}
	}

	return nil
}

// SetupZFSStorage creates a new ZFS pool and sets up Anemone on it
func SetupZFSStorage(opts ZFSSetupOptions) error {
	// Validate options
	if opts.PoolName == "" {
		return fmt.Errorf("pool name is required")
	}
	if len(opts.Devices) == 0 {
		return fmt.Errorf("at least one device is required")
	}

	// Determine RAID level based on device count and preference
	raidLevel := opts.RaidLevel
	if raidLevel == "" {
		if len(opts.Devices) == 1 {
			raidLevel = "single"
		} else if len(opts.Devices) == 2 {
			raidLevel = "mirror"
		} else {
			raidLevel = "raidz"
		}
	}

	// Create the ZFS pool
	createOpts := storage.PoolCreateOptions{
		Name:        opts.PoolName,
		Disks:       opts.Devices,
		VDevType:    raidLevel,
		Mountpoint:  opts.Mountpoint,
		Compression: "lz4",
		Owner:       opts.Owner,
	}

	if err := storage.CreatePool(createOpts); err != nil {
		return fmt.Errorf("failed to create ZFS pool: %w", err)
	}

	// Create directory structure on the pool
	dataDir := opts.Mountpoint
	if dataDir == "" {
		dataDir = "/" + opts.PoolName
	}

	return SetupDefaultStorage(dataDir, opts.Owner)
}

// ZFSSetupOptions contains options for ZFS storage setup
type ZFSSetupOptions struct {
	PoolName   string   `json:"pool_name"`
	Devices    []string `json:"devices"`
	RaidLevel  string   `json:"raid_level"` // "single", "mirror", "raidz", "raidz2"
	Mountpoint string   `json:"mountpoint"`
	Owner      string   `json:"owner"` // e.g., "anemone:anemone"
}

// SetupExistingZFS sets up Anemone on an existing ZFS pool
func SetupExistingZFS(poolName, datasetName, owner string) error {
	// Get pool info
	pools, err := storage.ListZFSPools()
	if err != nil {
		return fmt.Errorf("failed to list pools: %w", err)
	}

	var pool *storage.ZFSPool
	for i := range pools {
		if pools[i].Name == poolName {
			pool = &pools[i]
			break
		}
	}

	if pool == nil {
		return fmt.Errorf("pool %s not found", poolName)
	}

	// Create dataset for Anemone if needed
	datasetPath := poolName
	if datasetName != "" {
		datasetPath = poolName + "/" + datasetName

		// Check if dataset exists
		datasets, err := storage.ListDatasets(poolName)
		if err != nil {
			return fmt.Errorf("failed to list datasets: %w", err)
		}

		exists := false
		for _, d := range datasets {
			if d.Name == datasetPath {
				exists = true
				break
			}
		}

		if !exists {
			// Create the dataset
			createOpts := storage.DatasetCreateOptions{
				Name:        datasetPath,
				Compression: "lz4",
				Owner:       owner,
			}
			if err := storage.CreateDataset(createOpts); err != nil {
				return fmt.Errorf("failed to create dataset: %w", err)
			}
		}
	}

	// Get the mountpoint
	mountpoint, err := storage.GetDatasetProperty(datasetPath, "mountpoint")
	if err != nil {
		return fmt.Errorf("failed to get mountpoint: %w", err)
	}

	// Create directory structure
	return SetupDefaultStorage(mountpoint, owner)
}

// SetupCustomStorage sets up Anemone at a custom path
func SetupCustomStorage(dataDir, sharesDir, incomingDir, owner string) error {
	// Use defaults if not specified
	if sharesDir == "" {
		sharesDir = filepath.Join(dataDir, "shares")
	}
	if incomingDir == "" {
		incomingDir = filepath.Join(dataDir, "backups", "incoming")
	}

	// Create directories
	dirs := []string{
		dataDir,
		filepath.Join(dataDir, "db"),
		sharesDir,
		incomingDir,
		filepath.Join(dataDir, "certs"),
		filepath.Join(dataDir, "smb"),
	}

	for _, dir := range dirs {
		if err := createDirectoryWithSudo(dir); err != nil {
			return err
		}
	}

	// Set ownership
	if owner != "" {
		for _, dir := range dirs {
			if err := storage.FixMountpointOwnership(dir, owner); err != nil {
				return fmt.Errorf("failed to set ownership on %s: %w", dir, err)
			}
		}
	}

	return nil
}

// SetupIncomingDirectory creates the incoming directory for backups
func SetupIncomingDirectory(incomingDir string) error {
	if incomingDir == "" {
		return fmt.Errorf("incoming directory path is required")
	}

	// Use the common helper that checks if directory exists first
	return createDirectoryWithSudo(incomingDir)
}

// ValidateStorageConfig validates the storage configuration
func ValidateStorageConfig(config SetupConfig) error {
	switch config.StorageType {
	case "default":
		// Check if /srv/anemone exists and is writable, or if /srv is writable
		defaultDir := "/srv/anemone"
		if _, err := os.Stat(defaultDir); err == nil {
			// Directory exists, check if writable
			if err := checkWritable(defaultDir); err != nil {
				return fmt.Errorf("cannot write to %s: %w", defaultDir, err)
			}
		} else {
			// Directory doesn't exist, check if parent is writable
			if err := checkWritable("/srv"); err != nil {
				return fmt.Errorf("cannot write to /srv (need to create %s): %w", defaultDir, err)
			}
		}
	case "zfs_existing":
		if config.ZFSPoolName == "" {
			return fmt.Errorf("ZFS pool name is required")
		}
	case "zfs_new":
		if config.ZFSPoolName == "" {
			return fmt.Errorf("ZFS pool name is required")
		}
		if len(config.ZFSDevices) == 0 {
			return fmt.Errorf("at least one device is required for ZFS pool")
		}
	case "custom":
		if config.DataDir == "" {
			return fmt.Errorf("data directory is required")
		}
		// Validate that we can actually create/write to the target directory
		if err := checkCanCreateDirectory(config.DataDir); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown storage type: %s", config.StorageType)
	}

	// Validate separate incoming directory if specified
	if config.SeparateIncoming {
		if config.IncomingDir == "" {
			return fmt.Errorf("incoming directory path is required when using separate incoming storage")
		}
		if err := checkCanCreateDirectory(config.IncomingDir); err != nil {
			return fmt.Errorf("incoming directory: %w", err)
		}
	}

	return nil
}

// createDirectoryWithSudo creates a directory, trying without sudo first.
// If the parent directory is writable, creates directly. Otherwise uses sudo.
// Returns a helpful error message if sudo fails due to password requirement.
func createDirectoryWithSudo(path string) error {
	// Get current user for ownership
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = os.Getenv("LOGNAME")
	}

	// Check if directory already exists
	if _, err := os.Stat(path); err == nil {
		// Directory exists, just ensure correct ownership
		if currentUser != "" {
			cmd := exec.Command("sudo", "chown", currentUser+":"+currentUser, path)
			cmd.Run() // Ignore errors
		}
		return nil
	}

	// Directory doesn't exist - try to create it without sudo first
	if err := os.MkdirAll(path, 0755); err == nil {
		// Success without sudo
		return nil
	}

	// Failed without sudo, try with sudo
	cmd := exec.Command("sudo", "mkdir", "-p", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "password") || strings.Contains(outputStr, "terminal") {
			return fmt.Errorf("cannot create directory '%s'. Please create it manually:\n\nsudo mkdir -p %s\nsudo chown %s:%s %s", path, path, currentUser, currentUser, path)
		}
		return fmt.Errorf("failed to create directory %s: %s", path, outputStr)
	}

	// Set ownership to current user so anemone can write to it
	if currentUser != "" {
		cmd := exec.Command("sudo", "chown", currentUser+":"+currentUser, path)
		cmd.Run() // Ignore errors
	}

	return nil
}

// checkWritable checks if a path is writable
func checkWritable(path string) error {
	// Try to create a temp file
	testFile := filepath.Join(path, ".anemone-write-test")
	f, err := os.Create(testFile)
	if err != nil {
		return err
	}
	f.Close()
	os.Remove(testFile)
	return nil
}

// checkCanCreateDirectory validates that we can create a directory at the given path.
// It uses sudo since Anemone runs privileged operations with sudo.
// It handles three cases:
// 1. Directory already exists -> OK
// 2. Directory doesn't exist but sudo can create it -> OK (cleans up test dir)
// 3. Cannot create directory -> Returns helpful error message
func checkCanCreateDirectory(path string) error {
	// Clean the path to avoid issues with trailing slashes
	path = filepath.Clean(path)

	// Check if directory already exists
	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("path '%s' exists but is not a directory", path)
		}
		// Directory exists, that's fine (we'll use sudo for operations anyway)
		return nil
	}

	// Directory doesn't exist - try to create it with sudo
	cmd := exec.Command("sudo", "mkdir", "-p", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		outputStr := string(output)
		// Check if it's a sudo password issue
		if strings.Contains(outputStr, "password") || strings.Contains(outputStr, "terminal") {
			currentUser := os.Getenv("USER")
			if currentUser == "" {
				currentUser = os.Getenv("LOGNAME")
			}
			if currentUser == "" {
				currentUser = "YOUR_USER"
			}
			return fmt.Errorf("directory '%s' does not exist. Please create it manually:\n\nsudo mkdir -p %s\nsudo chown %s:%s %s", path, path, currentUser, currentUser, path)
		}
		return fmt.Errorf("cannot create directory '%s': %s", path, outputStr)
	}

	// Clean up - remove the test directory with sudo
	cmd = exec.Command("sudo", "rmdir", path)
	cmd.Run() // Ignore errors, directory might have parent dirs we created

	return nil
}
