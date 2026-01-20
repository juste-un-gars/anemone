// Package storage provides ZFS snapshot management operations.
package storage

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Snapshot represents a ZFS snapshot
type Snapshot struct {
	Name         string    `json:"name"`          // Full name (dataset@snapshot)
	Dataset      string    `json:"dataset"`       // Parent dataset
	SnapName     string    `json:"snap_name"`     // Just the snapshot name (after @)
	Used         uint64    `json:"used"`          // Space used by snapshot
	UsedHuman    string    `json:"used_human"`
	Referenced   uint64    `json:"referenced"`    // Space referenced
	RefHuman     string    `json:"ref_human"`
	CreationTime time.Time `json:"creation_time"` // When snapshot was created
	CreationStr  string    `json:"creation_str"`  // Human-readable creation time
}

// SnapshotCreateOptions contains options for creating a snapshot
type SnapshotCreateOptions struct {
	Dataset   string `json:"dataset"`   // Dataset to snapshot
	Name      string `json:"name"`      // Snapshot name (without @)
	Recursive bool   `json:"recursive"` // Snapshot child datasets too
}

// ValidateSnapshotName checks if a snapshot name is valid
func ValidateSnapshotName(name string) error {
	if name == "" {
		return fmt.Errorf("snapshot name cannot be empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("snapshot name too long")
	}
	// Full snapshot name must contain @
	if !strings.Contains(name, "@") {
		return fmt.Errorf("snapshot name must be in format dataset@snapshot")
	}
	// Validate characters
	validName := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_\-.:/@]*$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid snapshot name")
	}
	return nil
}

// ValidateSnapNameOnly validates just the snapshot portion (after @)
func ValidateSnapNameOnly(name string) error {
	if name == "" {
		return fmt.Errorf("snapshot name cannot be empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("snapshot name too long")
	}
	if strings.Contains(name, "@") {
		return fmt.Errorf("snapshot name should not contain @")
	}
	if strings.Contains(name, "/") {
		return fmt.Errorf("snapshot name should not contain /")
	}
	validName := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_\-.:]*$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid snapshot name: must start with alphanumeric")
	}
	return nil
}

// CreateSnapshot creates a new ZFS snapshot
func CreateSnapshot(opts SnapshotCreateOptions) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateDatasetNameOrPool(opts.Dataset); err != nil {
		return fmt.Errorf("invalid dataset: %w", err)
	}

	if err := ValidateSnapNameOnly(opts.Name); err != nil {
		return fmt.Errorf("invalid snapshot name: %w", err)
	}

	fullName := fmt.Sprintf("%s@%s", opts.Dataset, opts.Name)

	args := []string{"zfs", "snapshot"}
	if opts.Recursive {
		args = append(args, "-r")
	}
	args = append(args, fullName)

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// DeleteSnapshot destroys a ZFS snapshot
func DeleteSnapshot(name string, recursive bool, force bool) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateSnapshotName(name); err != nil {
		return err
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
		return fmt.Errorf("failed to delete snapshot: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// ListSnapshots lists all snapshots for a dataset
func ListSnapshots(dataset string) ([]Snapshot, error) {
	if !IsZFSAvailable() {
		return nil, fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateDatasetNameOrPool(dataset); err != nil {
		return nil, err
	}

	cmd := exec.Command("sudo", "zfs", "list", "-H", "-p", "-t", "snapshot",
		"-o", "name,used,referenced,creation",
		"-r", dataset)
	output, err := cmd.Output()
	if err != nil {
		// If no snapshots, zfs list may return error
		if strings.Contains(string(output), "no datasets available") {
			return []Snapshot{}, nil
		}
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	var snapshots []Snapshot

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		name := fields[0]
		if !strings.Contains(name, "@") {
			continue
		}

		parts := strings.SplitN(name, "@", 2)
		if len(parts) != 2 {
			continue
		}

		snap := Snapshot{
			Name:     name,
			Dataset:  parts[0],
			SnapName: parts[1],
		}

		if used, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
			snap.Used = used
			snap.UsedHuman = FormatBytes(used)
		}
		if ref, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
			snap.Referenced = ref
			snap.RefHuman = FormatBytes(ref)
		}

		// Parse creation time (Unix timestamp)
		if ts, err := strconv.ParseInt(fields[3], 10, 64); err == nil {
			snap.CreationTime = time.Unix(ts, 0)
			snap.CreationStr = snap.CreationTime.Format("2006-01-02 15:04:05")
		}

		snapshots = append(snapshots, snap)
	}

	return snapshots, nil
}

// ListAllSnapshots lists all snapshots on the system
func ListAllSnapshots() ([]Snapshot, error) {
	if !IsZFSAvailable() {
		return nil, fmt.Errorf("ZFS is not available on this system")
	}

	cmd := exec.Command("sudo", "zfs", "list", "-H", "-p", "-t", "snapshot",
		"-o", "name,used,referenced,creation")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	var snapshots []Snapshot

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}

		name := fields[0]
		if !strings.Contains(name, "@") {
			continue
		}

		parts := strings.SplitN(name, "@", 2)
		if len(parts) != 2 {
			continue
		}

		snap := Snapshot{
			Name:     name,
			Dataset:  parts[0],
			SnapName: parts[1],
		}

		if used, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
			snap.Used = used
			snap.UsedHuman = FormatBytes(used)
		}
		if ref, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
			snap.Referenced = ref
			snap.RefHuman = FormatBytes(ref)
		}
		if ts, err := strconv.ParseInt(fields[3], 10, 64); err == nil {
			snap.CreationTime = time.Unix(ts, 0)
			snap.CreationStr = snap.CreationTime.Format("2006-01-02 15:04:05")
		}

		snapshots = append(snapshots, snap)
	}

	return snapshots, nil
}

// RollbackOptions contains options for rollback operation
type RollbackOptions struct {
	Snapshot      string `json:"snapshot"`       // Full snapshot name (dataset@snapshot)
	Force         bool   `json:"force"`          // Force unmount
	DestroyRecent bool   `json:"destroy_recent"` // Destroy more recent snapshots (-r)
}

// Rollback rolls back a dataset to a snapshot
func Rollback(opts RollbackOptions) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateSnapshotName(opts.Snapshot); err != nil {
		return err
	}

	args := []string{"zfs", "rollback"}
	if opts.Force {
		args = append(args, "-f")
	}
	if opts.DestroyRecent {
		args = append(args, "-r")
	}
	args = append(args, opts.Snapshot)

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to rollback: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// CloneOptions contains options for clone operation
type CloneOptions struct {
	Snapshot   string            `json:"snapshot"`   // Source snapshot
	Target     string            `json:"target"`     // Target dataset name
	Properties map[string]string `json:"properties"` // Properties to set on clone
}

// Clone creates a clone from a snapshot
func Clone(opts CloneOptions) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateSnapshotName(opts.Snapshot); err != nil {
		return err
	}

	if err := ValidateDatasetName(opts.Target); err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}

	args := []string{"zfs", "clone"}

	// Add properties
	for key, value := range opts.Properties {
		args = append(args, "-o", fmt.Sprintf("%s=%s", key, value))
	}

	args = append(args, opts.Snapshot, opts.Target)

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// RenameSnapshot renames a snapshot
func RenameSnapshot(oldName, newName string) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateSnapshotName(oldName); err != nil {
		return fmt.Errorf("invalid old name: %w", err)
	}
	if err := ValidateSnapshotName(newName); err != nil {
		return fmt.Errorf("invalid new name: %w", err)
	}

	cmd := exec.Command("sudo", "zfs", "rename", oldName, newName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to rename snapshot: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// HoldSnapshot places a hold on a snapshot, preventing deletion
func HoldSnapshot(name, tag string) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateSnapshotName(name); err != nil {
		return err
	}

	if tag == "" {
		tag = "anemone"
	}

	cmd := exec.Command("sudo", "zfs", "hold", tag, name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to hold snapshot: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// ReleaseSnapshot releases a hold on a snapshot
func ReleaseSnapshot(name, tag string) error {
	if !IsZFSAvailable() {
		return fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateSnapshotName(name); err != nil {
		return err
	}

	if tag == "" {
		tag = "anemone"
	}

	cmd := exec.Command("sudo", "zfs", "release", tag, name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to release snapshot: %s - %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// GetSnapshotDiff shows differences between a snapshot and the live dataset
func GetSnapshotDiff(snapshot string) (string, error) {
	if !IsZFSAvailable() {
		return "", fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateSnapshotName(snapshot); err != nil {
		return "", err
	}

	cmd := exec.Command("sudo", "zfs", "diff", snapshot)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	return string(output), nil
}

// SendOptions contains options for send operation
type SendOptions struct {
	Snapshot    string `json:"snapshot"`
	Incremental string `json:"incremental"` // Base snapshot for incremental send
	Raw         bool   `json:"raw"`         // Raw send (encrypted)
	Compressed  bool   `json:"compressed"`  // Compressed stream
}

// EstimateSendSize estimates the size of a send stream
func EstimateSendSize(opts SendOptions) (uint64, error) {
	if !IsZFSAvailable() {
		return 0, fmt.Errorf("ZFS is not available on this system")
	}

	if err := ValidateSnapshotName(opts.Snapshot); err != nil {
		return 0, err
	}

	args := []string{"zfs", "send", "-nv"}
	if opts.Incremental != "" {
		if err := ValidateSnapshotName(opts.Incremental); err != nil {
			return 0, fmt.Errorf("invalid incremental snapshot: %w", err)
		}
		args = append(args, "-i", opts.Incremental)
	}
	args = append(args, opts.Snapshot)

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to estimate send size: %s - %w", strings.TrimSpace(string(output)), err)
	}

	// Parse output to find size
	// Example output: "size    1.23G"
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "size") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return parseSize(fields[len(fields)-1])
			}
		}
	}

	return 0, nil
}

// parseSize parses a human-readable size string to bytes
func parseSize(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	multiplier := uint64(1)
	suffix := s[len(s)-1]

	switch suffix {
	case 'K':
		multiplier = 1024
		s = s[:len(s)-1]
	case 'M':
		multiplier = 1024 * 1024
		s = s[:len(s)-1]
	case 'G':
		multiplier = 1024 * 1024 * 1024
		s = s[:len(s)-1]
	case 'T':
		multiplier = 1024 * 1024 * 1024 * 1024
		s = s[:len(s)-1]
	}

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}

	return uint64(val * float64(multiplier)), nil
}
