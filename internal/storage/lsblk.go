package storage

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
)

// lsblkDevice represents the JSON output from lsblk
type lsblkDevice struct {
	Name       string        `json:"name"`
	Path       string        `json:"path"`
	Size       string        `json:"size"`      // Size in bytes as string
	Type       string        `json:"type"`      // disk, part, rom, etc.
	Model      string        `json:"model"`
	Serial     string        `json:"serial"`
	Rota       string        `json:"rota"`      // "1" for rotational, "0" for SSD
	Tran       string        `json:"tran"`      // Transport: sata, nvme, usb, etc.
	Fstype     string        `json:"fstype"`    // Filesystem type
	Mountpoint string        `json:"mountpoint"`
	Label      string        `json:"label"`
	Children   []lsblkDevice `json:"children"`
}

type lsblkOutput struct {
	BlockDevices []lsblkDevice `json:"blockdevices"`
}

// ListDisks returns a list of all physical disks using lsblk
func ListDisks() ([]Disk, error) {
	// Run lsblk with JSON output
	cmd := exec.Command("lsblk", "-J", "-b", "-o",
		"NAME,PATH,SIZE,TYPE,MODEL,SERIAL,ROTA,TRAN,FSTYPE,MOUNTPOINT,LABEL")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var lsblk lsblkOutput
	if err := json.Unmarshal(output, &lsblk); err != nil {
		return nil, err
	}

	var disks []Disk
	for _, dev := range lsblk.BlockDevices {
		// Only include physical disks (type "disk")
		if dev.Type != "disk" {
			continue
		}

		// Skip loop devices, ram disks, etc.
		if strings.HasPrefix(dev.Name, "loop") ||
			strings.HasPrefix(dev.Name, "ram") ||
			strings.HasPrefix(dev.Name, "zram") {
			continue
		}

		disk := Disk{
			Name:       dev.Name,
			Path:       dev.Path,
			Model:      strings.TrimSpace(dev.Model),
			Serial:     strings.TrimSpace(dev.Serial),
			Rotational: dev.Rota == "1",
			Health:     HealthUnknown, // Will be updated by SMART
			Temperature: -1,
			PowerOnHours: -1,
		}

		// Parse size
		if size, err := strconv.ParseUint(dev.Size, 10, 64); err == nil {
			disk.Size = size
			disk.SizeHuman = FormatBytes(size)
		}

		// Determine disk type
		disk.Type = determineDiskType(dev)

		// Parse partitions
		for _, child := range dev.Children {
			if child.Type == "part" {
				part := Partition{
					Name:       child.Name,
					Path:       child.Path,
					Filesystem: child.Fstype,
					Mountpoint: child.Mountpoint,
					Label:      child.Label,
				}
				if size, err := strconv.ParseUint(child.Size, 10, 64); err == nil {
					part.Size = size
					part.SizeHuman = FormatBytes(size)
				}
				disk.Partitions = append(disk.Partitions, part)
			}
		}

		disks = append(disks, disk)
	}

	return disks, nil
}

// determineDiskType determines if a disk is HDD, SSD, or NVMe
func determineDiskType(dev lsblkDevice) DiskType {
	// Check transport type first
	if dev.Tran == "nvme" || strings.HasPrefix(dev.Name, "nvme") {
		return DiskTypeNVMe
	}

	// Check if rotational
	if dev.Rota == "0" {
		return DiskTypeSSD
	}

	return DiskTypeHDD
}

// GetDiskByName finds a disk by its name (e.g., "sda")
func GetDiskByName(name string) (*Disk, error) {
	disks, err := ListDisks()
	if err != nil {
		return nil, err
	}

	for _, disk := range disks {
		if disk.Name == name {
			return &disk, nil
		}
	}

	return nil, nil
}
