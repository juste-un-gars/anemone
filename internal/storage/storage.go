// Package storage provides disk and storage pool monitoring capabilities.
//
// This package supports:
// - Physical disk enumeration via lsblk
// - SMART health monitoring via smartctl
// - ZFS pool status via zpool/zfs commands
package storage

import (
	"fmt"
	"time"
)

// DiskType represents the type of storage device
type DiskType string

const (
	DiskTypeHDD  DiskType = "HDD"
	DiskTypeSSD  DiskType = "SSD"
	DiskTypeNVMe DiskType = "NVMe"
)

// HealthStatus represents the health state of a disk or pool
type HealthStatus string

const (
	HealthOK       HealthStatus = "OK"
	HealthWarning  HealthStatus = "WARNING"
	HealthCritical HealthStatus = "CRITICAL"
	HealthUnknown  HealthStatus = "UNKNOWN"
)

// Disk represents a physical disk device
type Disk struct {
	Name        string       `json:"name"`        // e.g., "sda", "nvme0n1"
	Path        string       `json:"path"`        // e.g., "/dev/sda"
	Model       string       `json:"model"`       // Disk model name
	Serial      string       `json:"serial"`      // Serial number
	Size        uint64       `json:"size"`        // Size in bytes
	SizeHuman   string       `json:"size_human"`  // Human-readable size
	Type        DiskType     `json:"type"`        // HDD, SSD, NVMe
	Rotational  bool         `json:"rotational"`  // true for HDD
	Health      HealthStatus `json:"health"`      // SMART health status
	Temperature int          `json:"temperature"` // Temperature in Celsius (-1 if unknown)
	PowerOnHours int         `json:"power_on_hours"` // Hours powered on (-1 if unknown)
	Partitions  []Partition  `json:"partitions"`  // List of partitions
	SMARTData   *SMARTInfo   `json:"smart_data"`  // Detailed SMART info (optional)
}

// Partition represents a disk partition
type Partition struct {
	Name       string `json:"name"`        // e.g., "sda1"
	Path       string `json:"path"`        // e.g., "/dev/sda1"
	Size       uint64 `json:"size"`        // Size in bytes
	SizeHuman  string `json:"size_human"`  // Human-readable size
	Filesystem string `json:"filesystem"`  // e.g., "ext4", "zfs"
	Mountpoint string `json:"mountpoint"`  // Where it's mounted (empty if not)
	Label      string `json:"label"`       // Partition label
}

// SMARTInfo contains detailed SMART data for a disk
type SMARTInfo struct {
	Available       bool         `json:"available"`        // SMART supported and enabled
	Healthy         bool         `json:"healthy"`          // Overall health assessment passed
	IsNVMe          bool         `json:"is_nvme"`          // True if this is an NVMe drive
	Temperature     int          `json:"temperature"`      // Current temperature in Celsius
	PowerOnHours    int          `json:"power_on_hours"`   // Total hours powered on
	PowerCycleCount int          `json:"power_cycle_count"`// Number of power cycles

	// SATA/SSD specific
	ReallocatedSectors   int     `json:"reallocated_sectors"`   // Bad sectors reallocated
	PendingSectors       int     `json:"pending_sectors"`       // Sectors pending reallocation
	UncorrectableSectors int     `json:"uncorrectable_sectors"` // Uncorrectable sector count

	// NVMe specific
	MediaErrors          int     `json:"media_errors"`           // NVMe media and data integrity errors
	UnsafeShutdowns      int     `json:"unsafe_shutdowns"`       // Number of unsafe shutdowns
	AvailableSpare       int     `json:"available_spare"`        // Available spare capacity (%)
	AvailableSpareThresh int     `json:"available_spare_thresh"` // Available spare threshold (%)
	PercentageUsed       int     `json:"percentage_used"`        // SSD life used (%)
	DataUnitsRead        int64   `json:"data_units_read"`        // Data units read (in 512KB units)
	DataUnitsWritten     int64   `json:"data_units_written"`     // Data units written (in 512KB units)

	Attributes      []SMARTAttribute `json:"attributes"`   // Raw SMART attributes
	LastChecked     time.Time    `json:"last_checked"`     // When SMART was last read
}

// SMARTAttribute represents a single SMART attribute
type SMARTAttribute struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Value      int    `json:"value"`
	Worst      int    `json:"worst"`
	Threshold  int    `json:"threshold"`
	RawValue   string `json:"raw_value"`
	Status     string `json:"status"` // "ok", "warning", "failing"
}

// ZFSPool represents a ZFS storage pool
type ZFSPool struct {
	Name        string       `json:"name"`
	State       string       `json:"state"`       // ONLINE, DEGRADED, FAULTED, etc.
	Health      HealthStatus `json:"health"`      // Mapped health status
	Size        uint64       `json:"size"`        // Total size in bytes
	SizeHuman   string       `json:"size_human"`
	Allocated   uint64       `json:"allocated"`   // Used space in bytes
	AllocHuman  string       `json:"alloc_human"`
	Free        uint64       `json:"free"`        // Free space in bytes
	FreeHuman   string       `json:"free_human"`
	UsedPercent float64      `json:"used_percent"`// Percentage used
	Fragmentation int        `json:"fragmentation"` // Fragmentation percentage
	Capacity    int          `json:"capacity"`    // Capacity percentage
	Dedup       float64      `json:"dedup"`       // Deduplication ratio
	VDevs       []ZFSVDev    `json:"vdevs"`       // Virtual devices in pool
	Datasets    []ZFSDataset `json:"datasets"`    // Datasets/filesystems
	ScanStatus  string       `json:"scan_status"` // Last scrub/resilver status
	Errors      string       `json:"errors"`      // Error summary
}

// ZFSVDev represents a virtual device in a ZFS pool
type ZFSVDev struct {
	Name   string       `json:"name"`   // e.g., "mirror-0", "raidz1-0"
	Type   string       `json:"type"`   // mirror, raidz1, raidz2, etc.
	State  string       `json:"state"`  // ONLINE, DEGRADED, etc.
	Health HealthStatus `json:"health"`
	Disks  []ZFSDisk    `json:"disks"`  // Physical disks in vdev
}

// ZFSDisk represents a disk within a ZFS vdev
type ZFSDisk struct {
	Name   string       `json:"name"`   // e.g., "sda", "nvme0n1"
	Path   string       `json:"path"`   // Full path
	State  string       `json:"state"`  // ONLINE, OFFLINE, DEGRADED, etc.
	Health HealthStatus `json:"health"`
	Read   uint64       `json:"read"`   // Read errors
	Write  uint64       `json:"write"`  // Write errors
	Cksum  uint64       `json:"cksum"`  // Checksum errors
}

// ZFSDataset represents a ZFS dataset (filesystem or volume)
type ZFSDataset struct {
	Name        string `json:"name"`
	Type        string `json:"type"`        // filesystem, volume, snapshot
	Used        uint64 `json:"used"`        // Space used
	UsedHuman   string `json:"used_human"`
	Available   uint64 `json:"available"`   // Space available
	AvailHuman  string `json:"avail_human"`
	Refer       uint64 `json:"refer"`       // Referenced data
	Mountpoint  string `json:"mountpoint"`  // Mount point (for filesystems)
	Compression string `json:"compression"` // Compression type
	CompRatio   float64 `json:"comp_ratio"` // Compression ratio
}

// StorageOverview provides a summary of all storage
type StorageOverview struct {
	TotalDisks      int          `json:"total_disks"`
	HealthyDisks    int          `json:"healthy_disks"`
	WarningDisks    int          `json:"warning_disks"`
	CriticalDisks   int          `json:"critical_disks"`
	TotalPools      int          `json:"total_pools"`
	HealthyPools    int          `json:"healthy_pools"`
	DegradedPools   int          `json:"degraded_pools"`
	TotalCapacity   uint64       `json:"total_capacity"`
	TotalCapHuman   string       `json:"total_cap_human"`
	UsedCapacity    uint64       `json:"used_capacity"`
	UsedCapHuman    string       `json:"used_cap_human"`
	FreeCapacity    uint64       `json:"free_capacity"`
	FreeCapHuman    string       `json:"free_cap_human"`
	UsedPercent     float64      `json:"used_percent"`
	Disks           []Disk       `json:"disks"`
	Pools           []ZFSPool    `json:"pools"`
	ZFSAvailable    bool         `json:"zfs_available"`
	SMARTAvailable  bool         `json:"smart_available"`
	LastUpdated     time.Time    `json:"last_updated"`
}

// FormatBytes converts bytes to human-readable format
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

// GetStorageOverview returns a complete overview of all storage
func GetStorageOverview() (*StorageOverview, error) {
	overview := &StorageOverview{
		SMARTAvailable: IsSmartAvailable(),
		ZFSAvailable:   IsZFSAvailable(),
		LastUpdated:    time.Now(),
	}

	// Get disks with SMART data
	disks, err := GetDisksWithSMART()
	if err != nil {
		disks = []Disk{} // Continue even if disk enumeration fails
	}
	overview.Disks = disks
	overview.TotalDisks = len(disks)

	// Count disk health status
	for _, disk := range disks {
		switch disk.Health {
		case HealthOK:
			overview.HealthyDisks++
		case HealthWarning:
			overview.WarningDisks++
		case HealthCritical:
			overview.CriticalDisks++
		}
	}

	// Get ZFS pools
	pools, err := ListZFSPools()
	if err != nil {
		pools = []ZFSPool{}
	}
	overview.Pools = pools
	overview.TotalPools = len(pools)

	// Calculate totals from ZFS pools
	for _, pool := range pools {
		overview.TotalCapacity += pool.Size
		overview.UsedCapacity += pool.Allocated
		overview.FreeCapacity += pool.Free

		switch pool.Health {
		case HealthOK:
			overview.HealthyPools++
		case HealthWarning, HealthCritical:
			overview.DegradedPools++
		}
	}

	// Format capacity strings
	overview.TotalCapHuman = FormatBytes(overview.TotalCapacity)
	overview.UsedCapHuman = FormatBytes(overview.UsedCapacity)
	overview.FreeCapHuman = FormatBytes(overview.FreeCapacity)

	// Calculate overall used percentage
	if overview.TotalCapacity > 0 {
		overview.UsedPercent = float64(overview.UsedCapacity) / float64(overview.TotalCapacity) * 100
	}

	return overview, nil
}
