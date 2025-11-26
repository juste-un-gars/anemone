// Anemone dfree - Disk space calculator for Samba quota enforcement
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const (
	// Block size in bytes (1KB blocks)
	blockSize = 1024
	// Default quota if not found (100GB)
	defaultQuotaGB = 100
)

func main() {
	// Samba calls: dfree_script <share_path>
	// Example: /srv/anemone/shares/smith/backup
	if len(os.Args) < 2 {
		// Fallback: return unlimited space
		fmt.Println("1024 10737418240 10737418240") // 10TB
		os.Exit(0)
	}

	sharePath := os.Args[1]

	// Extract username and share type from path
	// Path format: /srv/anemone/shares/{username}/{backup|data}
	parts := strings.Split(filepath.Clean(sharePath), string(os.PathSeparator))
	if len(parts) < 2 {
		fmt.Println("1024 10737418240 10737418240")
		os.Exit(0)
	}

	var username, shareType string
	for i, part := range parts {
		if part == "shares" && i+2 < len(parts) {
			username = parts[i+1]
			shareType = parts[i+2]
			break
		}
	}

	if username == "" {
		fmt.Println("1024 10737418240 10737418240")
		os.Exit(0)
	}

	// Get data directory from environment or use default
	// When called by Samba, environment variables are not available
	dataDir := os.Getenv("ANEMONE_DATA_DIR")
	if dataDir == "" {
		dataDir = "/srv/anemone" // Default production path
	}

	// Try to find database in common locations if default doesn't exist
	dbPath := filepath.Join(dataDir, "db", "anemone.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Try current directory
		if _, err := os.Stat("./data/db/anemone.db"); err == nil {
			dbPath = "./data/db/anemone.db"
		}
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		// Fallback on error
		fmt.Println("1024 10737418240 10737418240")
		os.Exit(0)
	}
	defer db.Close()

	// Get user quota from database
	var quotaTotalGB, quotaBackupGB int
	err = db.QueryRow(`
		SELECT quota_total_gb, quota_backup_gb
		FROM users
		WHERE username = ?
	`, username).Scan(&quotaTotalGB, &quotaBackupGB)

	if err != nil {
		// User not found or error - use default
		quotaTotalGB = defaultQuotaGB
		quotaBackupGB = defaultQuotaGB / 2
	}

	// Determine which quota to use based on share type
	// Quotas are independent: backup uses quota_backup_gb, data uses quota_total_gb
	var quotaGB int
	if shareType == "backup" || strings.HasPrefix(shareType, "backup_") {
		quotaGB = quotaBackupGB
	} else if shareType == "data" || strings.HasPrefix(shareType, "data_") {
		quotaGB = quotaTotalGB
	} else {
		// Unknown share type - return unlimited
		fmt.Println("1024 10737418240 10737418240")
		os.Exit(0)
	}

	// Calculate used space
	usedBytes := calculateDirectorySize(sharePath)
	usedBlocks := usedBytes / blockSize

	// Calculate total and free blocks
	var totalBlocks, freeBlocks int64
	if quotaGB == 0 {
		// Unlimited quota: return very large value (10 TB)
		totalBlocks = 10 * 1024 * 1024 * 1024 // 10 TB in KB blocks
		freeBlocks = totalBlocks - usedBlocks
	} else {
		totalBlocks = int64(quotaGB) * 1024 * 1024 // GB to KB blocks
		freeBlocks = totalBlocks - usedBlocks
	}

	// Ensure free blocks is not negative
	if freeBlocks < 0 {
		freeBlocks = 0
	}

	// Output in Samba dfree format: blocksize total_blocks free_blocks
	fmt.Printf("%d %d %d\n", blockSize, totalBlocks, freeBlocks)
}

// calculateDirectorySize calculates the total size of a directory in bytes
func calculateDirectorySize(path string) int64 {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0
	}

	return size
}
