// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package disk

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// FlexibleSize is a type that can unmarshal both string and number from JSON
type FlexibleSize string

// UnmarshalJSON implements custom unmarshaling for FlexibleSize
func (fs *FlexibleSize) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*fs = FlexibleSize(s)
		return nil
	}

	// Try to unmarshal as number
	var n uint64
	if err := json.Unmarshal(data, &n); err == nil {
		*fs = FlexibleSize(fmt.Sprintf("%d", n))
		return nil
	}

	return fmt.Errorf("size must be either string or number")
}

// DeviceInfo represents a block device
type DeviceInfo struct {
	Name       string       `json:"name"`        // Device name (e.g., "sda")
	Path       string       `json:"path"`        // Device path (e.g., "/dev/sda")
	Size       FlexibleSize `json:"size"`        // Size in bytes (flexible: string or number)
	SizeBytes  uint64       `json:"size_bytes"`  // Size in bytes
	Type       string       `json:"type"`        // Device type (disk, part, lvm, etc.)
	FSType     string       `json:"fstype"`      // Filesystem type
	MountPoint string       `json:"mountpoint"`  // Mount point (if mounted)
	Label      string       `json:"label"`       // Filesystem label
	UUID       string       `json:"uuid"`        // Filesystem UUID
	Model      string       `json:"model"`       // Disk model
	Serial     string       `json:"serial"`      // Disk serial number
	Children   []DeviceInfo `json:"children,omitempty"` // Child devices (partitions)
}

// FilesystemType represents a filesystem type with its magic number
type FilesystemType int

const (
	FSTypeUnknown FilesystemType = iota
	FSTypeBtrfs
	FSTypeExt4
	FSTypeXFS
	FSTypeZFS
)

// String returns the string representation of the filesystem type
func (ft FilesystemType) String() string {
	switch ft {
	case FSTypeBtrfs:
		return "btrfs"
	case FSTypeExt4:
		return "ext4"
	case FSTypeXFS:
		return "xfs"
	case FSTypeZFS:
		return "zfs"
	default:
		return "unknown"
	}
}

// ListDevices lists all block devices on the system
func ListDevices() ([]DeviceInfo, error) {
	// Use lsblk with JSON output for reliable parsing
	cmd := exec.Command("lsblk", "-J", "-b", "-o", "NAME,SIZE,TYPE,FSTYPE,MOUNTPOINT,LABEL,UUID,MODEL,SERIAL")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run lsblk: %w", err)
	}

	// Parse JSON output
	var result struct {
		BlockDevices []DeviceInfo `json:"blockdevices"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse lsblk output: %w", err)
	}

	// Add full device path
	for i := range result.BlockDevices {
		populateDevicePaths(&result.BlockDevices[i])
	}

	return result.BlockDevices, nil
}

// populateDevicePaths adds full device paths recursively
func populateDevicePaths(dev *DeviceInfo) {
	dev.Path = "/dev/" + dev.Name
	dev.SizeBytes = parseSizeToBytes(dev.Size)

	for i := range dev.Children {
		populateDevicePaths(&dev.Children[i])
	}
}

// parseSizeToBytes converts size string to bytes (lsblk -b outputs bytes as string or number)
func parseSizeToBytes(size FlexibleSize) uint64 {
	var bytes uint64
	fmt.Sscanf(string(size), "%d", &bytes)
	return bytes
}

// GetDeviceInfo gets detailed information about a specific device
func GetDeviceInfo(devicePath string) (*DeviceInfo, error) {
	// Remove /dev/ prefix if present
	deviceName := strings.TrimPrefix(devicePath, "/dev/")

	// List all devices and find the matching one
	devices, err := ListDevices()
	if err != nil {
		return nil, err
	}

	return findDevice(devices, deviceName)
}

// findDevice searches for a device by name recursively
func findDevice(devices []DeviceInfo, name string) (*DeviceInfo, error) {
	for i := range devices {
		if devices[i].Name == name {
			return &devices[i], nil
		}
		if len(devices[i].Children) > 0 {
			if found, err := findDevice(devices[i].Children, name); err == nil {
				return found, nil
			}
		}
	}
	return nil, fmt.Errorf("device not found: %s", name)
}

// DetectFilesystemType detects the filesystem type of a mounted path
func DetectFilesystemType(path string) (FilesystemType, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return FSTypeUnknown, fmt.Errorf("failed to stat filesystem: %w", err)
	}

	// Check filesystem type using magic number
	switch stat.Type {
	case 0x9123683E: // BTRFS_SUPER_MAGIC
		return FSTypeBtrfs, nil
	case 0xEF53: // EXT4_SUPER_MAGIC
		return FSTypeExt4, nil
	case 0x58465342: // XFS_SUPER_MAGIC
		return FSTypeXFS, nil
	case 0x2FC12FC1: // ZFS_SUPER_MAGIC
		return FSTypeZFS, nil
	default:
		return FSTypeUnknown, nil
	}
}

// GetAvailableDisks returns disks that can be used for RAID (not mounted, no partitions)
func GetAvailableDisks() ([]DeviceInfo, error) {
	devices, err := ListDevices()
	if err != nil {
		return nil, err
	}

	var available []DeviceInfo
	for _, dev := range devices {
		// Only consider whole disks (type=disk)
		if dev.Type == "disk" {
			// Check if disk is not mounted and has no filesystem
			if dev.MountPoint == "" && dev.FSType == "" {
				// Check if disk has no partitions
				hasPartitions := false
				for _, child := range dev.Children {
					if child.Type == "part" {
						hasPartitions = true
						break
					}
				}
				if !hasPartitions {
					available = append(available, dev)
				}
			}
		}
	}

	return available, nil
}

// GetMountedFilesystems returns all mounted filesystems with their types
func GetMountedFilesystems() (map[string]FilesystemType, error) {
	cmd := exec.Command("mount")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run mount command: %w", err)
	}

	result := make(map[string]FilesystemType)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		// Parse mount output: /dev/sda1 on /mnt/data type ext4 (rw,relatime)
		if strings.Contains(line, " on ") && strings.Contains(line, " type ") {
			parts := strings.Split(line, " ")
			if len(parts) >= 5 {
				mountPoint := parts[2]
				fsTypeStr := parts[4]

				var fsType FilesystemType
				switch fsTypeStr {
				case "btrfs":
					fsType = FSTypeBtrfs
				case "ext4":
					fsType = FSTypeExt4
				case "xfs":
					fsType = FSTypeXFS
				case "zfs":
					fsType = FSTypeZFS
				default:
					fsType = FSTypeUnknown
				}

				result[mountPoint] = fsType
			}
		}
	}

	return result, nil
}

// FormatBytes formats bytes to human-readable format
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// DiskSMARTInfo represents SMART information for a disk
type DiskSMARTInfo struct {
	DevicePath       string            `json:"device_path"`
	DeviceModel      string            `json:"device_model"`
	SerialNumber     string            `json:"serial_number"`
	FirmwareVersion  string            `json:"firmware_version"`
	Capacity         string            `json:"capacity"`
	RotationRate     string            `json:"rotation_rate"`
	SMARTSupport     bool              `json:"smart_support"`
	SMARTEnabled     bool              `json:"smart_enabled"`
	OverallHealth    string            `json:"overall_health"`
	Temperature      int               `json:"temperature"`
	PowerOnHours     int               `json:"power_on_hours"`
	PowerCycleCount  int               `json:"power_cycle_count"`
	Attributes       []SMARTAttribute  `json:"attributes,omitempty"`
	Error            string            `json:"error,omitempty"`
}

// SMARTAttribute represents a single SMART attribute
type SMARTAttribute struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Value       int    `json:"value"`
	Worst       int    `json:"worst"`
	Threshold   int    `json:"threshold"`
	RawValue    string `json:"raw_value"`
	Status      string `json:"status"` // "OK" or "FAILING"
}

// CheckSMARTInstalled checks if smartctl is available
func CheckSMARTInstalled() bool {
	cmd := exec.Command("which", "smartctl")
	return cmd.Run() == nil
}

// GetDiskSMARTInfo retrieves SMART information for a disk
func GetDiskSMARTInfo(devicePath string) (*DiskSMARTInfo, error) {
	if !CheckSMARTInstalled() {
		return nil, fmt.Errorf("smartctl not installed (install smartmontools package)")
	}

	// Run smartctl with JSON output
	cmd := exec.Command("sudo", "smartctl", "-a", "-j", devicePath)
	output, err := cmd.Output()
	if err != nil {
		// smartctl returns non-zero exit code even on success sometimes
		// Check if we got any output
		if len(output) == 0 {
			return nil, fmt.Errorf("failed to get SMART info: %w", err)
		}
	}

	// Parse JSON output
	var smartData struct {
		Device struct {
			Name     string `json:"name"`
			Type     string `json:"type"`
			Protocol string `json:"protocol"`
		} `json:"device"`
		ModelName       string `json:"model_name"`
		SerialNumber    string `json:"serial_number"`
		FirmwareVersion string `json:"firmware_version"`
		UserCapacity    struct {
			Bytes int64 `json:"bytes"`
		} `json:"user_capacity"`
		RotationRate int `json:"rotation_rate"`
		SmartStatus  struct {
			Passed bool `json:"passed"`
		} `json:"smart_status"`
		Temperature struct {
			Current int `json:"current"`
		} `json:"temperature"`
		PowerOnTime struct {
			Hours int `json:"hours"`
		} `json:"power_on_time"`
		PowerCycleCount int `json:"power_cycle_count"`
		AtaSmartData    struct {
			OfflineDataCollection struct {
				Status struct {
					Value int `json:"value"`
				} `json:"status"`
			} `json:"offline_data_collection"`
			SelfTest struct {
				Status struct {
					Value int `json:"value"`
				} `json:"status"`
			} `json:"self_test"`
		} `json:"ata_smart_data"`
		AtaSmartAttributes struct {
			Table []struct {
				ID        int    `json:"id"`
				Name      string `json:"name"`
				Value     int    `json:"value"`
				Worst     int    `json:"worst"`
				Thresh    int    `json:"thresh"`
				WhenFailed string `json:"when_failed"`
				Raw       struct {
					Value  int64  `json:"value"`
					String string `json:"string"`
				} `json:"raw"`
			} `json:"table"`
		} `json:"ata_smart_attributes"`
	}

	if err := json.Unmarshal(output, &smartData); err != nil {
		return nil, fmt.Errorf("failed to parse SMART data: %w", err)
	}

	// Build SMART info structure
	info := &DiskSMARTInfo{
		DevicePath:      devicePath,
		DeviceModel:     smartData.ModelName,
		SerialNumber:    smartData.SerialNumber,
		FirmwareVersion: smartData.FirmwareVersion,
		Capacity:        FormatBytes(uint64(smartData.UserCapacity.Bytes)),
		SMARTSupport:    true,
		SMARTEnabled:    true,
		Temperature:     smartData.Temperature.Current,
		PowerOnHours:    smartData.PowerOnTime.Hours,
		PowerCycleCount: smartData.PowerCycleCount,
	}

	// Rotation rate
	if smartData.RotationRate > 0 {
		info.RotationRate = fmt.Sprintf("%d RPM", smartData.RotationRate)
	} else {
		info.RotationRate = "SSD"
	}

	// Overall health
	if smartData.SmartStatus.Passed {
		info.OverallHealth = "PASSED"
	} else {
		info.OverallHealth = "FAILED"
	}

	// Parse SMART attributes
	for _, attr := range smartData.AtaSmartAttributes.Table {
		status := "OK"
		if attr.WhenFailed != "" && attr.WhenFailed != "-" {
			status = "FAILING"
		}

		info.Attributes = append(info.Attributes, SMARTAttribute{
			ID:        attr.ID,
			Name:      attr.Name,
			Value:     attr.Value,
			Worst:     attr.Worst,
			Threshold: attr.Thresh,
			RawValue:  attr.Raw.String,
			Status:    status,
		})
	}

	return info, nil
}

// GetAllDisksSMARTInfo retrieves SMART information for all available disks
func GetAllDisksSMARTInfo() (map[string]*DiskSMARTInfo, error) {
	if !CheckSMARTInstalled() {
		return nil, fmt.Errorf("smartctl not installed")
	}

	devices, err := ListDevices()
	if err != nil {
		return nil, err
	}

	result := make(map[string]*DiskSMARTInfo)

	// Get SMART info for each disk (not partitions)
	for _, dev := range devices {
		if dev.Type == "disk" {
			info, err := GetDiskSMARTInfo(dev.Path)
			if err != nil {
				// Store error but continue with other disks
				result[dev.Path] = &DiskSMARTInfo{
					DevicePath: dev.Path,
					Error:      err.Error(),
				}
			} else {
				result[dev.Path] = info
			}
		}
	}

	return result, nil
}
