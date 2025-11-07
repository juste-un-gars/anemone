// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package quota

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// QuotaManager is the universal interface for filesystem quota enforcement
type QuotaManager interface {
	// CreateQuotaDir creates a directory with quota enforcement
	// For Btrfs: creates a subvolume
	// For ext4/xfs: creates a regular dir with project quota
	// For ZFS: creates a dataset
	CreateQuotaDir(path string, limitGB int) error

	// UpdateQuota updates the quota limit for an existing directory
	UpdateQuota(path string, limitGB int) error

	// GetUsage returns the current usage in bytes for a directory
	GetUsage(path string) (usedBytes, limitBytes int64, err error)

	// RemoveQuotaDir removes a directory and its quota
	RemoveQuotaDir(path string) error
}

// NewQuotaManager creates a QuotaManager based on the filesystem type
func NewQuotaManager(basePath string) (QuotaManager, error) {
	fsType, err := detectFilesystem(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect filesystem: %w", err)
	}

	switch fsType {
	case "btrfs":
		return &BtrfsQuotaManager{}, nil
	default:
		// For non-Btrfs filesystems, use fallback mode (no kernel quota enforcement)
		fmt.Printf("⚠️  Warning: Filesystem '%s' detected. Quota enforcement requires Btrfs.\n", fsType)
		fmt.Printf("   Anemone will work but quotas will NOT be enforced by the kernel.\n")
		fmt.Printf("   For full quota support, please use Btrfs filesystem.\n")
		return &FallbackQuotaManager{}, nil
	}
}

// detectFilesystem detects the filesystem type for a given path
func detectFilesystem(path string) (string, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return "", fmt.Errorf("statfs failed: %w", err)
	}

	// Magic numbers from /usr/include/linux/magic.h
	switch stat.Type {
	case 0x9123683E: // BTRFS_SUPER_MAGIC
		return "btrfs", nil
	case 0xEF53: // EXT4_SUPER_MAGIC
		return "ext4", nil
	case 0x58465342: // XFS_SUPER_MAGIC
		return "xfs", nil
	case 0x2FC12FC1: // ZFS_SUPER_MAGIC
		return "zfs", nil
	default:
		return fmt.Sprintf("unknown(0x%x)", stat.Type), nil
	}
}

// ============================================================================
// Btrfs Implementation
// ============================================================================

// BtrfsQuotaManager manages quotas using Btrfs subvolumes and qgroups
type BtrfsQuotaManager struct{}

// CreateQuotaDir creates a Btrfs subvolume with quota enabled
func (m *BtrfsQuotaManager) CreateQuotaDir(path string, limitGB int) error {
	// Ensure parent directory exists
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Check if subvolume already exists
	if isSubvolume(path) {
		fmt.Printf("Subvolume already exists: %s\n", path)
		return m.UpdateQuota(path, limitGB)
	}

	// Check if path exists as regular directory
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			// Regular directory exists, need to convert
			return fmt.Errorf("regular directory exists at %s, cannot create subvolume (migration needed)", path)
		}
		return fmt.Errorf("file exists at %s, cannot create subvolume", path)
	}

	// Create subvolume
	cmd := exec.Command("sudo", "btrfs", "subvolume", "create", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create subvolume: %w\nOutput: %s", err, output)
	}

	// Enable quota on the filesystem (if not already enabled)
	if err := m.enableQuotaOnFS(path); err != nil {
		fmt.Printf("Warning: quota enable failed (may already be enabled): %v\n", err)
	}

	// Set quota limit
	if err := m.UpdateQuota(path, limitGB); err != nil {
		return fmt.Errorf("failed to set quota: %w", err)
	}

	return nil
}

// UpdateQuota updates the quota limit for a Btrfs subvolume
func (m *BtrfsQuotaManager) UpdateQuota(path string, limitGB int) error {
	if !isSubvolume(path) {
		return fmt.Errorf("%s is not a Btrfs subvolume", path)
	}

	limitBytes := int64(limitGB) * 1024 * 1024 * 1024

	// Set exclusive limit (excl = data only, no metadata/snapshots)
	cmd := exec.Command("sudo", "btrfs", "qgroup", "limit", fmt.Sprintf("%d", limitBytes), path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set quota limit: %w\nOutput: %s", err, output)
	}

	return nil
}

// GetUsage returns current usage for a Btrfs subvolume
func (m *BtrfsQuotaManager) GetUsage(path string) (usedBytes, limitBytes int64, err error) {
	if !isSubvolume(path) {
		return 0, 0, fmt.Errorf("%s is not a Btrfs subvolume", path)
	}

	// Get subvolume ID
	cmd := exec.Command("sudo", "btrfs", "subvolume", "show", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get subvolume info: %w", err)
	}

	// Parse subvolume ID
	var subvolID string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Subvolume ID:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				subvolID = parts[2]
				break
			}
		}
	}

	if subvolID == "" {
		return 0, 0, fmt.Errorf("failed to parse subvolume ID")
	}

	// Get qgroup info
	cmd = exec.Command("sudo", "btrfs", "qgroup", "show", "-r", "--raw", path)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get qgroup info: %w", err)
	}

	// Parse qgroup output
	scanner = bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "0/"+subvolID) {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				// Format: qgroupid rfer excl [max_rfer max_excl]
				used, _ := strconv.ParseInt(fields[2], 10, 64) // excl (exclusive bytes)
				limit := int64(0)
				if len(fields) >= 5 {
					limit, _ = strconv.ParseInt(fields[4], 10, 64) // max_excl
				}
				return used, limit, nil
			}
		}
	}

	return 0, 0, fmt.Errorf("qgroup not found for subvolume")
}

// RemoveQuotaDir removes a Btrfs subvolume
func (m *BtrfsQuotaManager) RemoveQuotaDir(path string) error {
	if !isSubvolume(path) {
		// Not a subvolume, just remove as regular directory
		return os.RemoveAll(path)
	}

	cmd := exec.Command("sudo", "btrfs", "subvolume", "delete", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete subvolume: %w\nOutput: %s", err, output)
	}

	return nil
}

// enableQuotaOnFS enables quota on the Btrfs filesystem
func (m *BtrfsQuotaManager) enableQuotaOnFS(path string) error {
	// Get filesystem root
	cmd := exec.Command("df", "-P", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get filesystem: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return fmt.Errorf("unexpected df output")
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 6 {
		return fmt.Errorf("unexpected df output format")
	}

	mountPoint := fields[5]

	// Enable quota
	cmd = exec.Command("sudo", "btrfs", "quota", "enable", mountPoint)
	output, err = cmd.CombinedOutput()
	if err != nil {
		// Already enabled is not an error
		if strings.Contains(string(output), "already enabled") {
			return nil
		}
		return fmt.Errorf("failed to enable quota: %w\nOutput: %s", err, output)
	}

	return nil
}

// isSubvolume checks if a path is a Btrfs subvolume
func isSubvolume(path string) bool {
	cmd := exec.Command("sudo", "btrfs", "subvolume", "show", path)
	return cmd.Run() == nil
}

// ============================================================================
// ext4/xfs Project Quotas Implementation
// ============================================================================

// ProjectQuotaManager manages quotas using project quotas (ext4/xfs)
type ProjectQuotaManager struct{}

// CreateQuotaDir creates a directory with project quota enforcement
func (m *ProjectQuotaManager) CreateQuotaDir(path string, limitGB int) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Get or create project ID for this path
	projectID, err := m.getOrCreateProjectID(path)
	if err != nil {
		return fmt.Errorf("failed to get project ID: %w", err)
	}

	// Set project ID on directory
	if err := m.setProjectID(path, projectID); err != nil {
		return fmt.Errorf("failed to set project ID: %w", err)
	}

	// Set quota limit
	if err := m.UpdateQuota(path, limitGB); err != nil {
		return fmt.Errorf("failed to set quota: %w", err)
	}

	return nil
}

// UpdateQuota updates the quota limit for a directory
func (m *ProjectQuotaManager) UpdateQuota(path string, limitGB int) error {
	projectID, err := m.getProjectID(path)
	if err != nil {
		return fmt.Errorf("failed to get project ID: %w", err)
	}

	limitBytes := int64(limitGB) * 1024 * 1024 * 1024
	mountPoint, fsType, err := m.getMountPoint(path)
	if err != nil {
		return fmt.Errorf("failed to get mount point: %w", err)
	}

	// Set quota based on filesystem type
	switch fsType {
	case "xfs":
		// XFS uses xfs_quota
		cmd := exec.Command("sudo", "xfs_quota", "-x", "-c",
			fmt.Sprintf("limit -p bhard=%d %d", limitBytes, projectID),
			mountPoint)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set XFS quota: %w\nOutput: %s", err, output)
		}
	case "ext4":
		// ext4 uses setquota
		cmd := exec.Command("sudo", "setquota", "-P",
			fmt.Sprintf("%d", projectID),
			"0", // soft block limit (0 = no soft limit)
			fmt.Sprintf("%d", limitBytes/1024), // hard block limit in KB
			"0", // soft inode limit
			"0", // hard inode limit (0 = no limit)
			mountPoint)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set ext4 quota: %w\nOutput: %s", err, output)
		}
	default:
		return fmt.Errorf("unsupported filesystem for project quotas: %s", fsType)
	}

	return nil
}

// GetUsage returns current usage for a directory with project quota
func (m *ProjectQuotaManager) GetUsage(path string) (usedBytes, limitBytes int64, err error) {
	projectID, err := m.getProjectID(path)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get project ID: %w", err)
	}

	mountPoint, fsType, err := m.getMountPoint(path)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get mount point: %w", err)
	}

	switch fsType {
	case "xfs":
		return m.getXFSQuotaUsage(mountPoint, projectID)
	case "ext4":
		return m.getExt4QuotaUsage(mountPoint, projectID)
	default:
		return 0, 0, fmt.Errorf("unsupported filesystem: %s", fsType)
	}
}

// RemoveQuotaDir removes a directory and cleans up project quota
func (m *ProjectQuotaManager) RemoveQuotaDir(path string) error {
	// Remove project ID mapping
	if err := m.removeProjectID(path); err != nil {
		fmt.Printf("Warning: failed to remove project ID mapping: %v\n", err)
	}

	// Remove directory
	return os.RemoveAll(path)
}

// getOrCreateProjectID gets or creates a unique project ID for a path
func (m *ProjectQuotaManager) getOrCreateProjectID(path string) (int, error) {
	// Check if path already has a project ID in /etc/projects
	projectID, err := m.getProjectID(path)
	if err == nil {
		return projectID, nil
	}

	// Generate new unique project ID (use hash of path for consistency)
	projectID = m.hashPathToProjectID(path)

	// Add to /etc/projects and /etc/projid
	if err := m.addProjectIDMapping(path, projectID); err != nil {
		return 0, fmt.Errorf("failed to add project ID mapping: %w", err)
	}

	return projectID, nil
}

// getProjectID retrieves the project ID for a given path
func (m *ProjectQuotaManager) getProjectID(path string) (int, error) {
	// Read /etc/projects to find project ID
	data, err := os.ReadFile("/etc/projects")
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("project not found")
		}
		return 0, fmt.Errorf("failed to read /etc/projects: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Format: project_id:path
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && parts[1] == path {
			return strconv.Atoi(parts[0])
		}
	}

	return 0, fmt.Errorf("project ID not found for path: %s", path)
}

// setProjectID sets the project ID attribute on a directory
func (m *ProjectQuotaManager) setProjectID(path string, projectID int) error {
	// Use chattr +P (project inheritance) and set project ID
	cmd := exec.Command("sudo", "chattr", "+P", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		// chattr +P may not be supported on all filesystems, continue anyway
		fmt.Printf("Warning: chattr +P failed: %v\n", string(output))
	}

	// Set project ID using xfs_io or fsctl
	mountPoint, fsType, err := m.getMountPoint(path)
	if err != nil {
		return err
	}

	if fsType == "xfs" {
		// For XFS, use xfs_quota project setup
		cmd := exec.Command("sudo", "xfs_quota", "-x", "-c",
			fmt.Sprintf("project -s -p %s %d", path, projectID),
			mountPoint)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to setup XFS project: %w\nOutput: %s", err, output)
		}
	}
	// For ext4, project ID is set via FS_IOC_FSSETXATTR ioctl (handled by kernel when quota is enabled)

	return nil
}

// addProjectIDMapping adds a project ID mapping to /etc/projects and /etc/projid
func (m *ProjectQuotaManager) addProjectIDMapping(path string, projectID int) error {
	projectName := fmt.Sprintf("anemone_%d", projectID)

	// Add to /etc/projid (name:id)
	projidLine := fmt.Sprintf("%s:%d\n", projectName, projectID)
	cmd := exec.Command("sudo", "tee", "-a", "/etc/projid")
	cmd.Stdin = strings.NewReader(projidLine)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to update /etc/projid: %w\nOutput: %s", err, output)
	}

	// Add to /etc/projects (id:path)
	projectsLine := fmt.Sprintf("%d:%s\n", projectID, path)
	cmd = exec.Command("sudo", "tee", "-a", "/etc/projects")
	cmd.Stdin = strings.NewReader(projectsLine)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to update /etc/projects: %w\nOutput: %s", err, output)
	}

	return nil
}

// removeProjectID removes project ID mapping from config files
func (m *ProjectQuotaManager) removeProjectID(path string) error {
	projectID, err := m.getProjectID(path)
	if err != nil {
		return err // Path not found in projects file
	}

	// Remove from /etc/projects
	cmd := exec.Command("sudo", "sed", "-i", fmt.Sprintf("/%d:%s/d", projectID, strings.ReplaceAll(path, "/", "\\/")), "/etc/projects")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to update /etc/projects: %w\nOutput: %s", err, output)
	}

	// Remove from /etc/projid
	cmd = exec.Command("sudo", "sed", "-i", fmt.Sprintf("/anemone_%d:%d/d", projectID, projectID), "/etc/projid")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to update /etc/projid: %w\nOutput: %s", err, output)
	}

	return nil
}

// getMountPoint returns the mount point and filesystem type for a path
func (m *ProjectQuotaManager) getMountPoint(path string) (string, string, error) {
	cmd := exec.Command("df", "-PT", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("failed to get mount point: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return "", "", fmt.Errorf("unexpected df output")
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 7 {
		return "", "", fmt.Errorf("unexpected df output format")
	}

	fsType := fields[1]     // Filesystem type
	mountPoint := fields[6] // Mount point

	return mountPoint, fsType, nil
}

// getXFSQuotaUsage gets quota usage for XFS filesystem
func (m *ProjectQuotaManager) getXFSQuotaUsage(mountPoint string, projectID int) (usedBytes, limitBytes int64, err error) {
	cmd := exec.Command("sudo", "xfs_quota", "-x", "-c",
		fmt.Sprintf("quota -p -N -b %d", projectID),
		mountPoint)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get XFS quota: %w\nOutput: %s", err, output)
	}

	// Parse output format: used soft hard
	fields := strings.Fields(string(output))
	if len(fields) < 3 {
		return 0, 0, fmt.Errorf("unexpected xfs_quota output: %s", output)
	}

	used, _ := strconv.ParseInt(fields[0], 10, 64)
	limit, _ := strconv.ParseInt(fields[2], 10, 64)

	// Convert from KB to bytes
	return used * 1024, limit * 1024, nil
}

// getExt4QuotaUsage gets quota usage for ext4 filesystem
func (m *ProjectQuotaManager) getExt4QuotaUsage(mountPoint string, projectID int) (usedBytes, limitBytes int64, err error) {
	cmd := exec.Command("sudo", "quota", "-P", "-p",
		fmt.Sprintf("%d", projectID),
		"-w") // Raw output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get ext4 quota: %w\nOutput: %s", err, output)
	}

	// Parse quota output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, mountPoint) {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				used, _ := strconv.ParseInt(fields[1], 10, 64)
				limit, _ := strconv.ParseInt(fields[3], 10, 64)
				// Convert from KB to bytes
				return used * 1024, limit * 1024, nil
			}
		}
	}

	return 0, 0, fmt.Errorf("quota info not found in output")
}

// hashPathToProjectID creates a consistent project ID from a path
func (m *ProjectQuotaManager) hashPathToProjectID(path string) int {
	// Simple hash function to generate project ID (10000-99999 range for Anemone)
	hash := 10000
	for _, c := range path {
		hash = (hash*31 + int(c)) % 90000
	}
	return hash + 10000 // Ensure range 10000-99999
}

// ============================================================================
// ZFS Quotas Implementation
// ============================================================================

// ZFSQuotaManager manages quotas using ZFS datasets
type ZFSQuotaManager struct{}

// CreateQuotaDir creates a ZFS dataset with quota
func (m *ZFSQuotaManager) CreateQuotaDir(path string, limitGB int) error {
	// Get the parent ZFS dataset
	parentDataset, err := m.getZFSDataset(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("failed to get parent ZFS dataset: %w", err)
	}

	// Create dataset name from path
	datasetName := m.pathToDataset(parentDataset, path)

	// Check if dataset already exists
	if m.datasetExists(datasetName) {
		fmt.Printf("ZFS dataset already exists: %s\n", datasetName)
		return m.UpdateQuota(path, limitGB)
	}

	// Create the dataset
	cmd := exec.Command("sudo", "zfs", "create", datasetName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create ZFS dataset: %w\nOutput: %s", err, output)
	}

	// Set quota
	if err := m.UpdateQuota(path, limitGB); err != nil {
		return fmt.Errorf("failed to set quota: %w", err)
	}

	return nil
}

// UpdateQuota updates the quota for a ZFS dataset
func (m *ZFSQuotaManager) UpdateQuota(path string, limitGB int) error {
	datasetName, err := m.getZFSDataset(path)
	if err != nil {
		return fmt.Errorf("failed to get ZFS dataset: %w", err)
	}

	limitBytes := int64(limitGB) * 1024 * 1024 * 1024
	quotaStr := fmt.Sprintf("%d", limitBytes)

	// Set quota property
	cmd := exec.Command("sudo", "zfs", "set", fmt.Sprintf("quota=%s", quotaStr), datasetName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set ZFS quota: %w\nOutput: %s", err, output)
	}

	return nil
}

// GetUsage returns current usage for a ZFS dataset
func (m *ZFSQuotaManager) GetUsage(path string) (usedBytes, limitBytes int64, err error) {
	datasetName, err := m.getZFSDataset(path)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get ZFS dataset: %w", err)
	}

	// Get used and quota properties
	cmd := exec.Command("sudo", "zfs", "get", "-Hp", "-o", "value", "used,quota", datasetName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get ZFS properties: %w\nOutput: %s", err, output)
	}

	// Parse output (two lines: used, quota)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return 0, 0, fmt.Errorf("unexpected zfs get output: %s", output)
	}

	used, err := strconv.ParseInt(strings.TrimSpace(lines[0]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse used bytes: %w", err)
	}

	quotaStr := strings.TrimSpace(lines[1])
	var limit int64
	if quotaStr == "0" || quotaStr == "none" {
		limit = 0 // No quota set
	} else {
		limit, err = strconv.ParseInt(quotaStr, 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse quota: %w", err)
		}
	}

	return used, limit, nil
}

// RemoveQuotaDir removes a ZFS dataset
func (m *ZFSQuotaManager) RemoveQuotaDir(path string) error {
	datasetName, err := m.getZFSDataset(path)
	if err != nil {
		// Not a ZFS dataset, try regular removal
		return os.RemoveAll(path)
	}

	// Destroy dataset recursively (includes snapshots and children)
	cmd := exec.Command("sudo", "zfs", "destroy", "-r", datasetName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to destroy ZFS dataset: %w\nOutput: %s", err, output)
	}

	return nil
}

// getZFSDataset returns the ZFS dataset name for a given path
func (m *ZFSQuotaManager) getZFSDataset(path string) (string, error) {
	// Use zfs list to find the dataset mounted at this path
	cmd := exec.Command("sudo", "zfs", "list", "-H", "-o", "name,mountpoint")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to list ZFS datasets: %w", err)
	}

	// Find exact match or closest parent
	var bestMatch string
	var bestMatchLen int

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		dataset := fields[0]
		mountpoint := fields[1]

		// Check if path starts with this mountpoint
		if path == mountpoint || strings.HasPrefix(path+"/", mountpoint+"/") {
			if len(mountpoint) > bestMatchLen {
				bestMatch = dataset
				bestMatchLen = len(mountpoint)
			}
		}
	}

	if bestMatch == "" {
		return "", fmt.Errorf("no ZFS dataset found for path: %s", path)
	}

	// If exact match, return it
	cmd = exec.Command("sudo", "zfs", "list", "-H", "-o", "mountpoint", bestMatch)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get dataset mountpoint: %w", err)
	}

	mountpoint := strings.TrimSpace(string(output))
	if mountpoint == path {
		return bestMatch, nil
	}

	// Create child dataset name from remaining path
	relativePath := strings.TrimPrefix(path, mountpoint)
	relativePath = strings.Trim(relativePath, "/")
	childDataset := bestMatch + "/" + strings.ReplaceAll(relativePath, "/", "/")

	return childDataset, nil
}

// pathToDataset converts a filesystem path to a ZFS dataset name
func (m *ZFSQuotaManager) pathToDataset(parentDataset, path string) string {
	// Get parent mountpoint
	cmd := exec.Command("sudo", "zfs", "list", "-H", "-o", "mountpoint", parentDataset)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback: use basename
		return parentDataset + "/" + filepath.Base(path)
	}

	parentMount := strings.TrimSpace(string(output))
	relativePath := strings.TrimPrefix(path, parentMount)
	relativePath = strings.Trim(relativePath, "/")

	// Replace slashes with ZFS dataset separator
	datasetSuffix := strings.ReplaceAll(relativePath, "/", "/")

	return parentDataset + "/" + datasetSuffix
}

// datasetExists checks if a ZFS dataset exists
func (m *ZFSQuotaManager) datasetExists(dataset string) bool {
	cmd := exec.Command("sudo", "zfs", "list", "-H", "-o", "name", dataset)
	return cmd.Run() == nil
}

// ============================================================================
// Fallback Implementation (for non-Btrfs filesystems)
// ============================================================================

// FallbackQuotaManager provides basic directory operations without kernel quota enforcement
// Used for ext4, xfs, and other filesystems that don't have easy quota support
type FallbackQuotaManager struct{}

// CreateQuotaDir creates a regular directory (no quota enforcement)
func (m *FallbackQuotaManager) CreateQuotaDir(path string, limitGB int) error {
	// Just create a regular directory
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	fmt.Printf("ℹ️  Created directory %s (quota limit: %dGB, not enforced by kernel)\n", path, limitGB)
	return nil
}

// UpdateQuota is a no-op for fallback mode
func (m *FallbackQuotaManager) UpdateQuota(path string, limitGB int) error {
	// No kernel enforcement, just log
	fmt.Printf("ℹ️  Quota updated for %s: %dGB (not enforced by kernel)\n", path, limitGB)
	return nil
}

// GetUsage returns current disk usage using du command
func (m *FallbackQuotaManager) GetUsage(path string) (usedBytes, limitBytes int64, err error) {
	// Calculate actual usage with du
	cmd := exec.Command("du", "-sb", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If du fails, directory might not exist yet
		return 0, 0, nil
	}

	// Parse du output: "bytes	path"
	fields := strings.Fields(string(output))
	if len(fields) < 1 {
		return 0, 0, fmt.Errorf("unexpected du output: %s", output)
	}

	used, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse usage: %w", err)
	}

	// Return 0 for limit (no enforcement)
	return used, 0, nil
}

// RemoveQuotaDir removes a regular directory
func (m *FallbackQuotaManager) RemoveQuotaDir(path string) error {
	return os.RemoveAll(path)
}
