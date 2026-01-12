// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package disk

import (
	"fmt"
)

// RAIDSystemType represents the type of RAID system
type RAIDSystemType string

const (
	RAIDTypeBtrfs RAIDSystemType = "btrfs"
	RAIDTypeZFS   RAIDSystemType = "zfs"
	RAIDTypeMDAdm RAIDSystemType = "mdadm"
	RAIDTypeNone  RAIDSystemType = "none"
)

// RAIDArrayInfo represents information about a RAID array
type RAIDArrayInfo struct {
	Name         string         `json:"name"`
	Type         RAIDSystemType `json:"type"`
	Level        string         `json:"level"`
	State        string         `json:"state"`
	TotalDevices int            `json:"total_devices"`
	ActiveDevices int           `json:"active_devices"`
	Size         uint64         `json:"size"`
	Used         uint64         `json:"used"`
	Available    uint64         `json:"available"`
	MountPoint   string         `json:"mountpoint"`
	Health       string         `json:"health"`
}

// GetAllRAIDArrays returns all RAID arrays on the system (Btrfs + ZFS)
func GetAllRAIDArrays() ([]RAIDArrayInfo, error) {
	var arrays []RAIDArrayInfo

	// Get Btrfs filesystems
	btrfsFS, err := GetBtrfsFilesystems()
	if err == nil {
		for _, fs := range btrfsFS {
			// Only include multi-device filesystems
			if fs.TotalDevices > 1 || fs.DataProfile != "single" {
				array := RAIDArrayInfo{
					Name:          fs.Label,
					Type:          RAIDTypeBtrfs,
					Level:         fs.DataProfile,
					TotalDevices:  fs.TotalDevices,
					ActiveDevices: len(fs.Devices),
					State:         "active",
					Health:        "healthy",
				}

				// Calculate sizes
				for _, dev := range fs.Devices {
					array.Size += dev.Size
					array.Used += dev.Used
				}
				array.Available = array.Size - array.Used

				// Get mount point from first device
				if len(fs.Devices) > 0 {
					// Try to find mount point
					if mounts, err := GetMountedFilesystems(); err == nil {
						for mp, fsType := range mounts {
							if fsType == FSTypeBtrfs {
								array.MountPoint = mp
								break
							}
						}
					}
				}

				arrays = append(arrays, array)
			}
		}
	}

	// Get ZFS pools
	if CheckZFSInstalled() {
		zfsPools, err := GetZFSPools()
		if err == nil {
			for _, pool := range zfsPools {
				array := RAIDArrayInfo{
					Name:          pool.Name,
					Type:          RAIDTypeZFS,
					Level:         pool.RAIDLevel,
					Size:          pool.Size,
					Used:          pool.Allocated,
					Available:     pool.Free,
					TotalDevices:  len(pool.Devices),
					ActiveDevices: 0, // Count active devices
					State:         pool.Health,
					Health:        pool.Health,
				}

				// Count active devices
				for _, dev := range pool.Devices {
					if dev.State == "ONLINE" {
						array.ActiveDevices++
					}
				}

				// Get mount point from first dataset
				if len(pool.Datasets) > 0 && pool.Datasets[0].MountPoint != "-" {
					array.MountPoint = pool.Datasets[0].MountPoint
				}

				arrays = append(arrays, array)
			}
		}
	}

	return arrays, nil
}

// GetRAIDArray gets information about a specific RAID array
func GetRAIDArray(name string, raidType RAIDSystemType) (*RAIDArrayInfo, error) {
	arrays, err := GetAllRAIDArrays()
	if err != nil {
		return nil, err
	}

	for _, array := range arrays {
		if array.Name == name && array.Type == raidType {
			return &array, nil
		}
	}

	return nil, fmt.Errorf("RAID array not found: %s", name)
}

// GetSystemRAIDCapabilities returns what RAID systems are available on the system
func GetSystemRAIDCapabilities() map[RAIDSystemType]bool {
	capabilities := make(map[RAIDSystemType]bool)

	// Check Btrfs (always available on Linux with btrfs-progs)
	if _, err := GetBtrfsFilesystems(); err == nil {
		capabilities[RAIDTypeBtrfs] = true
	}

	// Check ZFS
	capabilities[RAIDTypeZFS] = CheckZFSInstalled()

	return capabilities
}

// ValidateRAIDConfiguration validates a RAID configuration before creation
func ValidateRAIDConfiguration(raidType RAIDSystemType, level string, devices []string) error {
	if len(devices) == 0 {
		return fmt.Errorf("no devices specified")
	}

	// Check device availability
	availableDisks, err := GetAvailableDisks()
	if err != nil {
		return fmt.Errorf("failed to get available disks: %w", err)
	}

	availableMap := make(map[string]bool)
	for _, disk := range availableDisks {
		availableMap[disk.Path] = true
	}

	for _, device := range devices {
		if !availableMap[device] {
			return fmt.Errorf("device not available or already in use: %s", device)
		}
	}

	// Validate based on RAID type
	switch raidType {
	case RAIDTypeBtrfs:
		return validateBtrfsRAIDConfig(level, devices)
	case RAIDTypeZFS:
		return validateZFSRAIDConfig(level, devices)
	default:
		return fmt.Errorf("unsupported RAID type: %s", raidType)
	}
}

// validateBtrfsRAIDConfig validates Btrfs RAID configuration
func validateBtrfsRAIDConfig(level string, devices []string) error {
	switch BtrfsRAIDLevel(level) {
	case RAIDSingle:
		if len(devices) < 1 {
			return fmt.Errorf("single requires at least 1 device")
		}
	case RAID0:
		if len(devices) < 2 {
			return fmt.Errorf("RAID0 requires at least 2 devices")
		}
	case RAID1, RAID10:
		if len(devices) < 2 {
			return fmt.Errorf("%s requires at least 2 devices", level)
		}
	case RAID1C3:
		if len(devices) < 3 {
			return fmt.Errorf("RAID1C3 requires at least 3 devices")
		}
	case RAID1C4:
		if len(devices) < 4 {
			return fmt.Errorf("RAID1C4 requires at least 4 devices")
		}
	case RAID5:
		if len(devices) < 2 {
			return fmt.Errorf("RAID5 requires at least 2 devices")
		}
	case RAID6:
		if len(devices) < 3 {
			return fmt.Errorf("RAID6 requires at least 3 devices")
		}
	default:
		return fmt.Errorf("unsupported Btrfs RAID level: %s", level)
	}
	return nil
}

// validateZFSRAIDConfig validates ZFS RAID configuration
func validateZFSRAIDConfig(level string, devices []string) error {
	switch ZFSVdevType(level) {
	case ZFSSingle:
		if len(devices) < 1 {
			return fmt.Errorf("single requires at least 1 device")
		}
	case ZFSMirror:
		if len(devices) < 2 {
			return fmt.Errorf("mirror requires at least 2 devices")
		}
	case ZFSRAIDZ:
		if len(devices) < 3 {
			return fmt.Errorf("raidz requires at least 3 devices")
		}
	case ZFSRAIDZ2:
		if len(devices) < 4 {
			return fmt.Errorf("raidz2 requires at least 4 devices")
		}
	case ZFSRAIDZ3:
		if len(devices) < 5 {
			return fmt.Errorf("raidz3 requires at least 5 devices")
		}
	default:
		return fmt.Errorf("unsupported ZFS vdev type: %s", level)
	}
	return nil
}

// GetRAIDLevelInfo returns human-readable information about a RAID level
func GetRAIDLevelInfo(raidType RAIDSystemType, level string) string {
	descriptions := map[string]map[string]string{
		string(RAIDTypeBtrfs): {
			"single":  "Single - No redundancy (1 copy)",
			"raid0":   "RAID0 - Striping (no redundancy, maximum performance)",
			"raid1":   "RAID1 - Mirroring (2 copies, tolerates 1 disk failure)",
			"raid1c3": "RAID1C3 - Triple mirroring (3 copies, tolerates 2 disk failures)",
			"raid1c4": "RAID1C4 - Quad mirroring (4 copies, tolerates 3 disk failures)",
			"raid10":  "RAID10 - Striped mirrors (2 copies, tolerates 1 disk failure per mirror pair)",
			"raid5":   "RAID5 - Striping with parity (tolerates 1 disk failure)",
			"raid6":   "RAID6 - Striping with double parity (tolerates 2 disk failures)",
		},
		string(RAIDTypeZFS): {
			"":       "Single - No redundancy",
			"mirror": "Mirror - 2+ copies (tolerates N-1 disk failures)",
			"raidz":  "RAIDZ - Single parity (like RAID5, tolerates 1 disk failure)",
			"raidz2": "RAIDZ2 - Double parity (like RAID6, tolerates 2 disk failures)",
			"raidz3": "RAIDZ3 - Triple parity (tolerates 3 disk failures)",
		},
	}

	if typeMap, ok := descriptions[string(raidType)]; ok {
		if desc, ok := typeMap[level]; ok {
			return desc
		}
	}

	return "Unknown RAID level"
}
