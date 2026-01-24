// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package usbbackup

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// DetectDrives scans for mounted USB/external drives
// Returns a list of detected drives that could be used for backup
func DetectDrives() ([]DriveInfo, error) {
	var drives []DriveInfo

	// Read /proc/mounts to find mounted filesystems
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, fmt.Errorf("failed to read mounts: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		device := fields[0]
		mountPoint := fields[1]
		fsType := fields[2]

		// Skip non-physical filesystems and system mounts
		if !isExternalDrive(device, mountPoint, fsType) {
			continue
		}

		drive := DriveInfo{
			DevicePath:  device,
			MountPath:   mountPoint,
			Filesystem:  fsType,
			IsRemovable: isRemovableDevice(device),
		}

		// Get drive label
		drive.Label = getDriveLabel(device)

		// Get disk space info
		drive.TotalBytes, drive.FreeBytes = getDiskSpace(mountPoint)

		drives = append(drives, drive)
	}

	return drives, nil
}

// isExternalDrive checks if a mount point is likely an external/USB drive
func isExternalDrive(device, mountPoint, fsType string) bool {
	// Must be a block device
	if !strings.HasPrefix(device, "/dev/") {
		return false
	}

	// Skip system partitions
	systemMounts := []string{"/", "/boot", "/home", "/var", "/tmp", "/usr", "/opt"}
	for _, sys := range systemMounts {
		if mountPoint == sys {
			return false
		}
	}

	// Skip internal system devices (typically sda on most systems)
	// But allow sdb, sdc, etc. and nvme secondary drives
	if strings.HasPrefix(device, "/dev/sda") {
		return false
	}

	// Skip virtual filesystems
	virtualFS := []string{"tmpfs", "devtmpfs", "sysfs", "proc", "cgroup", "overlay", "squashfs"}
	for _, vfs := range virtualFS {
		if fsType == vfs {
			return false
		}
	}

	// Common external drive mount points
	externalPaths := []string{"/media/", "/mnt/", "/run/media/"}
	for _, prefix := range externalPaths {
		if strings.HasPrefix(mountPoint, prefix) {
			return true
		}
	}

	// USB drives on /dev/sd[b-z] are usually external
	if len(device) >= 8 && strings.HasPrefix(device, "/dev/sd") {
		letter := device[7]
		if letter >= 'b' && letter <= 'z' {
			return true
		}
	}

	return false
}

// isRemovableDevice checks if a device is marked as removable
func isRemovableDevice(device string) bool {
	// Extract base device (e.g., /dev/sdb1 -> sdb)
	baseName := filepath.Base(device)
	// Remove partition number
	for i := len(baseName) - 1; i >= 0; i-- {
		if baseName[i] < '0' || baseName[i] > '9' {
			baseName = baseName[:i+1]
			break
		}
	}

	// Check /sys/block/{device}/removable
	removablePath := fmt.Sprintf("/sys/block/%s/removable", baseName)
	data, err := os.ReadFile(removablePath)
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(data)) == "1"
}

// getDriveLabel gets the label of a drive using lsblk
func getDriveLabel(device string) string {
	cmd := exec.Command("lsblk", "-n", "-o", "LABEL", device)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getDiskSpace returns total and free bytes for a mount point
func getDiskSpace(mountPoint string) (total, free int64) {
	cmd := exec.Command("df", "-B1", mountPoint)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0, 0
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return 0, 0
	}

	total, _ = strconv.ParseInt(fields[1], 10, 64)
	free, _ = strconv.ParseInt(fields[3], 10, 64)
	return total, free
}

// GetDriveInfo returns detailed information about a specific drive
func GetDriveInfo(mountPath string) (*DriveInfo, error) {
	drives, err := DetectDrives()
	if err != nil {
		return nil, err
	}

	for _, drive := range drives {
		if drive.MountPath == mountPath {
			return &drive, nil
		}
	}

	return nil, fmt.Errorf("drive not found: %s", mountPath)
}

// FormatBytes formats bytes into human-readable string
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
