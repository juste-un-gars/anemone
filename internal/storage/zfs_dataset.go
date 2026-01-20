// Package storage provides ZFS dataset management operations.
package storage

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// DatasetCreateOptions contains options for creating a ZFS dataset
type DatasetCreateOptions struct {
	Name        string `json:"name"`        // Full dataset name (pool/dataset)
	Mountpoint  string `json:"mountpoint"`  // Custom mountpoint (optional)
	Compression string `json:"compression"` // off, lz4, zstd, gzip
	Quota       string `json:"quota"`       // Quota size (e.g., "10G", "500M")
	RecordSize  string `json:"recordsize"`  // Record size (e.g., "128K")
	Atime       string `json:"atime"`       // on, off
	Sync        string `json:"sync"`        // standard, always, disabled
	Owner       string `json:"owner"`       // Owner user:group for mountpoint (optional, e.g., "anemone:anemone")
}

// DatasetInfo contains detailed dataset information
type DatasetInfo struct {
	Name              string  `json:"name"`
	Type              string  `json:"type"` // filesystem, volume, snapshot
	Used              uint64  `json:"used"`
	UsedHuman         string  `json:"used_human"`
	Available         uint64  `json:"available"`
	AvailHuman        string  `json:"avail_human"`
	Referenced        uint64  `json:"referenced"`
	RefHuman          string  `json:"ref_human"`
	Mountpoint        string  `json:"mountpoint"`
	Compression       string  `json:"compression"`
	CompressionRatio  float64 `json:"compression_ratio"`
	Quota             string  `json:"quota"`
	RecordSize        string  `json:"recordsize"`
	Atime             string  `json:"atime"`
	Sync              string  `json:"sync"`
	SnapshotCount     int     `json:"snapshot_count"`
	CreationTime      string  `json:"creation_time"`
}

// ValidateDatasetName checks if a dataset name is valid
func ValidateDatasetName(name string) error {
	if name == "" {
		return fmt.Errorf("dataset name cannot be empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("dataset name too long")
	}
	// Must contain at least one slash (pool/dataset)
	if !strings.Contains(name, "/") {
		return fmt.Errorf("dataset name must be in format pool/dataset")
	}
	// Validate characters
	validName := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_\-.:/@]*$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid dataset name: must start with a letter and contain only valid characters")
	}
	return nil
}

// ValidateDatasetNameOrPool checks if a name is a valid dataset or pool name
func ValidateDatasetNameOrPool(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("name too long")
	}
	validName := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_\-.:/@]*$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid name format")
	}
	return nil
}

// CreateDataset creates a new ZFS dataset
func CreateDataset(opts DatasetCreateOptions) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateDatasetName(opts.Name); err != nil {
		return err
	}

	args := []string{"zfs", "create"}

	// Add options
	if opts.Mountpoint != "" {
		args = append(args, "-o", fmt.Sprintf("mountpoint=%s", opts.Mountpoint))
	}
	if opts.Compression != "" {
		args = append(args, "-o", fmt.Sprintf("compression=%s", opts.Compression))
	}
	if opts.Quota != "" {
		args = append(args, "-o", fmt.Sprintf("quota=%s", opts.Quota))
	}
	if opts.RecordSize != "" {
		args = append(args, "-o", fmt.Sprintf("recordsize=%s", opts.RecordSize))
	}
	if opts.Atime != "" {
		args = append(args, "-o", fmt.Sprintf("atime=%s", opts.Atime))
	}
	if opts.Sync != "" {
		args = append(args, "-o", fmt.Sprintf("sync=%s", opts.Sync))
	}

	args = append(args, opts.Name)

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create dataset: %s - %w", strings.TrimSpace(string(output)), err)
	}

	// Fix mountpoint permissions if owner is specified
	if opts.Owner != "" {
		// Get the actual mountpoint
		mountpoint := opts.Mountpoint
		if mountpoint == "" {
			// Get mountpoint from ZFS (it inherits from parent or uses default)
			mp, err := GetDatasetProperty(opts.Name, "mountpoint")
			if err != nil {
				return fmt.Errorf("dataset created but failed to get mountpoint: %w", err)
			}
			mountpoint = mp
		}
		if mountpoint != "" && mountpoint != "none" && mountpoint != "-" {
			if err := FixMountpointOwnership(mountpoint, opts.Owner); err != nil {
				return fmt.Errorf("dataset created but failed to set ownership: %w", err)
			}
		}
	}

	return nil
}

// DeleteDataset destroys a ZFS dataset
func DeleteDataset(name string, recursive bool, force bool) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateDatasetNameOrPool(name); err != nil {
		return err
	}

	// Prevent deleting root datasets (pools) - they must be destroyed with zpool destroy
	if !strings.Contains(name, "/") {
		return fmt.Errorf("cannot delete root dataset '%s': use 'Destroy Pool' to remove the entire pool", name)
	}

	args := []string{"zfs", "destroy"}
	if recursive {
		args = append(args, "-r")
	}
	if force {
		args = append(args, "-f")
	}
	args = append(args, name)

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete dataset: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// SetDatasetProperty sets a property on a ZFS dataset
func SetDatasetProperty(name, property, value string) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateDatasetNameOrPool(name); err != nil {
		return err
	}

	// Validate property name
	validProp := regexp.MustCompile(`^[a-z_:]+$`)
	if !validProp.MatchString(property) {
		return fmt.Errorf("invalid property name")
	}

	// Validate common property values
	switch property {
	case "compression":
		valid := []string{"off", "on", "lz4", "zstd", "zstd-fast", "gzip", "gzip-1", "gzip-9", "lzjb"}
		if !contains(valid, value) {
			return fmt.Errorf("invalid compression value: %s", value)
		}
	case "atime":
		if value != "on" && value != "off" {
			return fmt.Errorf("atime must be 'on' or 'off'")
		}
	case "sync":
		valid := []string{"standard", "always", "disabled"}
		if !contains(valid, value) {
			return fmt.Errorf("invalid sync value: %s", value)
		}
	}

	cmd := exec.Command("sudo", "zfs", "set", fmt.Sprintf("%s=%s", property, value), name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set property: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// GetDatasetProperty gets a property value from a ZFS dataset
func GetDatasetProperty(name, property string) (string, error) {
	if !IsZFSAvailable() {
		return "", fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateDatasetNameOrPool(name); err != nil {
		return "", err
	}

	cmd := exec.Command("sudo", "zfs", "get", "-H", "-o", "value", property, name)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get property: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetDatasetInfo returns detailed information about a dataset
func GetDatasetInfo(name string) (*DatasetInfo, error) {
	if !IsZFSAvailable() {
		return nil, fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateDatasetNameOrPool(name); err != nil {
		return nil, err
	}

	// Get multiple properties at once
	cmd := exec.Command("sudo", "zfs", "get", "-H", "-p",
		"type,used,available,referenced,mountpoint,compression,compressratio,quota,recordsize,atime,sync,creation",
		name)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get dataset info: %w", err)
	}

	info := &DatasetInfo{Name: name}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) < 3 {
			continue
		}
		property := fields[1]
		value := fields[2]

		switch property {
		case "type":
			info.Type = value
		case "used":
			if v, err := strconv.ParseUint(value, 10, 64); err == nil {
				info.Used = v
				info.UsedHuman = FormatBytes(v)
			}
		case "available":
			if v, err := strconv.ParseUint(value, 10, 64); err == nil {
				info.Available = v
				info.AvailHuman = FormatBytes(v)
			}
		case "referenced":
			if v, err := strconv.ParseUint(value, 10, 64); err == nil {
				info.Referenced = v
				info.RefHuman = FormatBytes(v)
			}
		case "mountpoint":
			info.Mountpoint = value
		case "compression":
			info.Compression = value
		case "compressratio":
			if v, err := strconv.ParseFloat(strings.TrimSuffix(value, "x"), 64); err == nil {
				info.CompressionRatio = v
			}
		case "quota":
			if value == "0" || value == "none" {
				info.Quota = "none"
			} else {
				info.Quota = value
			}
		case "recordsize":
			info.RecordSize = value
		case "atime":
			info.Atime = value
		case "sync":
			info.Sync = value
		case "creation":
			info.CreationTime = value
		}
	}

	// Count snapshots
	snapshots, err := ListSnapshots(name)
	if err == nil {
		info.SnapshotCount = len(snapshots)
	}

	return info, nil
}

// ListDatasets lists all datasets in a pool or under a dataset
func ListDatasets(parent string) ([]DatasetInfo, error) {
	if !IsZFSAvailable() {
		return nil, fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateDatasetNameOrPool(parent); err != nil {
		return nil, err
	}

	cmd := exec.Command("sudo", "zfs", "list", "-H", "-p", "-r", "-t", "filesystem,volume",
		"-o", "name,type,used,available,referenced,mountpoint,compression,compressratio",
		parent)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list datasets: %w", err)
	}

	var datasets []DatasetInfo

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 8 {
			continue
		}

		info := DatasetInfo{
			Name:        fields[0],
			Type:        fields[1],
			Mountpoint:  fields[5],
			Compression: fields[6],
		}

		if used, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
			info.Used = used
			info.UsedHuman = FormatBytes(used)
		}
		if avail, err := strconv.ParseUint(fields[3], 10, 64); err == nil {
			info.Available = avail
			info.AvailHuman = FormatBytes(avail)
		}
		if ref, err := strconv.ParseUint(fields[4], 10, 64); err == nil {
			info.Referenced = ref
			info.RefHuman = FormatBytes(ref)
		}
		if ratio, err := strconv.ParseFloat(strings.TrimSuffix(fields[7], "x"), 64); err == nil {
			info.CompressionRatio = ratio
		}

		datasets = append(datasets, info)
	}

	return datasets, nil
}

// RenameDataset renames a dataset
func RenameDataset(oldName, newName string) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateDatasetNameOrPool(oldName); err != nil {
		return fmt.Errorf("invalid old name: %w", err)
	}
	if err := ValidateDatasetNameOrPool(newName); err != nil {
		return fmt.Errorf("invalid new name: %w", err)
	}

	cmd := exec.Command("sudo", "zfs", "rename", oldName, newName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to rename dataset: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// MountDataset mounts a ZFS dataset
func MountDataset(name string) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateDatasetNameOrPool(name); err != nil {
		return err
	}

	cmd := exec.Command("sudo", "zfs", "mount", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to mount dataset: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// UnmountDataset unmounts a ZFS dataset
func UnmountDataset(name string, force bool) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateDatasetNameOrPool(name); err != nil {
		return err
	}

	args := []string{"zfs", "unmount"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, name)

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unmount dataset: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// InheritDatasetProperty resets a property to its inherited value
func InheritDatasetProperty(name, property string) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateDatasetNameOrPool(name); err != nil {
		return err
	}

	validProp := regexp.MustCompile(`^[a-z_:]+$`)
	if !validProp.MatchString(property) {
		return fmt.Errorf("invalid property name")
	}

	cmd := exec.Command("sudo", "zfs", "inherit", property, name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to inherit property: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
