// Package storage provides ZFS pool management operations.
package storage

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// PoolCreateOptions contains options for creating a ZFS pool
type PoolCreateOptions struct {
	Name        string   `json:"name"`        // Pool name
	VDevType    string   `json:"vdev_type"`   // stripe, mirror, raidz1, raidz2, raidz3
	Disks       []string `json:"disks"`       // List of disk paths
	Force       bool     `json:"force"`       // Force creation (-f flag)
	Mountpoint  string   `json:"mountpoint"`  // Custom mountpoint (optional)
	Compression string   `json:"compression"` // Compression type: off, lz4, zstd, gzip (optional)
	Ashift      int      `json:"ashift"`      // Sector size alignment (optional, 0 for auto)
}

// ImportablePool represents a pool that can be imported
type ImportablePool struct {
	Name   string `json:"name"`
	ID     string `json:"id"`
	State  string `json:"state"`
	Status string `json:"status"`
}

// ValidatePoolName checks if a pool name is valid
func ValidatePoolName(name string) error {
	if name == "" {
		return fmt.Errorf("pool name cannot be empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("pool name too long (max 255 characters)")
	}
	// ZFS pool names must start with a letter and can contain alphanumerics, underscores, hyphens, periods, and colons
	validName := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_\-.:]*$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid pool name: must start with a letter and contain only alphanumerics, underscore, hyphen, period, or colon")
	}
	// Reserved names
	reserved := []string{"mirror", "raidz", "raidz1", "raidz2", "raidz3", "spare", "log", "cache"}
	for _, r := range reserved {
		if name == r {
			return fmt.Errorf("'%s' is a reserved name", name)
		}
	}
	return nil
}

// ValidateDiskPath checks if a disk path is valid
func ValidateDiskPath(path string) error {
	if path == "" {
		return fmt.Errorf("disk path cannot be empty")
	}
	// Must start with /dev/
	if !strings.HasPrefix(path, "/dev/") {
		return fmt.Errorf("disk path must start with /dev/")
	}
	// Basic path validation - no command injection
	validPath := regexp.MustCompile(`^/dev/[a-zA-Z0-9/_\-]+$`)
	if !validPath.MatchString(path) {
		return fmt.Errorf("invalid disk path format")
	}
	return nil
}

// ValidateMountpoint checks if a mountpoint path is valid
func ValidateMountpoint(path string) error {
	if path == "" {
		return nil // Empty is allowed (uses default)
	}
	// Must be an absolute path
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("mountpoint must be an absolute path starting with /")
	}
	// Basic path validation - no command injection
	validPath := regexp.MustCompile(`^/[a-zA-Z0-9/_\-]*$`)
	if !validPath.MatchString(path) {
		return fmt.Errorf("invalid mountpoint format: only alphanumerics, underscore, hyphen, and slash are allowed")
	}
	// Prevent dangerous paths
	dangerousPaths := []string{"/", "/bin", "/boot", "/dev", "/etc", "/lib", "/lib64", "/proc", "/root", "/sbin", "/sys", "/usr", "/var"}
	for _, dp := range dangerousPaths {
		if path == dp {
			return fmt.Errorf("cannot use system path '%s' as mountpoint", path)
		}
	}
	return nil
}

// CreatePool creates a new ZFS pool
func CreatePool(opts PoolCreateOptions) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	// Validate pool name
	if err := ValidatePoolName(opts.Name); err != nil {
		return err
	}

	// Validate mountpoint
	if err := ValidateMountpoint(opts.Mountpoint); err != nil {
		return err
	}

	// Validate disks
	if len(opts.Disks) == 0 {
		return fmt.Errorf("at least one disk is required")
	}
	for _, disk := range opts.Disks {
		if err := ValidateDiskPath(disk); err != nil {
			return fmt.Errorf("invalid disk %s: %w", disk, err)
		}
	}

	// Validate vdev type and disk count
	switch opts.VDevType {
	case "stripe", "":
		// Stripe (no redundancy) - any number of disks
	case "mirror":
		if len(opts.Disks) < 2 {
			return fmt.Errorf("mirror requires at least 2 disks")
		}
	case "raidz1", "raidz":
		if len(opts.Disks) < 2 {
			return fmt.Errorf("raidz1 requires at least 2 disks")
		}
	case "raidz2":
		if len(opts.Disks) < 3 {
			return fmt.Errorf("raidz2 requires at least 3 disks")
		}
	case "raidz3":
		if len(opts.Disks) < 4 {
			return fmt.Errorf("raidz3 requires at least 4 disks")
		}
	default:
		return fmt.Errorf("invalid vdev type: %s (use stripe, mirror, raidz1, raidz2, or raidz3)", opts.VDevType)
	}

	// Build command
	args := []string{"zpool", "create"}

	if opts.Force {
		args = append(args, "-f")
	}

	// Add options
	if opts.Mountpoint != "" {
		args = append(args, "-m", opts.Mountpoint)
	}

	// Add pool properties
	if opts.Ashift > 0 {
		args = append(args, "-o", fmt.Sprintf("ashift=%d", opts.Ashift))
	}

	// Add dataset properties (compression)
	if opts.Compression != "" && opts.Compression != "off" {
		args = append(args, "-O", fmt.Sprintf("compression=%s", opts.Compression))
	}

	// Add pool name
	args = append(args, opts.Name)

	// Add vdev type (if not stripe)
	if opts.VDevType != "" && opts.VDevType != "stripe" {
		vdevType := opts.VDevType
		if vdevType == "raidz" {
			vdevType = "raidz1"
		}
		args = append(args, vdevType)
	}

	// Add disks
	args = append(args, opts.Disks...)

	// Execute command
	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create pool: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// DestroyPool destroys a ZFS pool
func DestroyPool(name string, force bool) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidatePoolName(name); err != nil {
		return err
	}

	args := []string{"zpool", "destroy"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, name)

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to destroy pool: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// ExportPool exports a ZFS pool
func ExportPool(name string, force bool) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidatePoolName(name); err != nil {
		return err
	}

	args := []string{"zpool", "export"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, name)

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to export pool: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// ListImportablePools lists pools that can be imported
func ListImportablePools() ([]ImportablePool, error) {
	if !IsZFSAvailable() {
		return nil, fmt.Errorf("ZFS is not available on this system")
	}

	cmd := exec.Command("sudo", "zpool", "import")
	output, err := cmd.CombinedOutput()

	// zpool import returns exit code 1 if no pools found, but output may contain useful info
	outputStr := string(output)

	if strings.Contains(outputStr, "no pools available to import") {
		return []ImportablePool{}, nil
	}

	if err != nil && !strings.Contains(outputStr, "pool:") {
		return nil, fmt.Errorf("failed to list importable pools: %s - %w", strings.TrimSpace(outputStr), err)
	}

	var pools []ImportablePool
	var currentPool *ImportablePool

	scanner := bufio.NewScanner(strings.NewReader(outputStr))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "pool:") {
			if currentPool != nil {
				pools = append(pools, *currentPool)
			}
			currentPool = &ImportablePool{
				Name: strings.TrimSpace(strings.TrimPrefix(line, "pool:")),
			}
		} else if currentPool != nil {
			if strings.HasPrefix(line, "id:") {
				currentPool.ID = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
			} else if strings.HasPrefix(line, "state:") {
				currentPool.State = strings.TrimSpace(strings.TrimPrefix(line, "state:"))
			} else if strings.HasPrefix(line, "status:") {
				currentPool.Status = strings.TrimSpace(strings.TrimPrefix(line, "status:"))
			}
		}
	}

	if currentPool != nil {
		pools = append(pools, *currentPool)
	}

	return pools, nil
}

// ImportPool imports a ZFS pool
func ImportPool(name string, force bool, altRoot string) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidatePoolName(name); err != nil {
		return err
	}

	args := []string{"zpool", "import"}
	if force {
		args = append(args, "-f")
	}
	if altRoot != "" {
		args = append(args, "-R", altRoot)
	}
	args = append(args, name)

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to import pool: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// AddVDevOptions contains options for adding a vdev to a pool
type AddVDevOptions struct {
	PoolName string   `json:"pool_name"`
	VDevType string   `json:"vdev_type"` // stripe, mirror, raidz1, raidz2, raidz3, spare, log, cache
	Disks    []string `json:"disks"`
	Force    bool     `json:"force"`
}

// AddVDev adds a new vdev to an existing pool
func AddVDev(opts AddVDevOptions) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidatePoolName(opts.PoolName); err != nil {
		return err
	}

	if len(opts.Disks) == 0 {
		return fmt.Errorf("at least one disk is required")
	}

	for _, disk := range opts.Disks {
		if err := ValidateDiskPath(disk); err != nil {
			return fmt.Errorf("invalid disk %s: %w", disk, err)
		}
	}

	// Validate vdev type
	validTypes := []string{"", "stripe", "mirror", "raidz1", "raidz2", "raidz3", "spare", "log", "cache"}
	typeValid := false
	for _, t := range validTypes {
		if opts.VDevType == t {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return fmt.Errorf("invalid vdev type: %s", opts.VDevType)
	}

	args := []string{"zpool", "add"}
	if opts.Force {
		args = append(args, "-f")
	}
	args = append(args, opts.PoolName)

	// Add vdev type if specified
	if opts.VDevType != "" && opts.VDevType != "stripe" {
		args = append(args, opts.VDevType)
	}

	args = append(args, opts.Disks...)

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add vdev: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// ReplaceOptions contains options for replacing a disk
type ReplaceOptions struct {
	PoolName string `json:"pool_name"`
	OldDisk  string `json:"old_disk"`
	NewDisk  string `json:"new_disk"`
	Force    bool   `json:"force"`
}

// ReplaceDisk replaces a disk in a ZFS pool
func ReplaceDisk(opts ReplaceOptions) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidatePoolName(opts.PoolName); err != nil {
		return err
	}

	if err := ValidateDiskPath(opts.OldDisk); err != nil {
		return fmt.Errorf("invalid old disk: %w", err)
	}

	if err := ValidateDiskPath(opts.NewDisk); err != nil {
		return fmt.Errorf("invalid new disk: %w", err)
	}

	args := []string{"zpool", "replace"}
	if opts.Force {
		args = append(args, "-f")
	}
	args = append(args, opts.PoolName, opts.OldDisk, opts.NewDisk)

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to replace disk: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// GetPoolStatus returns detailed status for a pool
func GetPoolStatus(name string) (string, error) {
	if !IsZFSAvailable() {
		return "", fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidatePoolName(name); err != nil {
		return "", err
	}

	cmd := exec.Command("sudo", "zpool", "status", name)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get pool status: %w", err)
	}

	return string(output), nil
}

// SetPoolProperty sets a property on a ZFS pool
func SetPoolProperty(name, property, value string) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidatePoolName(name); err != nil {
		return err
	}

	// Validate property name (basic validation)
	validProp := regexp.MustCompile(`^[a-z_]+$`)
	if !validProp.MatchString(property) {
		return fmt.Errorf("invalid property name")
	}

	cmd := exec.Command("sudo", "zpool", "set", fmt.Sprintf("%s=%s", property, value), name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set property: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// OnlineDisk brings a disk online in a ZFS pool
func OnlineDisk(poolName, disk string) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidatePoolName(poolName); err != nil {
		return err
	}

	if err := ValidateDiskPath(disk); err != nil {
		return err
	}

	cmd := exec.Command("sudo", "zpool", "online", poolName, disk)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to online disk: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// OfflineDisk takes a disk offline in a ZFS pool
func OfflineDisk(poolName, disk string, temporary bool) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidatePoolName(poolName); err != nil {
		return err
	}

	if err := ValidateDiskPath(disk); err != nil {
		return err
	}

	args := []string{"zpool", "offline"}
	if temporary {
		args = append(args, "-t")
	}
	args = append(args, poolName, disk)

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to offline disk: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// ClearPoolErrors clears error counts for a pool
func ClearPoolErrors(poolName string) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidatePoolName(poolName); err != nil {
		return err
	}

	cmd := exec.Command("sudo", "zpool", "clear", poolName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clear errors: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}
