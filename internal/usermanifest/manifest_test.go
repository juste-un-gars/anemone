// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package usermanifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildUserManifest(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Create some test files
	testFiles := map[string]string{
		"Documents/report.pdf":   "This is a test PDF content",
		"Documents/notes.txt":    "Some notes here",
		"Images/photo.jpg":       "Fake image data",
		"Projects/code/main.go":  "package main",
		"Projects/code/utils.go": "package main\nfunc helper() {}",
	}

	for relPath, content := range testFiles {
		fullPath := filepath.Join(tempDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	// Build manifest
	manifest, err := BuildUserManifest(tempDir, "data_testuser", "data", "testuser")
	if err != nil {
		t.Fatalf("BuildUserManifest failed: %v", err)
	}

	// Verify manifest metadata
	if manifest.Version != 1 {
		t.Errorf("Expected version 1, got %d", manifest.Version)
	}
	if manifest.ShareName != "data_testuser" {
		t.Errorf("Expected share name 'data_testuser', got '%s'", manifest.ShareName)
	}
	if manifest.ShareType != "data" {
		t.Errorf("Expected share type 'data', got '%s'", manifest.ShareType)
	}
	if manifest.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", manifest.Username)
	}
	if manifest.FileCount != len(testFiles) {
		t.Errorf("Expected %d files, got %d", len(testFiles), manifest.FileCount)
	}

	// Verify all files are present
	fileMap := make(map[string]UserFileEntry)
	for _, f := range manifest.Files {
		fileMap[f.Path] = f
	}

	for relPath, content := range testFiles {
		// Convert to forward slashes for comparison
		normalizedPath := filepath.ToSlash(relPath)
		entry, ok := fileMap[normalizedPath]
		if !ok {
			t.Errorf("File %s not found in manifest", normalizedPath)
			continue
		}

		expectedSize := int64(len(content))
		if entry.Size != expectedSize {
			t.Errorf("File %s: expected size %d, got %d", normalizedPath, expectedSize, entry.Size)
		}

		if entry.Hash == "" || len(entry.Hash) < 10 {
			t.Errorf("File %s: invalid hash '%s'", normalizedPath, entry.Hash)
		}

		if entry.Mtime == 0 {
			t.Errorf("File %s: mtime is zero", normalizedPath)
		}
	}

	// Verify total size
	var expectedTotal int64
	for _, content := range testFiles {
		expectedTotal += int64(len(content))
	}
	if manifest.TotalSize != expectedTotal {
		t.Errorf("Expected total size %d, got %d", expectedTotal, manifest.TotalSize)
	}
}

func TestBuildUserManifest_SkipsHiddenFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create regular and hidden files
	regularFile := filepath.Join(tempDir, "visible.txt")
	hiddenFile := filepath.Join(tempDir, ".hidden")
	hiddenDir := filepath.Join(tempDir, ".hiddendir")
	fileInHiddenDir := filepath.Join(hiddenDir, "secret.txt")
	anemoneDir := filepath.Join(tempDir, ".anemone")
	manifestFile := filepath.Join(anemoneDir, "manifest.json")

	os.WriteFile(regularFile, []byte("visible"), 0644)
	os.WriteFile(hiddenFile, []byte("hidden"), 0644)
	os.MkdirAll(hiddenDir, 0755)
	os.WriteFile(fileInHiddenDir, []byte("secret"), 0644)
	os.MkdirAll(anemoneDir, 0755)
	os.WriteFile(manifestFile, []byte("{}"), 0644)

	manifest, err := BuildUserManifest(tempDir, "test_share", "data", "testuser")
	if err != nil {
		t.Fatalf("BuildUserManifest failed: %v", err)
	}

	// Should only contain the visible file
	if manifest.FileCount != 1 {
		t.Errorf("Expected 1 file, got %d", manifest.FileCount)
	}

	if len(manifest.Files) != 1 || manifest.Files[0].Path != "visible.txt" {
		t.Errorf("Expected only 'visible.txt', got %v", manifest.Files)
	}
}

func TestWriteManifest(t *testing.T) {
	tempDir := t.TempDir()

	manifest := &UserManifest{
		Version:     1,
		GeneratedAt: time.Now().UTC(),
		ShareName:   "test_share",
		ShareType:   "data",
		Username:    "testuser",
		FileCount:   2,
		TotalSize:   1024,
		Files: []UserFileEntry{
			{Path: "file1.txt", Size: 512, Mtime: time.Now().Unix(), Hash: "sha256:abc123"},
			{Path: "file2.txt", Size: 512, Mtime: time.Now().Unix(), Hash: "sha256:def456"},
		},
	}

	err := WriteManifest(manifest, tempDir)
	if err != nil {
		t.Fatalf("WriteManifest failed: %v", err)
	}

	// Verify .anemone directory was created
	anemoneDir := filepath.Join(tempDir, ".anemone")
	if _, err := os.Stat(anemoneDir); os.IsNotExist(err) {
		t.Error(".anemone directory was not created")
	}

	// Verify manifest.json was created
	manifestPath := filepath.Join(anemoneDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read manifest file: %v", err)
	}

	// Verify JSON is valid and contains expected data
	var readManifest UserManifest
	if err := json.Unmarshal(data, &readManifest); err != nil {
		t.Fatalf("Failed to parse manifest JSON: %v", err)
	}

	if readManifest.ShareName != manifest.ShareName {
		t.Errorf("ShareName mismatch: expected %s, got %s", manifest.ShareName, readManifest.ShareName)
	}
	if readManifest.FileCount != manifest.FileCount {
		t.Errorf("FileCount mismatch: expected %d, got %d", manifest.FileCount, readManifest.FileCount)
	}
	if len(readManifest.Files) != len(manifest.Files) {
		t.Errorf("Files count mismatch: expected %d, got %d", len(manifest.Files), len(readManifest.Files))
	}
}

func TestWriteManifest_AtomicWrite(t *testing.T) {
	tempDir := t.TempDir()

	// Write initial manifest
	manifest1 := &UserManifest{
		Version:   1,
		ShareName: "test_share",
		FileCount: 1,
		Files:     []UserFileEntry{{Path: "file1.txt", Size: 100}},
	}
	if err := WriteManifest(manifest1, tempDir); err != nil {
		t.Fatalf("First write failed: %v", err)
	}

	// Write updated manifest
	manifest2 := &UserManifest{
		Version:   1,
		ShareName: "test_share",
		FileCount: 2,
		Files: []UserFileEntry{
			{Path: "file1.txt", Size: 100},
			{Path: "file2.txt", Size: 200},
		},
	}
	if err := WriteManifest(manifest2, tempDir); err != nil {
		t.Fatalf("Second write failed: %v", err)
	}

	// Verify temp file was cleaned up
	tempFile := filepath.Join(tempDir, ".anemone", "manifest.json.tmp")
	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Error("Temporary file should not exist after successful write")
	}

	// Verify final manifest has correct content
	data, _ := os.ReadFile(filepath.Join(tempDir, ".anemone", "manifest.json"))
	var readManifest UserManifest
	json.Unmarshal(data, &readManifest)

	if readManifest.FileCount != 2 {
		t.Errorf("Expected FileCount 2, got %d", readManifest.FileCount)
	}
}

func TestCacheReuse(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "data.txt")
	content := "Test content for caching"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Build first manifest
	manifest1, err := BuildUserManifest(tempDir, "test_share", "data", "testuser")
	if err != nil {
		t.Fatalf("First BuildUserManifest failed: %v", err)
	}

	// Write it so it can be used as cache
	if err := WriteManifest(manifest1, tempDir); err != nil {
		t.Fatalf("WriteManifest failed: %v", err)
	}

	// Build second manifest - should reuse checksum
	manifest2, err := BuildUserManifest(tempDir, "test_share", "data", "testuser")
	if err != nil {
		t.Fatalf("Second BuildUserManifest failed: %v", err)
	}

	// Verify hashes match (checksum was reused)
	if len(manifest1.Files) != 1 || len(manifest2.Files) != 1 {
		t.Fatalf("Expected 1 file in each manifest")
	}

	if manifest1.Files[0].Hash != manifest2.Files[0].Hash {
		t.Errorf("Hashes should match: %s vs %s", manifest1.Files[0].Hash, manifest2.Files[0].Hash)
	}
}

func TestDetermineShareType(t *testing.T) {
	tests := []struct {
		shareName    string
		expectedType string
	}{
		{"data_alice", "data"},
		{"backup_alice", "backup"},
		{"backup", "backup"},
		{"data", "data"},
		{"photos_bob", "data"}, // Unknown prefix defaults to data
		{"alice_backup", "data"}, // backup must be prefix
	}

	for _, tc := range tests {
		t.Run(tc.shareName, func(t *testing.T) {
			result := determineShareType(tc.shareName)
			if result != tc.expectedType {
				t.Errorf("determineShareType(%s) = %s, expected %s", tc.shareName, result, tc.expectedType)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := FormatSize(tc.bytes)
			if result != tc.expected {
				t.Errorf("FormatSize(%d) = %s, expected %s", tc.bytes, result, tc.expected)
			}
		})
	}
}

func TestManifestJSONFormat(t *testing.T) {
	// Test that the JSON format matches AnemoneSync expectations
	manifest := &UserManifest{
		Version:     1,
		GeneratedAt: time.Date(2026, 1, 18, 10, 30, 0, 0, time.UTC),
		ShareName:   "data_alice",
		ShareType:   "data",
		Username:    "alice",
		FileCount:   1,
		TotalSize:   1048576,
		Files: []UserFileEntry{
			{
				Path:  "Documents/report.pdf",
				Size:  1048576,
				Mtime: 1737193800,
				Hash:  "sha256:a1b2c3d4e5f6",
			},
		},
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	// Verify expected fields are present in JSON
	jsonStr := string(data)
	expectedFields := []string{
		`"version": 1`,
		`"share_name": "data_alice"`,
		`"share_type": "data"`,
		`"username": "alice"`,
		`"file_count": 1`,
		`"total_size": 1048576`,
		`"path": "Documents/report.pdf"`,
		`"size": 1048576`,
		`"mtime": 1737193800`,
		`"hash": "sha256:a1b2c3d4e5f6"`,
	}

	for _, field := range expectedFields {
		if !contains(jsonStr, field) {
			t.Errorf("Expected JSON to contain %s", field)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
