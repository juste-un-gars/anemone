// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// test-manifest is a demonstration program to test the manifest system
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/juste-un-gars/anemone/internal/sync"
)

func main() {
	// Create a temporary test directory
	tmpDir, err := os.MkdirTemp("", "anemone-manifest-test-*")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Printf("🧪 Testing Anemone Manifest System\n")
	fmt.Printf("📁 Test directory: %s\n\n", tmpDir)

	// Create test file structure
	createTestFiles(tmpDir)

	// Phase 1: Build initial manifest
	fmt.Println("📋 Phase 1: Building initial manifest...")
	manifest1, err := sync.BuildManifest(tmpDir, 5, "backup")
	if err != nil {
		log.Fatalf("Failed to build manifest: %v", err)
	}

	fileCount1, totalSize1 := sync.GetManifestStats(manifest1)
	fmt.Printf("   ✓ Files: %d\n", fileCount1)
	fmt.Printf("   ✓ Total size: %d bytes (%.2f KB)\n", totalSize1, float64(totalSize1)/1024)
	fmt.Printf("   ✓ Files list:\n")
	for path, meta := range manifest1.Files {
		fmt.Printf("      - %s (%d bytes, checksum: %s...)\n",
			path, meta.Size, meta.Checksum[:20])
	}

	// Phase 2: Simulate changes
	fmt.Println("\n🔄 Phase 2: Simulating file changes...")

	// Add a new file
	newFile := filepath.Join(tmpDir, "photos", "new_photo.jpg")
	if err := os.WriteFile(newFile, []byte("New photo content"), 0644); err != nil {
		log.Fatalf("Failed to create new file: %v", err)
	}
	fmt.Println("   + Added: photos/new_photo.jpg")

	// Modify an existing file
	modifiedFile := filepath.Join(tmpDir, "documents", "report.pdf")
	if err := os.WriteFile(modifiedFile, []byte("Modified report content - VERSION 2"), 0644); err != nil {
		log.Fatalf("Failed to modify file: %v", err)
	}
	fmt.Println("   ✏️  Modified: documents/report.pdf")

	// Delete a file
	deletedFile := filepath.Join(tmpDir, "documents", "invoice.xlsx")
	if err := os.Remove(deletedFile); err != nil {
		log.Fatalf("Failed to delete file: %v", err)
	}
	fmt.Println("   - Deleted: documents/invoice.xlsx")

	// Phase 3: Build new manifest
	fmt.Println("\n📋 Phase 3: Building updated manifest...")
	manifest2, err := sync.BuildManifest(tmpDir, 5, "backup")
	if err != nil {
		log.Fatalf("Failed to build manifest: %v", err)
	}

	fileCount2, totalSize2 := sync.GetManifestStats(manifest2)
	fmt.Printf("   ✓ Files: %d\n", fileCount2)
	fmt.Printf("   ✓ Total size: %d bytes (%.2f KB)\n", totalSize2, float64(totalSize2)/1024)

	// Phase 4: Compare manifests
	fmt.Println("\n🔍 Phase 4: Comparing manifests...")
	delta, err := sync.CompareManifests(manifest2, manifest1)
	if err != nil {
		log.Fatalf("Failed to compare manifests: %v", err)
	}

	fmt.Printf("   %s\n", sync.PrintDelta(delta))

	if len(delta.ToAdd) > 0 {
		fmt.Println("\n   ➕ Files to ADD:")
		for _, path := range delta.ToAdd {
			fmt.Printf("      - %s\n", path)
		}
	}

	if len(delta.ToUpdate) > 0 {
		fmt.Println("\n   ✏️  Files to UPDATE:")
		for _, path := range delta.ToUpdate {
			fmt.Printf("      - %s\n", path)
		}
	}

	if len(delta.ToDelete) > 0 {
		fmt.Println("\n   🗑️  Files to DELETE on remote:")
		for _, path := range delta.ToDelete {
			fmt.Printf("      - %s\n", path)
		}
	}

	// Phase 5: Test JSON serialization
	fmt.Println("\n💾 Phase 5: Testing JSON serialization...")
	jsonData, err := sync.MarshalManifest(manifest2)
	if err != nil {
		log.Fatalf("Failed to marshal manifest: %v", err)
	}
	fmt.Printf("   ✓ Manifest JSON size: %d bytes\n", len(jsonData))

	restored, err := sync.UnmarshalManifest(jsonData)
	if err != nil {
		log.Fatalf("Failed to unmarshal manifest: %v", err)
	}
	restoredCount, restoredSize := sync.GetManifestStats(restored)
	fmt.Printf("   ✓ Restored manifest: %d files, %d bytes\n", restoredCount, restoredSize)

	// Final summary
	fmt.Println("\n✅ All tests completed successfully!")
	fmt.Println("\n📊 Summary:")
	fmt.Printf("   • Initial manifest:  %d files, %d bytes\n", fileCount1, totalSize1)
	fmt.Printf("   • Updated manifest:  %d files, %d bytes\n", fileCount2, totalSize2)
	fmt.Printf("   • Changes detected:  %d add, %d update, %d delete\n",
		len(delta.ToAdd), len(delta.ToUpdate), len(delta.ToDelete))
	fmt.Printf("   • JSON round-trip:   ✓ Success\n")
}

func createTestFiles(baseDir string) {
	// Create directory structure
	dirs := []string{
		"documents",
		"photos",
		"videos",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(baseDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			log.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	// Create test files
	files := map[string]string{
		"documents/report.pdf":       "This is a PDF report content",
		"documents/invoice.xlsx":     "Invoice spreadsheet data",
		"photos/vacation.jpg":        "JPEG photo binary data would be here",
		"videos/birthday.mp4":        "MP4 video binary data would be here",
	}

	for path, content := range files {
		fullPath := filepath.Join(baseDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			log.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	// Create hidden directory (should be excluded)
	hiddenDir := filepath.Join(baseDir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		log.Fatalf("Failed to create hidden dir: %v", err)
	}
	hiddenFile := filepath.Join(hiddenDir, "secret.txt")
	if err := os.WriteFile(hiddenFile, []byte("Hidden content"), 0644); err != nil {
		log.Fatalf("Failed to create hidden file: %v", err)
	}

	// Create .trash directory (should be excluded)
	trashDir := filepath.Join(baseDir, ".trash")
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		log.Fatalf("Failed to create trash dir: %v", err)
	}
	trashFile := filepath.Join(trashDir, "deleted.txt")
	if err := os.WriteFile(trashFile, []byte("Deleted content"), 0644); err != nil {
		log.Fatalf("Failed to create trash file: %v", err)
	}

	fmt.Println("📂 Created test file structure:")
	fmt.Println("   ✓ 3 directories (documents, photos, videos)")
	fmt.Println("   ✓ 4 files (report, invoice, vacation, birthday)")
	fmt.Println("   ✓ 1 hidden directory (.hidden) - will be excluded")
	fmt.Println("   ✓ 1 trash directory (.trash) - will be excluded")
	fmt.Println()
}
