// Package storage provides disk formatting and wiping operations.
package storage

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// FormatOptions contains options for formatting a disk
type FormatOptions struct {
	Device     string `json:"device"`      // Device path (e.g., /dev/sdb)
	Filesystem string `json:"filesystem"`  // ext4, xfs, fat32, exfat
	Label      string `json:"label"`       // Volume label (optional)
	Force      bool   `json:"force"`       // Force format even if device has data
	Mount      bool   `json:"mount"`       // Mount after formatting
	MountPath  string `json:"mount_path"`  // Mount point (e.g., /mnt/sda)
}

// WipeOptions contains options for wiping a disk
type WipeOptions struct {
	Device string `json:"device"` // Device path
	Quick  bool   `json:"quick"`  // Quick wipe (just first MB) vs full wipe
}

// AvailableDisk represents a disk that can be used for operations
type AvailableDisk struct {
	Name      string `json:"name"`       // e.g., sdb
	Path      string `json:"path"`       // e.g., /dev/sdb
	Model     string `json:"model"`      // Disk model
	Size      uint64 `json:"size"`       // Size in bytes
	SizeHuman string `json:"size_human"` // Human-readable size
	InUse     bool   `json:"in_use"`     // Whether disk is in use
	InUseBy   string `json:"in_use_by"`  // What is using it (zfs, mount, etc.)
}

// ValidateDevicePath validates a block device path
func ValidateDevicePath(path string) error {
	if path == "" {
		return fmt.Errorf("device path cannot be empty")
	}
	if !strings.HasPrefix(path, "/dev/") {
		return fmt.Errorf("device path must start with /dev/")
	}
	// Prevent command injection - only allow specific patterns
	validPath := regexp.MustCompile(`^/dev/(sd[a-z]+|nvme\d+n\d+|vd[a-z]+|loop\d+)$`)
	if !validPath.MatchString(path) {
		return fmt.Errorf("invalid device path format: must be a whole disk (e.g., /dev/sdb, /dev/nvme0n1)")
	}
	return nil
}

// IsDiskInUse checks if a disk is currently in use
func IsDiskInUse(device string) (bool, string, error) {
	if err := ValidateDevicePath(device); err != nil {
		return false, "", err
	}

	// Extract device name from path
	name := strings.TrimPrefix(device, "/dev/")

	// Check if device is part of a ZFS pool
	if IsZFSAvailable() {
		cmd := exec.Command("sudo", "zpool", "status")
		output, err := cmd.Output()
		if err == nil && strings.Contains(string(output), name) {
			return true, "ZFS pool", nil
		}
	}

	// Check if device or partitions are mounted
	cmd := exec.Command("mount")
	output, err := cmd.Output()
	if err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, device) || strings.Contains(line, name) {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					return true, fmt.Sprintf("mounted at %s", fields[2]), nil
				}
				return true, "mounted", nil
			}
		}
	}

	// Check if device has partitions that are in use
	cmd = exec.Command("lsblk", "-n", "-o", "NAME,MOUNTPOINT", device)
	output, err = cmd.Output()
	if err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 2 && fields[1] != "" {
				return true, fmt.Sprintf("partition mounted at %s", fields[1]), nil
			}
		}
	}

	// Check if device is part of MD RAID
	cmd = exec.Command("cat", "/proc/mdstat")
	output, err = cmd.Output()
	if err == nil && strings.Contains(string(output), name) {
		return true, "MD RAID array", nil
	}

	return false, "", nil
}

// GetAvailableDisks returns disks that can be used for operations
func GetAvailableDisks() ([]AvailableDisk, error) {
	// Get all block devices
	cmd := exec.Command("lsblk", "-d", "-n", "-b", "-o", "NAME,SIZE,MODEL,TYPE")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list disks: %w", err)
	}

	var disks []AvailableDisk

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}

		name := fields[0]
		deviceType := fields[len(fields)-1]

		// Only include whole disks
		if deviceType != "disk" {
			continue
		}

		// Skip loop devices unless they look like test disks
		if strings.HasPrefix(name, "loop") {
			// Check if it's a significant size (> 10MB) - likely a test disk
			if len(fields) >= 2 {
				if size, err := parseNumeric(fields[1]); err == nil && size < 10*1024*1024 {
					continue
				}
			}
		}

		disk := AvailableDisk{
			Name: name,
			Path: "/dev/" + name,
		}

		// Parse size
		if len(fields) >= 2 {
			if size, err := parseNumeric(fields[1]); err == nil {
				disk.Size = uint64(size)
				disk.SizeHuman = FormatBytes(uint64(size))
			}
		}

		// Parse model (everything between size and type)
		if len(fields) > 4 {
			disk.Model = strings.Join(fields[2:len(fields)-1], " ")
		} else if len(fields) >= 3 && fields[2] != deviceType {
			disk.Model = fields[2]
		}

		// Check if in use
		inUse, usedBy, _ := IsDiskInUse(disk.Path)
		disk.InUse = inUse
		disk.InUseBy = usedBy

		disks = append(disks, disk)
	}

	return disks, nil
}

// parseNumeric parses a numeric string
func parseNumeric(s string) (int64, error) {
	s = strings.TrimSpace(s)
	var val int64
	_, err := fmt.Sscanf(s, "%d", &val)
	return val, err
}

// FormatDisk formats a disk with the specified filesystem
func FormatDisk(opts FormatOptions) error {
	if err := ValidateDevicePath(opts.Device); err != nil {
		return err
	}

	// Validate filesystem
	switch opts.Filesystem {
	case "ext4", "xfs", "vfat", "fat32", "exfat":
		// OK - vfat and fat32 are aliases
	default:
		return fmt.Errorf("unsupported filesystem: %s (use ext4, xfs, fat32, or exfat)", opts.Filesystem)
	}

	// Check if disk is in use
	if !opts.Force {
		inUse, usedBy, err := IsDiskInUse(opts.Device)
		if err != nil {
			return fmt.Errorf("failed to check if disk is in use: %w", err)
		}
		if inUse {
			return fmt.Errorf("disk is in use by %s - use force option to override", usedBy)
		}
	}

	// Wipe partition table first
	cmd := exec.Command("sudo", "wipefs", "-a", opts.Device)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to wipe partition table: %s - %w", strings.TrimSpace(string(output)), err)
	}

	// Build format command
	var args []string
	switch opts.Filesystem {
	case "ext4":
		args = []string{"mkfs.ext4"}
		if opts.Force {
			args = append(args, "-F")
		}
		if opts.Label != "" && len(opts.Label) <= 16 {
			args = append(args, "-L", opts.Label)
		}
		args = append(args, opts.Device)
	case "xfs":
		args = []string{"mkfs.xfs"}
		if opts.Force {
			args = append(args, "-f")
		}
		if opts.Label != "" && len(opts.Label) <= 12 {
			args = append(args, "-L", opts.Label)
		}
		args = append(args, opts.Device)
	case "vfat", "fat32":
		args = []string{"mkfs.vfat", "-F", "32"}
		if opts.Label != "" && len(opts.Label) <= 11 {
			args = append(args, "-n", strings.ToUpper(opts.Label))
		}
		args = append(args, opts.Device)
	case "exfat":
		args = []string{"mkfs.exfat"}
		if opts.Label != "" && len(opts.Label) <= 15 {
			args = append(args, "-L", opts.Label)
		}
		args = append(args, opts.Device)
	}

	cmd = exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to format disk: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// WipeDisk wipes a disk
func WipeDisk(opts WipeOptions) error {
	if err := ValidateDevicePath(opts.Device); err != nil {
		return err
	}

	// Check if disk is in use
	inUse, usedBy, err := IsDiskInUse(opts.Device)
	if err != nil {
		return fmt.Errorf("failed to check if disk is in use: %w", err)
	}
	if inUse {
		return fmt.Errorf("disk is in use by %s - cannot wipe", usedBy)
	}

	if opts.Quick {
		// Quick wipe - just the first MB (destroys partition table and filesystem headers)
		cmd := exec.Command("sudo", "dd", "if=/dev/zero", "of="+opts.Device, "bs=1M", "count=1", "conv=fsync")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to wipe disk: %s - %w", strings.TrimSpace(string(output)), err)
		}

		// Also wipe the last MB (GPT backup)
		// Get disk size first
		sizeCmd := exec.Command("blockdev", "--getsize64", opts.Device)
		sizeOutput, err := sizeCmd.Output()
		if err == nil {
			if size, err := parseNumeric(string(sizeOutput)); err == nil && size > 1024*1024 {
				skipMB := (size / (1024 * 1024)) - 1
				cmd = exec.Command("sudo", "dd", "if=/dev/zero", "of="+opts.Device,
					"bs=1M", "count=1", fmt.Sprintf("seek=%d", skipMB), "conv=fsync")
				cmd.CombinedOutput() // Best effort
			}
		}
	} else {
		// Full wipe using wipefs
		cmd := exec.Command("sudo", "wipefs", "-a", opts.Device)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to wipe disk: %s - %w", strings.TrimSpace(string(output)), err)
		}

		// Zero out partition table area
		cmd = exec.Command("sudo", "dd", "if=/dev/zero", "of="+opts.Device, "bs=1M", "count=1", "conv=fsync")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to zero disk start: %s - %w", strings.TrimSpace(string(output)), err)
		}
	}

	return nil
}

// CreatePartition creates a partition table and single partition on a disk
type CreatePartitionOptions struct {
	Device     string `json:"device"`      // Device path
	TableType  string `json:"table_type"`  // gpt or msdos
	Filesystem string `json:"filesystem"`  // ext4, xfs (optional - format partition)
	Label      string `json:"label"`       // Partition label (optional)
}

// CreatePartition creates a partition on a disk
func CreatePartition(opts CreatePartitionOptions) error {
	if err := ValidateDevicePath(opts.Device); err != nil {
		return err
	}

	// Validate table type
	if opts.TableType != "gpt" && opts.TableType != "msdos" {
		opts.TableType = "gpt" // Default to GPT
	}

	// Check if disk is in use
	inUse, usedBy, err := IsDiskInUse(opts.Device)
	if err != nil {
		return fmt.Errorf("failed to check if disk is in use: %w", err)
	}
	if inUse {
		return fmt.Errorf("disk is in use by %s", usedBy)
	}

	// Create partition table
	cmd := exec.Command("sudo", "parted", "-s", opts.Device, "mklabel", opts.TableType)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create partition table: %s - %w", strings.TrimSpace(string(output)), err)
	}

	// Create single partition using all space
	cmd = exec.Command("sudo", "parted", "-s", "-a", "optimal", opts.Device, "mkpart", "primary", "0%", "100%")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create partition: %s - %w", strings.TrimSpace(string(output)), err)
	}

	// Determine partition path
	partPath := opts.Device + "1"
	if strings.Contains(opts.Device, "nvme") || strings.Contains(opts.Device, "loop") {
		partPath = opts.Device + "p1"
	}

	// Wait for partition to appear
	cmd = exec.Command("sudo", "partprobe", opts.Device)
	cmd.Run()

	// Format partition if filesystem specified
	if opts.Filesystem != "" {
		var args []string
		switch opts.Filesystem {
		case "ext4":
			args = []string{"mkfs.ext4", "-F"}
			if opts.Label != "" && len(opts.Label) <= 16 {
				args = append(args, "-L", opts.Label)
			}
			args = append(args, partPath)
		case "xfs":
			args = []string{"mkfs.xfs", "-f"}
			if opts.Label != "" && len(opts.Label) <= 12 {
				args = append(args, "-L", opts.Label)
			}
			args = append(args, partPath)
		case "vfat", "fat32":
			args = []string{"mkfs.vfat", "-F", "32"}
			if opts.Label != "" && len(opts.Label) <= 11 {
				args = append(args, "-n", strings.ToUpper(opts.Label))
			}
			args = append(args, partPath)
		case "exfat":
			args = []string{"mkfs.exfat"}
			if opts.Label != "" && len(opts.Label) <= 15 {
				args = append(args, "-L", opts.Label)
			}
			args = append(args, partPath)
		default:
			return fmt.Errorf("unsupported filesystem: %s (use ext4, xfs, fat32, or exfat)", opts.Filesystem)
		}

		cmd = exec.Command("sudo", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to format partition: %s - %w", strings.TrimSpace(string(output)), err)
		}
	}

	return nil
}

// GetDiskInfo returns detailed information about a disk
func GetDiskInfo(device string) (*Disk, error) {
	if err := ValidateDevicePath(device); err != nil {
		return nil, err
	}

	name := strings.TrimPrefix(device, "/dev/")

	// Get disk info from lsblk
	cmd := exec.Command("lsblk", "-d", "-b", "-o", "NAME,SIZE,MODEL,ROTA,TYPE", device)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk info: %w", err)
	}

	disk := &Disk{
		Name:        name,
		Path:        device,
		Temperature: -1,
		PowerOnHours: -1,
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	// Skip header
	scanner.Scan()
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			if size, err := parseNumeric(fields[1]); err == nil {
				disk.Size = uint64(size)
				disk.SizeHuman = FormatBytes(uint64(size))
			}
		}
		if len(fields) >= 3 {
			disk.Model = fields[2]
		}
		if len(fields) >= 4 {
			disk.Rotational = fields[3] == "1"
			if disk.Rotational {
				disk.Type = DiskTypeHDD
			} else if strings.HasPrefix(name, "nvme") {
				disk.Type = DiskTypeNVMe
			} else {
				disk.Type = DiskTypeSSD
			}
		}
	}

	// Get partitions
	cmd = exec.Command("lsblk", "-b", "-o", "NAME,SIZE,FSTYPE,MOUNTPOINT,LABEL", device)
	output, err = cmd.Output()
	if err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		scanner.Scan() // Skip header
		scanner.Scan() // Skip disk line
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 1 {
				partName := strings.TrimPrefix(fields[0], "├─")
				partName = strings.TrimPrefix(partName, "└─")
				partName = strings.TrimPrefix(partName, "|-")
				partName = strings.TrimPrefix(partName, "`-")
				partName = strings.TrimSpace(partName)

				part := Partition{
					Name: partName,
					Path: "/dev/" + partName,
				}
				if len(fields) >= 2 {
					if size, err := parseNumeric(fields[1]); err == nil {
						part.Size = uint64(size)
						part.SizeHuman = FormatBytes(uint64(size))
					}
				}
				if len(fields) >= 3 {
					part.Filesystem = fields[2]
				}
				if len(fields) >= 4 {
					part.Mountpoint = fields[3]
				}
				if len(fields) >= 5 {
					part.Label = fields[4]
				}
				disk.Partitions = append(disk.Partitions, part)
			}
		}
	}

	return disk, nil
}

// UnmountDisk unmounts a disk and optionally ejects it
func UnmountDisk(mountPath string, eject bool) error {
	// Validate mount path
	if mountPath == "" {
		return fmt.Errorf("mount path cannot be empty")
	}
	if !strings.HasPrefix(mountPath, "/") {
		return fmt.Errorf("mount path must be absolute")
	}
	// Prevent path traversal
	if strings.Contains(mountPath, "..") {
		return fmt.Errorf("invalid mount path")
	}

	// Find the device for this mount point
	var device string
	if eject {
		cmd := exec.Command("findmnt", "-n", "-o", "SOURCE", mountPath)
		output, err := cmd.Output()
		if err == nil {
			device = strings.TrimSpace(string(output))
			// Remove partition number to get base device (e.g., /dev/sda1 -> /dev/sda)
			if matched, _ := regexp.MatchString(`^/dev/[a-z]+[0-9]+$`, device); matched {
				device = regexp.MustCompile(`[0-9]+$`).ReplaceAllString(device, "")
			}
		}
	}

	// Unmount the disk
	cmd := exec.Command("sudo", "umount", mountPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to unmount disk: %s - %w", strings.TrimSpace(string(output)), err)
	}

	// Remove the mount point directory if it's empty
	// Only attempt to remove if it's in /mnt or /media (safe locations)
	// Use sudo rmdir because the directory was created with sudo mkdir
	if strings.HasPrefix(mountPath, "/mnt/") || strings.HasPrefix(mountPath, "/media/") {
		cmd = exec.Command("sudo", "rmdir", mountPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			// Not critical if removal fails (directory might not be empty)
			log.Printf("Note: Could not remove mount point directory %s: %v - %s", mountPath, err, strings.TrimSpace(string(output)))
		}
	}

	// Eject the disk if requested
	if eject && device != "" {
		cmd = exec.Command("sudo", "eject", device)
		if output, err := cmd.CombinedOutput(); err != nil {
			// Eject failure is not critical, just log it
			return fmt.Errorf("unmounted but failed to eject: %s", strings.TrimSpace(string(output)))
		}
	}

	return nil
}

// MountDisk mounts a disk at the specified mount point
func MountDisk(device, mountPath string) error {
	if err := ValidateDevicePath(device); err != nil {
		return err
	}

	// Validate mount path
	if mountPath == "" {
		return fmt.Errorf("mount path cannot be empty")
	}
	if !strings.HasPrefix(mountPath, "/") {
		return fmt.Errorf("mount path must be absolute")
	}
	// Security: only allow mounting under /mnt/ or /media/
	if !strings.HasPrefix(mountPath, "/mnt/") && !strings.HasPrefix(mountPath, "/media/") {
		return fmt.Errorf("mount path must be under /mnt/ or /media/")
	}
	// Prevent path traversal
	if strings.Contains(mountPath, "..") {
		return fmt.Errorf("invalid mount path")
	}

	// Create mount point directory
	cmd := exec.Command("sudo", "mkdir", "-p", mountPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create mount point: %s - %w", strings.TrimSpace(string(output)), err)
	}

	// Detect filesystem type
	cmd = exec.Command("lsblk", "-no", "FSTYPE", device)
	fsOutput, _ := cmd.Output()
	fsType := strings.TrimSpace(string(fsOutput))

	// Get current user's UID and GID for mount options
	uid := fmt.Sprintf("%d", os.Getuid())
	gid := fmt.Sprintf("%d", os.Getgid())

	// Mount the disk with appropriate options based on filesystem
	var mountCmd *exec.Cmd
	switch fsType {
	case "vfat", "exfat":
		// FAT filesystems need uid/gid options at mount time
		mountCmd = exec.Command("sudo", "mount", "-o", fmt.Sprintf("uid=%s,gid=%s", uid, gid), device, mountPath)
	default:
		// For ext4, xfs, etc. - mount normally
		mountCmd = exec.Command("sudo", "mount", device, mountPath)
	}

	if output, err := mountCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to mount disk: %s - %w", strings.TrimSpace(string(output)), err)
	}

	// For native Linux filesystems (ext4, xfs), change ownership after mounting
	if fsType != "vfat" && fsType != "exfat" && fsType != "" {
		cmd = exec.Command("sudo", "chown", fmt.Sprintf("%s:%s", uid, gid), mountPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			// Non-fatal, just log warning
			return fmt.Errorf("mounted but failed to set ownership: %s - %w", strings.TrimSpace(string(output)), err)
		}
	}

	return nil
}

// AddToFstab adds a mount entry to /etc/fstab for persistent mounting
func AddToFstab(device, mountPath string) error {
	// Validate inputs
	if err := ValidateDevicePath(device); err != nil {
		return err
	}
	if !strings.HasPrefix(mountPath, "/mnt/") && !strings.HasPrefix(mountPath, "/media/") {
		return fmt.Errorf("mount path must be under /mnt/ or /media/")
	}

	// Get UUID of the device (more reliable than device path)
	cmd := exec.Command("lsblk", "-no", "UUID", device)
	uuidOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get UUID: %w", err)
	}
	uuid := strings.TrimSpace(string(uuidOutput))
	if uuid == "" {
		return fmt.Errorf("device has no UUID")
	}

	// Get filesystem type
	cmd = exec.Command("lsblk", "-no", "FSTYPE", device)
	fsOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get filesystem type: %w", err)
	}
	fsType := strings.TrimSpace(string(fsOutput))
	if fsType == "" {
		return fmt.Errorf("device has no filesystem")
	}

	// Determine mount options based on filesystem
	// Get current user's UID and GID for mount options
	uid := os.Getuid()
	gid := os.Getgid()

	var options string
	switch fsType {
	case "vfat", "exfat":
		// FAT filesystems - use nofail and uid/gid for proper permissions
		options = fmt.Sprintf("defaults,nofail,uid=%d,gid=%d", uid, gid)
	default:
		// Linux filesystems (ext4, xfs, etc.)
		options = "defaults,nofail"
	}

	// Check if entry already exists in fstab
	fstabContent, err := os.ReadFile("/etc/fstab")
	if err != nil {
		return fmt.Errorf("failed to read fstab: %w", err)
	}
	if strings.Contains(string(fstabContent), uuid) {
		return nil // Already exists
	}
	if strings.Contains(string(fstabContent), mountPath) {
		return fmt.Errorf("mount path %s already exists in fstab", mountPath)
	}

	// Create fstab entry
	entry := fmt.Sprintf("\n# Anemone mount - %s\nUUID=%s %s %s %s 0 2\n", device, uuid, mountPath, fsType, options)

	// Append to fstab using tee (requires sudo)
	cmd = exec.Command("sudo", "tee", "-a", "/etc/fstab")
	cmd.Stdin = strings.NewReader(entry)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to update fstab: %s - %w", strings.TrimSpace(string(output)), err)
	}

	log.Printf("Added fstab entry for %s (UUID=%s) at %s", device, uuid, mountPath)
	return nil
}
