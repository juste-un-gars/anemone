// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package btrfs

import (
	"os"
	"testing"
)

// TestIsSubvolume tests the IsSubvolume function
// Note: This test runs without requiring a real btrfs filesystem
func TestIsSubvolume(t *testing.T) {
	// Non-existent path should return false
	result := IsSubvolume("/nonexistent/path/that/does/not/exist")
	if result {
		t.Error("IsSubvolume should return false for non-existent path")
	}

	// Regular directory (not a subvolume) should return false
	// Use temp directory which is unlikely to be a btrfs subvolume
	tmpDir := os.TempDir()
	result = IsSubvolume(tmpDir)
	// We don't assert the result here because it depends on the filesystem
	// This just verifies the function doesn't panic
	t.Logf("IsSubvolume(%s) = %v", tmpDir, result)
}

// TestIsSubvolumeSudo tests the IsSubvolumeSudo function
// Note: This test may fail if sudo is not available or requires password
func TestIsSubvolumeSudo(t *testing.T) {
	// Skip if not running with sudo capabilities
	if os.Getuid() != 0 {
		t.Skip("Skipping sudo test - not running as root")
	}

	// Non-existent path should return false
	result := IsSubvolumeSudo("/nonexistent/path/that/does/not/exist")
	if result {
		t.Error("IsSubvolumeSudo should return false for non-existent path")
	}
}

// TestDeleteSubvolume tests the DeleteSubvolume function
// Note: This test only verifies error handling, not actual deletion
func TestDeleteSubvolume(t *testing.T) {
	// Skip if not running with sudo capabilities
	if os.Getuid() != 0 {
		t.Skip("Skipping sudo test - not running as root")
	}

	// Non-existent path should return error
	output, err := DeleteSubvolume("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("DeleteSubvolume should return error for non-existent path")
	}
	t.Logf("DeleteSubvolume error output: %s", string(output))
}

// TestIsSubvolumeEmptyPath tests behavior with empty path
func TestIsSubvolumeEmptyPath(t *testing.T) {
	// Empty path should return false (command will fail)
	result := IsSubvolume("")
	if result {
		t.Error("IsSubvolume should return false for empty path")
	}
}
