package storage

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
)

// flexInt handles JSON numbers that may be int or string
type flexInt uint64

func (f *flexInt) UnmarshalJSON(data []byte) error {
	// Try as number first
	var n uint64
	if err := json.Unmarshal(data, &n); err == nil {
		*f = flexInt(n)
		return nil
	}
	// Try as string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		*f = flexInt(n)
		return nil
	}
	*f = 0
	return nil
}

// flexBool handles JSON booleans that may be bool or string ("0"/"1")
type flexBool bool

func (f *flexBool) UnmarshalJSON(data []byte) error {
	// Try as boolean first
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		*f = flexBool(b)
		return nil
	}
	// Try as string "0" or "1"
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = flexBool(s == "1" || s == "true")
		return nil
	}
	*f = false
	return nil
}

// lsblkDevice represents the JSON output from lsblk
type lsblkDevice struct {
	Name       string        `json:"name"`
	Path       string        `json:"path"`
	Size       flexInt       `json:"size"`       // Size in bytes (int or string)
	Type       string        `json:"type"`       // disk, part, rom, etc.
	Model      string        `json:"model"`
	Serial     string        `json:"serial"`
	Rota       flexBool      `json:"rota"`       // true for rotational (HDD)
	Tran       string        `json:"tran"`       // Transport: sata, nvme, usb, etc.
	Fstype     string        `json:"fstype"`     // Filesystem type
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
			Name:         dev.Name,
			Path:         dev.Path,
			Model:        strings.TrimSpace(dev.Model),
			Serial:       strings.TrimSpace(dev.Serial),
			Size:         uint64(dev.Size),
			SizeHuman:    FormatBytes(uint64(dev.Size)),
			Rotational:   bool(dev.Rota),
			Health:       HealthUnknown, // Will be updated by SMART
			Temperature:  -1,
			PowerOnHours: -1,
			Filesystem:   dev.Fstype,     // Filesystem if formatted without partitions
			Mountpoint:   dev.Mountpoint, // Mount point if mounted directly
		}

		// Determine disk type
		disk.Type = determineDiskType(dev)

		// Parse partitions
		for _, child := range dev.Children {
			if child.Type == "part" {
				part := Partition{
					Name:       child.Name,
					Path:       child.Path,
					Size:       uint64(child.Size),
					SizeHuman:  FormatBytes(uint64(child.Size)),
					Filesystem: child.Fstype,
					Mountpoint: child.Mountpoint,
					Label:      child.Label,
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

	// Check if rotational (true = HDD, false = SSD)
	if !bool(dev.Rota) {
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
