// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCalculateChecksum tests SHA-256 checksum calculation
func TestCalculateChecksum(t *testing.T) {
	// Create a temporary file with known content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := []byte("Hello, Anemone!")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate checksum
	checksum, err := CalculateChecksum(testFile)
	if err != nil {
		t.Fatalf("CalculateChecksum failed: %v", err)
	}

	// Check the format (starts with "sha256:" and has correct length)
	if len(checksum) != 71 || checksum[:7] != "sha256:" {
		t.Errorf("Invalid checksum format: %s (length: %d)", checksum, len(checksum))
	}

	// Test consistency: same file should give same checksum
	checksum2, err := CalculateChecksum(testFile)
	if err != nil {
		t.Fatalf("Second CalculateChecksum failed: %v", err)
	}

	if checksum != checksum2 {
		t.Errorf("Checksum not consistent: %s != %s", checksum, checksum2)
	}
}

// TestBuildManifest tests manifest building from a directory
func TestBuildManifest(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"file1.txt":              "Content 1",
		"subdir/file2.txt":       "Content 2",
		"subdir/nested/file3.txt": "Content 3",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", path, err)
		}
	}

	// Create hidden file (should be excluded)
	hiddenFile := filepath.Join(tmpDir, ".hidden.txt")
	if err := os.WriteFile(hiddenFile, []byte("Hidden"), 0644); err != nil {
		t.Fatalf("Failed to create hidden file: %v", err)
	}

	// Create .trash directory (should be excluded)
	trashDir := filepath.Join(tmpDir, ".trash")
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		t.Fatalf("Failed to create .trash dir: %v", err)
	}
	trashFile := filepath.Join(trashDir, "deleted.txt")
	if err := os.WriteFile(trashFile, []byte("Trash"), 0644); err != nil {
		t.Fatalf("Failed to create trash file: %v", err)
	}

	// Build manifest
	manifest, err := BuildManifest(tmpDir, 5, "backup", "test-server")
	if err != nil {
		t.Fatalf("BuildManifest failed: %v", err)
	}

	// Check basic properties
	if manifest.Version != 1 {
		t.Errorf("Expected version 1, got %d", manifest.Version)
	}
	if manifest.UserID != 5 {
		t.Errorf("Expected UserID 5, got %d", manifest.UserID)
	}
	if manifest.ShareName != "backup" {
		t.Errorf("Expected ShareName 'backup', got %s", manifest.ShareName)
	}

	// Check file count (should be 3, not 5 because .hidden and .trash/* are excluded)
	if len(manifest.Files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(manifest.Files))
		for path := range manifest.Files {
			t.Logf("  - %s", path)
		}
	}

	// Check that regular files are included
	expectedFiles := []string{"file1.txt", "subdir/file2.txt", "subdir/nested/file3.txt"}
	for _, expectedPath := range expectedFiles {
		if _, exists := manifest.Files[expectedPath]; !exists {
			t.Errorf("Expected file %s not found in manifest", expectedPath)
		}
	}

	// Check that hidden file is excluded
	if _, exists := manifest.Files[".hidden.txt"]; exists {
		t.Errorf("Hidden file should be excluded from manifest")
	}

	// Check that trash files are excluded
	if _, exists := manifest.Files[".trash/deleted.txt"]; exists {
		t.Errorf("Trash file should be excluded from manifest")
	}

	// Check file metadata
	meta := manifest.Files["file1.txt"]
	if meta.Size != 9 { // "Content 1" = 9 bytes
		t.Errorf("Expected size 9, got %d", meta.Size)
	}
	if meta.Checksum == "" || len(meta.Checksum) != 71 {
		t.Errorf("Invalid checksum: %s", meta.Checksum)
	}
	if meta.EncryptedPath != "file1.txt.enc" {
		t.Errorf("Expected encrypted_path 'file1.txt.enc', got %s", meta.EncryptedPath)
	}
}

// TestCompareManifests_AllNew tests comparison when remote is nil (first sync)
func TestCompareManifests_AllNew(t *testing.T) {
	local := &SyncManifest{
		Version:   1,
		LastSync:  time.Now(),
		UserID:    5,
		ShareName: "backup",
		Files: map[string]FileMetadata{
			"file1.txt": {Size: 100, Checksum: "sha256:abc123"},
			"file2.txt": {Size: 200, Checksum: "sha256:def456"},
		},
	}

	// Compare with nil remote (first sync)
	delta, err := CompareManifests(local, nil)
	if err != nil {
		t.Fatalf("CompareManifests failed: %v", err)
	}

	// All files should be marked as ToAdd
	if len(delta.ToAdd) != 2 {
		t.Errorf("Expected 2 files to add, got %d", len(delta.ToAdd))
	}
	if len(delta.ToUpdate) != 0 {
		t.Errorf("Expected 0 files to update, got %d", len(delta.ToUpdate))
	}
	if len(delta.ToDelete) != 0 {
		t.Errorf("Expected 0 files to delete, got %d", len(delta.ToDelete))
	}
}

// TestCompareManifests_WithChanges tests detection of added, modified, and deleted files
func TestCompareManifests_WithChanges(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)

	local := &SyncManifest{
		Version:   1,
		LastSync:  now,
		UserID:    5,
		ShareName: "backup",
		Files: map[string]FileMetadata{
			"unchanged.txt": {Size: 100, ModTime: earlier, Checksum: "sha256:aaa111"},
			"modified.txt":  {Size: 200, ModTime: now, Checksum: "sha256:bbb222"},     // Modified (different checksum)
			"new.txt":       {Size: 300, ModTime: now, Checksum: "sha256:ccc333"},     // New file
		},
	}

	remote := &SyncManifest{
		Version:   1,
		LastSync:  earlier,
		UserID:    5,
		ShareName: "backup",
		Files: map[string]FileMetadata{
			"unchanged.txt": {Size: 100, ModTime: earlier, Checksum: "sha256:aaa111"}, // Same
			"modified.txt":  {Size: 150, ModTime: earlier, Checksum: "sha256:bbb111"}, // Different
			"deleted.txt":   {Size: 400, ModTime: earlier, Checksum: "sha256:ddd444"}, // Deleted locally
		},
	}

	delta, err := CompareManifests(local, remote)
	if err != nil {
		t.Fatalf("CompareManifests failed: %v", err)
	}

	// Check ToAdd (new.txt)
	if len(delta.ToAdd) != 1 {
		t.Errorf("Expected 1 file to add, got %d: %v", len(delta.ToAdd), delta.ToAdd)
	} else if delta.ToAdd[0] != "new.txt" {
		t.Errorf("Expected 'new.txt' to add, got %s", delta.ToAdd[0])
	}

	// Check ToUpdate (modified.txt)
	if len(delta.ToUpdate) != 1 {
		t.Errorf("Expected 1 file to update, got %d: %v", len(delta.ToUpdate), delta.ToUpdate)
	} else if delta.ToUpdate[0] != "modified.txt" {
		t.Errorf("Expected 'modified.txt' to update, got %s", delta.ToUpdate[0])
	}

	// Check ToDelete (deleted.txt)
	if len(delta.ToDelete) != 1 {
		t.Errorf("Expected 1 file to delete, got %d: %v", len(delta.ToDelete), delta.ToDelete)
	} else if delta.ToDelete[0] != "deleted.txt" {
		t.Errorf("Expected 'deleted.txt' to delete, got %s", delta.ToDelete[0])
	}
}

// TestMarshalUnmarshalManifest tests JSON serialization
func TestMarshalUnmarshalManifest(t *testing.T) {
	original := &SyncManifest{
		Version:   1,
		LastSync:  time.Now().Truncate(time.Second), // Truncate to avoid nanosecond precision issues
		UserID:    5,
		ShareName: "backup",
		Files: map[string]FileMetadata{
			"file1.txt": {
				Size:          1024,
				ModTime:       time.Now().Truncate(time.Second),
				Checksum:      "sha256:abc123def456",
				EncryptedPath: "file1.txt.enc",
			},
		},
	}

	// Marshal to JSON
	data, err := MarshalManifest(original)
	if err != nil {
		t.Fatalf("MarshalManifest failed: %v", err)
	}

	// Check JSON is not empty
	if len(data) == 0 {
		t.Errorf("Marshaled data is empty")
	}

	// Unmarshal back
	restored, err := UnmarshalManifest(data)
	if err != nil {
		t.Fatalf("UnmarshalManifest failed: %v", err)
	}

	// Compare values
	if restored.Version != original.Version {
		t.Errorf("Version mismatch: %d != %d", restored.Version, original.Version)
	}
	if restored.UserID != original.UserID {
		t.Errorf("UserID mismatch: %d != %d", restored.UserID, original.UserID)
	}
	if restored.ShareName != original.ShareName {
		t.Errorf("ShareName mismatch: %s != %s", restored.ShareName, original.ShareName)
	}
	if len(restored.Files) != len(original.Files) {
		t.Errorf("Files count mismatch: %d != %d", len(restored.Files), len(original.Files))
	}

	// Check file metadata
	origMeta := original.Files["file1.txt"]
	restMeta := restored.Files["file1.txt"]

	if restMeta.Size != origMeta.Size {
		t.Errorf("File size mismatch: %d != %d", restMeta.Size, origMeta.Size)
	}
	if restMeta.Checksum != origMeta.Checksum {
		t.Errorf("Checksum mismatch: %s != %s", restMeta.Checksum, origMeta.Checksum)
	}
}

// TestGetManifestStats tests statistics calculation
func TestGetManifestStats(t *testing.T) {
	manifest := &SyncManifest{
		Version:   1,
		LastSync:  time.Now(),
		UserID:    5,
		ShareName: "backup",
		Files: map[string]FileMetadata{
			"file1.txt": {Size: 1000},
			"file2.txt": {Size: 2000},
			"file3.txt": {Size: 3000},
		},
	}

	fileCount, totalSize := GetManifestStats(manifest)

	if fileCount != 3 {
		t.Errorf("Expected 3 files, got %d", fileCount)
	}
	if totalSize != 6000 {
		t.Errorf("Expected total size 6000, got %d", totalSize)
	}
}

// TestPrintDelta tests delta summary formatting
func TestPrintDelta(t *testing.T) {
	delta := &SyncDelta{
		ToAdd:    []string{"new1.txt", "new2.txt"},
		ToUpdate: []string{"modified.txt"},
		ToDelete: []string{"deleted1.txt", "deleted2.txt", "deleted3.txt"},
	}

	summary := PrintDelta(delta)
	expected := "Delta: 2 to add, 1 to update, 3 to delete"

	if summary != expected {
		t.Errorf("Expected '%s', got '%s'", expected, summary)
	}
}
