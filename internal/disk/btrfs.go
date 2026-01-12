// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package disk

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// BtrfsFilesystem represents a Btrfs filesystem
type BtrfsFilesystem struct {
	Label       string            `json:"label"`
	UUID        string            `json:"uuid"`
	TotalDevices int              `json:"total_devices"`
	Used        uint64            `json:"used"`
	Devices     []BtrfsDevice     `json:"devices"`
	DataProfile string            `json:"data_profile"`
	MetaProfile string            `json:"metadata_profile"`
}

// BtrfsDevice represents a device in a Btrfs filesystem
type BtrfsDevice struct {
	DevID   int    `json:"devid"`
	Path    string `json:"path"`
	Size    uint64 `json:"size"`
	Used    uint64 `json:"used"`
}

// BtrfsBalance represents the status of a balance operation
type BtrfsBalance struct {
	Running     bool   `json:"running"`
	DataChunks  int    `json:"data_chunks"`
	MetaChunks  int    `json:"meta_chunks"`
	SysChunks   int    `json:"sys_chunks"`
	Progress    string `json:"progress"`
}

// BtrfsRAIDLevel represents Btrfs RAID levels
type BtrfsRAIDLevel string

const (
	RAIDSingle  BtrfsRAIDLevel = "single"
	RAID0       BtrfsRAIDLevel = "raid0"
	RAID1       BtrfsRAIDLevel = "raid1"
	RAID1C3     BtrfsRAIDLevel = "raid1c3"
	RAID1C4     BtrfsRAIDLevel = "raid1c4"
	RAID10      BtrfsRAIDLevel = "raid10"
	RAID5       BtrfsRAIDLevel = "raid5"
	RAID6       BtrfsRAIDLevel = "raid6"
)

// GetBtrfsFilesystems returns all Btrfs filesystems on the system
func GetBtrfsFilesystems() ([]BtrfsFilesystem, error) {
	cmd := exec.Command("sudo", "btrfs", "filesystem", "show")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run btrfs filesystem show: %w", err)
	}

	return parseBtrfsShow(string(output))
}

// parseBtrfsShow parses the output of 'btrfs filesystem show'
func parseBtrfsShow(output string) ([]BtrfsFilesystem, error) {
	var filesystems []BtrfsFilesystem
	var currentFS *BtrfsFilesystem

	lines := strings.Split(output, "\n")
	labelRe := regexp.MustCompile(`Label: '([^']+)'.*uuid: ([a-f0-9-]+)`)
	labelNoneRe := regexp.MustCompile(`Label: none.*uuid: ([a-f0-9-]+)`)
	totalRe := regexp.MustCompile(`Total devices (\d+)`)
	deviceRe := regexp.MustCompile(`devid\s+(\d+).*size\s+([\d.]+\s*[KMGTP]?i?B)\s+used\s+([\d.]+\s*[KMGTP]?i?B)\s+path\s+(.+)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for new filesystem (Label line)
		if matches := labelRe.FindStringSubmatch(line); matches != nil {
			if currentFS != nil {
				filesystems = append(filesystems, *currentFS)
			}
			currentFS = &BtrfsFilesystem{
				Label:   matches[1],
				UUID:    matches[2],
				Devices: []BtrfsDevice{},
			}
		} else if matches := labelNoneRe.FindStringSubmatch(line); matches != nil {
			if currentFS != nil {
				filesystems = append(filesystems, *currentFS)
			}
			currentFS = &BtrfsFilesystem{
				Label:   "(none)",
				UUID:    matches[1],
				Devices: []BtrfsDevice{},
			}
		}

		// Check for total devices
		if matches := totalRe.FindStringSubmatch(line); matches != nil && currentFS != nil {
			totalDevices, _ := strconv.Atoi(matches[1])
			currentFS.TotalDevices = totalDevices
		}

		// Check for device line
		if matches := deviceRe.FindStringSubmatch(line); matches != nil && currentFS != nil {
			devID, _ := strconv.Atoi(matches[1])
			size := parseBtrfsSize(matches[2])
			used := parseBtrfsSize(matches[3])
			path := strings.TrimSpace(matches[4])

			currentFS.Devices = append(currentFS.Devices, BtrfsDevice{
				DevID: devID,
				Path:  path,
				Size:  size,
				Used:  used,
			})
		}
	}

	// Add last filesystem
	if currentFS != nil {
		filesystems = append(filesystems, *currentFS)
	}

	// Get RAID profiles for each filesystem
	for i := range filesystems {
		if len(filesystems[i].Devices) > 0 {
			profile, err := getBtrfsRAIDProfile(filesystems[i].Devices[0].Path)
			if err == nil {
				filesystems[i].DataProfile = profile.Data
				filesystems[i].MetaProfile = profile.Metadata
			}
		}
	}

	return filesystems, nil
}

// parseBtrfsSize converts Btrfs size string to bytes
func parseBtrfsSize(sizeStr string) uint64 {
	sizeStr = strings.TrimSpace(sizeStr)
	// Remove 'iB' suffix if present (e.g., "1.00GiB" -> "1.00G")
	sizeStr = strings.Replace(sizeStr, "iB", "", 1)

	var value float64
	var unit string
	fmt.Sscanf(sizeStr, "%f%s", &value, &unit)

	multiplier := uint64(1)
	switch strings.ToUpper(unit) {
	case "K", "KB":
		multiplier = 1024
	case "M", "MB":
		multiplier = 1024 * 1024
	case "G", "GB":
		multiplier = 1024 * 1024 * 1024
	case "T", "TB":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "P", "PB":
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
	}

	return uint64(value * float64(multiplier))
}

// BtrfsRAIDProfile represents the RAID profile of a Btrfs filesystem
type BtrfsRAIDProfile struct {
	Data     string `json:"data"`
	Metadata string `json:"metadata"`
	System   string `json:"system"`
}

// getBtrfsRAIDProfile gets the RAID profile of a Btrfs filesystem
func getBtrfsRAIDProfile(mountPoint string) (*BtrfsRAIDProfile, error) {
	cmd := exec.Command("sudo", "btrfs", "filesystem", "usage", mountPoint)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get Btrfs usage: %w", err)
	}

	profile := &BtrfsRAIDProfile{}
	lines := strings.Split(string(output), "\n")

	dataRe := regexp.MustCompile(`Data,(\w+):`)
	metadataRe := regexp.MustCompile(`Metadata,(\w+):`)
	systemRe := regexp.MustCompile(`System,(\w+):`)

	for _, line := range lines {
		if matches := dataRe.FindStringSubmatch(line); matches != nil {
			profile.Data = strings.ToLower(matches[1])
		}
		if matches := metadataRe.FindStringSubmatch(line); matches != nil {
			profile.Metadata = strings.ToLower(matches[1])
		}
		if matches := systemRe.FindStringSubmatch(line); matches != nil {
			profile.System = strings.ToLower(matches[1])
		}
	}

	return profile, nil
}

// CreateBtrfsRAID creates a new Btrfs RAID filesystem
func CreateBtrfsRAID(label string, raidLevel BtrfsRAIDLevel, devices []string, mountPoint string) error {
	if len(devices) < 2 && raidLevel != RAIDSingle && raidLevel != RAID0 {
		return fmt.Errorf("RAID level %s requires at least 2 devices", raidLevel)
	}

	// Build command arguments
	args := []string{"btrfs", "filesystem", "mkfs"}

	if label != "" {
		args = append(args, "-L", label)
	}

	// Set RAID level for data and metadata
	if raidLevel != RAIDSingle {
		args = append(args, "-d", string(raidLevel), "-m", string(raidLevel))
	}

	args = append(args, devices...)

	// Create filesystem
	cmd := exec.Command("sudo", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create Btrfs RAID: %w\nOutput: %s", err, string(output))
	}

	// Mount the filesystem
	if mountPoint != "" {
		if err := MountBtrfs(devices[0], mountPoint); err != nil {
			return fmt.Errorf("failed to mount Btrfs: %w", err)
		}
	}

	return nil
}

// AddBtrfsDevice adds a device to an existing Btrfs filesystem
func AddBtrfsDevice(mountPoint string, devicePath string) error {
	cmd := exec.Command("sudo", "btrfs", "device", "add", devicePath, mountPoint)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add device: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// RemoveBtrfsDevice removes a device from a Btrfs filesystem
func RemoveBtrfsDevice(mountPoint string, devicePath string) error {
	cmd := exec.Command("sudo", "btrfs", "device", "remove", devicePath, mountPoint)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove device: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// BalanceBtrfs starts a balance operation to convert RAID level
func BalanceBtrfs(mountPoint string, dataProfile, metadataProfile string) error {
	args := []string{"btrfs", "balance", "start"}

	if dataProfile != "" {
		args = append(args, "-dconvert="+dataProfile)
	}
	if metadataProfile != "" {
		args = append(args, "-mconvert="+metadataProfile)
	}

	args = append(args, mountPoint)

	cmd := exec.Command("sudo", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start balance: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// GetBtrfsBalanceStatus gets the status of a balance operation
func GetBtrfsBalanceStatus(mountPoint string) (*BtrfsBalance, error) {
	cmd := exec.Command("sudo", "btrfs", "balance", "status", mountPoint)
	output, err := cmd.CombinedOutput()

	status := &BtrfsBalance{}

	// If no balance is running, command returns error
	if err != nil {
		if strings.Contains(string(output), "No balance found") {
			status.Running = false
			return status, nil
		}
		return nil, fmt.Errorf("failed to get balance status: %w", err)
	}

	status.Running = true
	// Parse balance status (implementation can be expanded)
	return status, nil
}

// MountBtrfs mounts a Btrfs filesystem
func MountBtrfs(devicePath string, mountPoint string) error {
	// Create mount point if it doesn't exist
	mkdirCmd := exec.Command("sudo", "mkdir", "-p", mountPoint)
	if err := mkdirCmd.Run(); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	// Mount the filesystem
	cmd := exec.Command("sudo", "mount", devicePath, mountPoint)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to mount: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// UnmountBtrfs unmounts a Btrfs filesystem
func UnmountBtrfs(mountPoint string) error {
	cmd := exec.Command("sudo", "umount", mountPoint)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to unmount: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// GetBtrfsDeviceStats gets device statistics for a Btrfs filesystem
func GetBtrfsDeviceStats(mountPoint string) (map[string]map[string]int64, error) {
	cmd := exec.Command("sudo", "btrfs", "device", "stats", mountPoint)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get device stats: %w", err)
	}

	// Parse device stats
	stats := make(map[string]map[string]int64)
	lines := strings.Split(string(output), "\n")

	deviceRe := regexp.MustCompile(`\[([^\]]+)\]`)
	statRe := regexp.MustCompile(`(\w+)\s+(\d+)`)

	var currentDevice string
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for device line
		if matches := deviceRe.FindStringSubmatch(line); matches != nil {
			currentDevice = matches[1]
			stats[currentDevice] = make(map[string]int64)
		}

		// Check for stat line
		if matches := statRe.FindStringSubmatch(line); matches != nil && currentDevice != "" {
			value, _ := strconv.ParseInt(matches[2], 10, 64)
			stats[currentDevice][matches[1]] = value
		}
	}

	return stats, nil
}
