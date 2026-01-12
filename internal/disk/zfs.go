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

// ZFSPool represents a ZFS pool
type ZFSPool struct {
	Name       string       `json:"name"`
	Size       uint64       `json:"size"`
	Allocated  uint64       `json:"allocated"`
	Free       uint64       `json:"free"`
	Health     string       `json:"health"`
	RAIDLevel  string       `json:"raid_level"`
	Devices    []ZFSDevice  `json:"devices"`
	Datasets   []ZFSDataset `json:"datasets"`
}

// ZFSDevice represents a device in a ZFS pool
type ZFSDevice struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Read   int64  `json:"read_errors"`
	Write  int64  `json:"write_errors"`
	Cksum  int64  `json:"checksum_errors"`
}

// ZFSDataset represents a ZFS dataset
type ZFSDataset struct {
	Name      string `json:"name"`
	Used      uint64 `json:"used"`
	Available uint64 `json:"available"`
	Refer     uint64 `json:"refer"`
	MountPoint string `json:"mountpoint"`
}

// ZFSVdevType represents ZFS vdev types
type ZFSVdevType string

const (
	ZFSSingle  ZFSVdevType = ""
	ZFSMirror  ZFSVdevType = "mirror"
	ZFSRAIDZ   ZFSVdevType = "raidz"
	ZFSRAIDZ2  ZFSVdevType = "raidz2"
	ZFSRAIDZ3  ZFSVdevType = "raidz3"
)

// CheckZFSInstalled checks if ZFS utilities are installed
func CheckZFSInstalled() bool {
	cmd := exec.Command("which", "zpool")
	return cmd.Run() == nil
}

// GetZFSPools returns all ZFS pools on the system
func GetZFSPools() ([]ZFSPool, error) {
	if !CheckZFSInstalled() {
		return nil, fmt.Errorf("ZFS utilities not installed")
	}

	cmd := exec.Command("sudo", "zpool", "list", "-H", "-o", "name,size,alloc,free,health")
	output, err := cmd.Output()
	if err != nil {
		// If no pools exist, zpool list returns error
		if strings.Contains(string(output), "no pools available") {
			return []ZFSPool{}, nil
		}
		return nil, fmt.Errorf("failed to list ZFS pools: %w", err)
	}

	var pools []ZFSPool
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 5 {
			pool := ZFSPool{
				Name:   fields[0],
				Health: fields[4],
			}

			// Parse sizes
			pool.Size = parseZFSSize(fields[1])
			pool.Allocated = parseZFSSize(fields[2])
			pool.Free = parseZFSSize(fields[3])

			// Get pool status details
			if status, err := getZFSPoolStatus(pool.Name); err == nil {
				pool.Devices = status.Devices
				pool.RAIDLevel = status.RAIDLevel
			}

			// Get datasets
			if datasets, err := getZFSDatasets(pool.Name); err == nil {
				pool.Datasets = datasets
			}

			pools = append(pools, pool)
		}
	}

	return pools, nil
}

// parseZFSSize converts ZFS size string to bytes
func parseZFSSize(sizeStr string) uint64 {
	sizeStr = strings.TrimSpace(sizeStr)

	var value float64
	var unit string
	fmt.Sscanf(sizeStr, "%f%s", &value, &unit)

	multiplier := uint64(1)
	switch strings.ToUpper(unit) {
	case "K":
		multiplier = 1024
	case "M":
		multiplier = 1024 * 1024
	case "G":
		multiplier = 1024 * 1024 * 1024
	case "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "P":
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
	}

	return uint64(value * float64(multiplier))
}

// ZFSPoolStatus represents the detailed status of a ZFS pool
type ZFSPoolStatus struct {
	Name      string
	State     string
	RAIDLevel string
	Devices   []ZFSDevice
}

// getZFSPoolStatus gets the detailed status of a ZFS pool
func getZFSPoolStatus(poolName string) (*ZFSPoolStatus, error) {
	cmd := exec.Command("sudo", "zpool", "status", poolName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get pool status: %w", err)
	}

	status := &ZFSPoolStatus{
		Name:    poolName,
		Devices: []ZFSDevice{},
	}

	lines := strings.Split(string(output), "\n")
	deviceRe := regexp.MustCompile(`^\s+([\w\/\-]+)\s+(\w+)\s+(\d+)\s+(\d+)\s+(\d+)`)
	raidLevelRe := regexp.MustCompile(`^\s+(mirror|raidz\d?)-\d+\s+(\w+)`)

	inConfig := false
	for _, line := range lines {
		// Check if we're in the config section
		if strings.Contains(line, "config:") {
			inConfig = true
			continue
		}

		if !inConfig {
			if strings.Contains(line, "state:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					status.State = fields[1]
				}
			}
			continue
		}

		// Parse RAID level
		if matches := raidLevelRe.FindStringSubmatch(line); matches != nil {
			if status.RAIDLevel == "" {
				status.RAIDLevel = matches[1]
			}
		}

		// Parse device lines
		if matches := deviceRe.FindStringSubmatch(line); matches != nil {
			device := ZFSDevice{
				Name:  matches[1],
				State: matches[2],
			}
			device.Read, _ = strconv.ParseInt(matches[3], 10, 64)
			device.Write, _ = strconv.ParseInt(matches[4], 10, 64)
			device.Cksum, _ = strconv.ParseInt(matches[5], 10, 64)

			status.Devices = append(status.Devices, device)
		}
	}

	return status, nil
}

// getZFSDatasets gets all datasets in a ZFS pool
func getZFSDatasets(poolName string) ([]ZFSDataset, error) {
	cmd := exec.Command("sudo", "zfs", "list", "-H", "-r", "-o", "name,used,avail,refer,mountpoint", poolName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list datasets: %w", err)
	}

	var datasets []ZFSDataset
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 5 {
			dataset := ZFSDataset{
				Name:       fields[0],
				Used:       parseZFSSize(fields[1]),
				Available:  parseZFSSize(fields[2]),
				Refer:      parseZFSSize(fields[3]),
				MountPoint: fields[4],
			}
			datasets = append(datasets, dataset)
		}
	}

	return datasets, nil
}

// CreateZFSPool creates a new ZFS pool
func CreateZFSPool(poolName string, vdevType ZFSVdevType, devices []string, mountPoint string) error {
	if !CheckZFSInstalled() {
		return fmt.Errorf("ZFS utilities not installed")
	}

	// Validate device count based on vdev type
	minDevices := 1
	switch vdevType {
	case ZFSMirror:
		minDevices = 2
	case ZFSRAIDZ:
		minDevices = 3
	case ZFSRAIDZ2:
		minDevices = 4
	case ZFSRAIDZ3:
		minDevices = 5
	}

	if len(devices) < minDevices {
		return fmt.Errorf("%s requires at least %d devices", vdevType, minDevices)
	}

	// Build command arguments
	args := []string{"zpool", "create"}

	if mountPoint != "" {
		args = append(args, "-m", mountPoint)
	}

	args = append(args, poolName)

	if vdevType != ZFSSingle {
		args = append(args, string(vdevType))
	}

	args = append(args, devices...)

	// Create pool
	cmd := exec.Command("sudo", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create ZFS pool: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// AddZFSVdev adds a vdev to an existing ZFS pool
func AddZFSVdev(poolName string, vdevType ZFSVdevType, devices []string) error {
	args := []string{"zpool", "add", poolName}

	if vdevType != ZFSSingle {
		args = append(args, string(vdevType))
	}

	args = append(args, devices...)

	cmd := exec.Command("sudo", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add vdev: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// RemoveZFSDevice removes a device from a ZFS pool (only mirrors supported)
func RemoveZFSDevice(poolName string, devicePath string) error {
	cmd := exec.Command("sudo", "zpool", "remove", poolName, devicePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove device: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// ScrubZFSPool starts a scrub operation on a ZFS pool
func ScrubZFSPool(poolName string) error {
	cmd := exec.Command("sudo", "zpool", "scrub", poolName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start scrub: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// ExportZFSPool exports a ZFS pool (unmounts it)
func ExportZFSPool(poolName string) error {
	cmd := exec.Command("sudo", "zpool", "export", poolName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to export pool: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// ImportZFSPool imports a ZFS pool
func ImportZFSPool(poolName string) error {
	cmd := exec.Command("sudo", "zpool", "import", poolName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to import pool: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// DestroyZFSPool destroys a ZFS pool (DANGEROUS!)
func DestroyZFSPool(poolName string, force bool) error {
	args := []string{"zpool", "destroy"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, poolName)

	cmd := exec.Command("sudo", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to destroy pool: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// CreateZFSDataset creates a new ZFS dataset
func CreateZFSDataset(datasetName string, mountPoint string) error {
	args := []string{"zfs", "create"}

	if mountPoint != "" {
		args = append(args, "-o", "mountpoint="+mountPoint)
	}

	args = append(args, datasetName)

	cmd := exec.Command("sudo", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create dataset: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// DestroyZFSDataset destroys a ZFS dataset
func DestroyZFSDataset(datasetName string, recursive bool) error {
	args := []string{"zfs", "destroy"}
	if recursive {
		args = append(args, "-r")
	}
	args = append(args, datasetName)

	cmd := exec.Command("sudo", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to destroy dataset: %w\nOutput: %s", err, string(output))
	}
	return nil
}
