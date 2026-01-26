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

	// Skip virtual filesystems
	virtualFS := []string{"tmpfs", "devtmpfs", "sysfs", "proc", "cgroup", "overlay", "squashfs"}
	for _, vfs := range virtualFS {
		if fsType == vfs {
			return false
		}
	}

	// Skip system partitions (regardless of device name)
	systemMounts := []string{"/", "/boot", "/boot/efi", "/home", "/var", "/tmp", "/usr", "/opt"}
	for _, sys := range systemMounts {
		if mountPoint == sys {
			return false
		}
	}

	// Accept drives mounted in common external mount points
	// This works regardless of device name (sda, sdb, nvme, etc.)
	externalPaths := []string{"/media/", "/mnt/", "/run/media/"}
	for _, prefix := range externalPaths {
		if strings.HasPrefix(mountPoint, prefix) {
			return true
		}
	}

	return false
}

// getSystemDisk returns the base disk name containing the root filesystem
// e.g., "nvme0n1" or "sda"
func getSystemDisk() string {
	// Find device mounted at /
	cmd := exec.Command("findmnt", "-n", "-o", "SOURCE", "/")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	device := strings.TrimSpace(string(output))
	// e.g., /dev/nvme0n1p3 or /dev/sda1

	// Extract base disk name
	baseName := filepath.Base(device)

	// Handle NVMe: nvme0n1p3 -> nvme0n1
	if strings.HasPrefix(baseName, "nvme") {
		if idx := strings.LastIndex(baseName, "p"); idx > 0 {
			return baseName[:idx]
		}
		return baseName
	}

	// Handle sd*: sda1 -> sda
	if strings.HasPrefix(baseName, "sd") {
		// Remove trailing digits (partition number)
		for i := len(baseName) - 1; i >= 0; i-- {
			if baseName[i] < '0' || baseName[i] > '9' {
				return baseName[:i+1]
			}
		}
	}

	return baseName
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

// UnmountedDisk represents a disk that is not mounted (can be formatted)
type UnmountedDisk struct {
	Name        string `json:"name"`         // e.g., sdb
	Path        string `json:"path"`         // e.g., /dev/sdb
	Model       string `json:"model"`        // Disk model
	Size        int64  `json:"size"`         // Size in bytes
	SizeHuman   string `json:"size_human"`   // Human-readable size
	IsRemovable bool   `json:"is_removable"` // Whether disk is removable (USB)
	HasParts    bool   `json:"has_parts"`    // Whether disk has partitions
	Filesystem  string `json:"filesystem"`   // Current filesystem (if any)
}

// DetectUnmountedDisks returns USB/external disks that are not mounted
// These can be formatted by the user
func DetectUnmountedDisks() ([]UnmountedDisk, error) {
	var disks []UnmountedDisk

	// First, detect the system disk (the one containing / or /boot)
	systemDisk := getSystemDisk()

	// Get all block devices with details
	cmd := exec.Command("lsblk", "-d", "-n", "-b", "-o", "NAME,SIZE,MODEL,TYPE,RM,FSTYPE")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list disks: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 {
			continue
		}

		name := fields[0]
		deviceType := fields[3]
		removable := fields[4]

		// Only include whole disks
		if deviceType != "disk" {
			continue
		}

		// Skip system disk (dynamically detected)
		if name == systemDisk {
			continue
		}

		// Skip loop devices
		if strings.HasPrefix(name, "loop") {
			continue
		}

		// Skip zram devices
		if strings.HasPrefix(name, "zram") {
			continue
		}

		// Skip CD/DVD drives
		if strings.HasPrefix(name, "sr") {
			continue
		}

		// Include removable disks or sd* devices (likely external on NVMe systems)
		isRemovable := removable == "1"
		isSDDevice := strings.HasPrefix(name, "sd")

		if !isRemovable && !isSDDevice {
			continue
		}

		devicePath := "/dev/" + name

		// Check if any partition is mounted
		isMounted, _ := isDiskOrPartitionMounted(name)
		if isMounted {
			continue
		}

		disk := UnmountedDisk{
			Name:        name,
			Path:        devicePath,
			IsRemovable: isRemovable,
		}

		// Parse size
		if len(fields) >= 2 {
			if size, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				disk.Size = size
				disk.SizeHuman = FormatBytes(size)
			}
		}

		// Parse model (field 2, but may be empty)
		if len(fields) >= 3 && fields[2] != deviceType {
			disk.Model = fields[2]
		}

		// Get filesystem type (from partitions if any)
		disk.Filesystem, disk.HasParts = getDiskFilesystem(name)

		disks = append(disks, disk)
	}

	return disks, nil
}

// isDiskOrPartitionMounted checks if a disk or any of its partitions are mounted
func isDiskOrPartitionMounted(diskName string) (bool, string) {
	cmd := exec.Command("lsblk", "-n", "-o", "NAME,MOUNTPOINT", "/dev/"+diskName)
	output, err := cmd.Output()
	if err != nil {
		return false, ""
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[1] != "" {
			return true, fields[1]
		}
	}
	return false, ""
}

// getDiskFilesystem returns the filesystem type of a disk or its first partition
func getDiskFilesystem(diskName string) (string, bool) {
	cmd := exec.Command("lsblk", "-n", "-o", "NAME,FSTYPE", "/dev/"+diskName)
	output, err := cmd.Output()
	if err != nil {
		return "", false
	}

	hasParts := false
	fsType := ""
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	lineNum := 0
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		lineNum++
		if lineNum > 1 {
			hasParts = true
		}
		if len(fields) >= 2 && fields[1] != "" && fsType == "" {
			fsType = fields[1]
		}
	}
	return fsType, hasParts
}
