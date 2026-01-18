// Package btrfs provides utilities for working with Btrfs filesystems.
package btrfs

import "os/exec"

// IsSubvolume checks if a path is a Btrfs subvolume.
// This runs without sudo - use IsSubvolumeSudo if elevated privileges are needed.
func IsSubvolume(path string) bool {
	cmd := exec.Command("btrfs", "subvolume", "show", path)
	return cmd.Run() == nil
}

// IsSubvolumeSudo checks if a path is a Btrfs subvolume using sudo.
// Use this when the calling process doesn't have direct access to btrfs commands.
func IsSubvolumeSudo(path string) bool {
	cmd := exec.Command("sudo", "btrfs", "subvolume", "show", path)
	return cmd.Run() == nil
}

// DeleteSubvolume deletes a Btrfs subvolume using sudo.
// Returns the combined output and any error.
func DeleteSubvolume(path string) ([]byte, error) {
	cmd := exec.Command("sudo", "btrfs", "subvolume", "delete", path)
	return cmd.CombinedOutput()
}
