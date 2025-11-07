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
