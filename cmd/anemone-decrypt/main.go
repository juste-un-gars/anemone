// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// anemone-decrypt is a standalone tool to manually decrypt backup files
// Usage: anemone-decrypt -key=<encryption_key> [-dir=<source_dir>] [-out=<output_dir>]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/juste-un-gars/anemone/internal/crypto"
)

func main() {
	// Parse command line flags
	keyFlag := flag.String("key", "", "Base64-encoded encryption key (32 bytes)")
	dirFlag := flag.String("dir", ".", "Directory containing encrypted files (default: current directory)")
	outFlag := flag.String("out", "", "Output directory for decrypted files (default: same as input)")
	recursiveFlag := flag.Bool("r", false, "Recursively decrypt files in subdirectories")
	helpFlag := flag.Bool("h", false, "Show help")

	flag.Parse()

	// Show help
	if *helpFlag {
		printHelp()
		os.Exit(0)
	}

	// Validate encryption key
	if *keyFlag == "" {
		fmt.Fprintf(os.Stderr, "❌ Error: Encryption key is required\n\n")
		printHelp()
		os.Exit(1)
	}

	// Validate source directory
	sourceDir := *dirFlag
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "❌ Error: Directory does not exist: %s\n", sourceDir)
		os.Exit(1)
	}

	// Set output directory (default: same as input)
	outputDir := *outFlag
	if outputDir == "" {
		outputDir = sourceDir
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: Failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🔐 Anemone Manual Decryption Tool\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Source directory: %s\n", sourceDir)
	fmt.Printf("Output directory: %s\n", outputDir)
	fmt.Printf("Recursive: %v\n\n", *recursiveFlag)

	// Find all .enc files
	encryptedFiles, err := findEncryptedFiles(sourceDir, *recursiveFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: Failed to scan directory: %v\n", err)
		os.Exit(1)
	}

	if len(encryptedFiles) == 0 {
		fmt.Printf("⚠️  No encrypted files found (*.enc)\n")
		os.Exit(0)
	}

	fmt.Printf("Found %d encrypted file(s)\n\n", len(encryptedFiles))

	// Decrypt each file
	successCount := 0
	errorCount := 0

	for i, encFile := range encryptedFiles {
		// Calculate relative path for output
		relPath, err := filepath.Rel(sourceDir, encFile)
		if err != nil {
			relPath = filepath.Base(encFile)
		}

		// Remove .enc extension
		decFileName := strings.TrimSuffix(relPath, ".enc")
		decFilePath := filepath.Join(outputDir, decFileName)

		// Create output subdirectory if needed
		decFileDir := filepath.Dir(decFilePath)
		if err := os.MkdirAll(decFileDir, 0755); err != nil {
			fmt.Printf("[%d/%d] ❌ %s - Failed to create directory: %v\n", i+1, len(encryptedFiles), relPath, err)
			errorCount++
			continue
		}

		// Decrypt file
		fmt.Printf("[%d/%d] 🔓 %s...", i+1, len(encryptedFiles), relPath)

		err = decryptFile(encFile, decFilePath, *keyFlag)
		if err != nil {
			fmt.Printf(" ❌ FAILED\n")
			fmt.Printf("       Error: %v\n", err)
			errorCount++
		} else {
			// Get file size
			info, _ := os.Stat(decFilePath)
			size := "unknown size"
			if info != nil {
				size = formatBytes(info.Size())
			}
			fmt.Printf(" ✅ OK (%s)\n", size)
			successCount++
		}
	}

	// Summary
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✅ Successfully decrypted: %d\n", successCount)
	if errorCount > 0 {
		fmt.Printf("❌ Failed: %d\n", errorCount)
		os.Exit(1)
	}
	fmt.Printf("\n🎉 All files decrypted successfully!\n")
}

// findEncryptedFiles scans directory for .enc files
func findEncryptedFiles(dir string, recursive bool) ([]string, error) {
	var files []string

	if recursive {
		// Walk directory tree
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".enc") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		// Only scan current directory
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".enc") {
				files = append(files, filepath.Join(dir, entry.Name()))
			}
		}
	}

	return files, nil
}

// decryptFile decrypts a single file
func decryptFile(encryptedPath, outputPath, encryptionKey string) error {
	// Open encrypted file
	encFile, err := os.Open(encryptedPath)
	if err != nil {
		return fmt.Errorf("failed to open encrypted file: %w", err)
	}
	defer encFile.Close()

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Decrypt using crypto library
	if err := crypto.DecryptStream(encFile, outFile, encryptionKey); err != nil {
		// Clean up failed output file
		os.Remove(outputPath)
		return fmt.Errorf("decryption failed: %w", err)
	}

	return nil
}

// formatBytes formats byte size in human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// printHelp prints usage information
func printHelp() {
	fmt.Println("🔐 Anemone Manual Decryption Tool")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("DESCRIPTION:")
	fmt.Println("  Manually decrypt backup files that were encrypted by Anemone.")
	fmt.Println("  Useful for disaster recovery when you only have the encrypted")
	fmt.Println("  backups and your encryption key.")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  anemone-decrypt -key=<encryption_key> [options]")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -key string")
	fmt.Println("        Base64-encoded encryption key (32 bytes) - REQUIRED")
	fmt.Println("        This is the key shown when you created/activated your account")
	fmt.Println()
	fmt.Println("  -dir string")
	fmt.Println("        Directory containing encrypted files (default: current directory)")
	fmt.Println()
	fmt.Println("  -out string")
	fmt.Println("        Output directory for decrypted files (default: same as input)")
	fmt.Println()
	fmt.Println("  -r")
	fmt.Println("        Recursively decrypt files in subdirectories")
	fmt.Println()
	fmt.Println("  -h")
	fmt.Println("        Show this help message")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Decrypt all .enc files in current directory")
	fmt.Println("  anemone-decrypt -key=YOUR_BASE64_KEY")
	fmt.Println()
	fmt.Println("  # Decrypt files from a specific directory")
	fmt.Println("  anemone-decrypt -key=YOUR_BASE64_KEY -dir=/path/to/backups")
	fmt.Println()
	fmt.Println("  # Decrypt recursively to a different output directory")
	fmt.Println("  anemone-decrypt -key=YOUR_BASE64_KEY -dir=/backups -out=/restored -r")
	fmt.Println()
	fmt.Println("NOTES:")
	fmt.Println("  - Only files with .enc extension will be decrypted")
	fmt.Println("  - Decrypted files will have the .enc extension removed")
	fmt.Println("  - If decryption fails, the output file will be deleted")
	fmt.Println("  - Original encrypted files are never modified")
	fmt.Println()
}
