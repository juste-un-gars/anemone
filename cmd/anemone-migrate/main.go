// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/juste-un-gars/anemone/internal/quota"
	"github.com/juste-un-gars/anemone/internal/shares"
	"github.com/juste-un-gars/anemone/internal/users"
)

func main() {
	dataDir := flag.String("data-dir", os.Getenv("ANEMONE_DATA_DIR"), "Data directory (default: $ANEMONE_DATA_DIR)")
	dryRun := flag.Bool("dry-run", false, "Dry run mode (don't actually migrate)")
	force := flag.Bool("force", false, "Force migration even if already a subvolume")
	flag.Parse()

	if *dataDir == "" {
		*dataDir = "./data"
	}

	dbPath := filepath.Join(*dataDir, "db", "anemone.db")
	sharesDir := filepath.Join(*dataDir, "shares")

	log.Printf("🪸 Anemone Share Migration Tool")
	log.Printf("Data dir: %s", *dataDir)
	log.Printf("Database: %s", dbPath)
	log.Printf("Shares dir: %s", sharesDir)
	log.Printf("Dry run: %v", *dryRun)
	log.Printf("")

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Initialize quota manager
	qm, err := quota.NewQuotaManager(sharesDir)
	if err != nil {
		log.Fatalf("Failed to initialize quota manager: %v", err)
	}

	// Get all shares
	allShares, err := shares.GetAll(db)
	if err != nil {
		log.Fatalf("Failed to get shares: %v", err)
	}

	if len(allShares) == 0 {
		log.Printf("No shares found in database")
		return
	}

	log.Printf("Found %d shares to process\n", len(allShares))

	successCount := 0
	skipCount := 0
	errorCount := 0

	for _, share := range allShares {
		log.Printf("\n---")
		log.Printf("Processing share: %s (ID: %d, User ID: %d)", share.Name, share.ID, share.UserID)
		log.Printf("  Path: %s", share.Path)

		// Get user info for quota
		user, err := users.GetByID(db, share.UserID)
		if err != nil {
			log.Printf("  ❌ ERROR: Failed to get user info: %v", err)
			errorCount++
			continue
		}

		// Determine quota based on share name
		var quotaGB int
		if share.Name == "backup" || filepath.Base(share.Path) == "backup" {
			quotaGB = user.QuotaBackupGB
		} else {
			// Data share quota = total - backup
			quotaGB = user.QuotaTotalGB - user.QuotaBackupGB
			if quotaGB < 0 {
				quotaGB = user.QuotaTotalGB / 2
			}
		}

		log.Printf("  User: %s (Quota: %dGB)", user.Username, quotaGB)

		// Check if path exists
		info, err := os.Stat(share.Path)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("  ⚠️  SKIP: Path does not exist")
				skipCount++
				continue
			}
			log.Printf("  ❌ ERROR: Failed to stat path: %v", err)
			errorCount++
			continue
		}

		if !info.IsDir() {
			log.Printf("  ❌ ERROR: Path is not a directory")
			errorCount++
			continue
		}

		// Check if already a subvolume
		isSubvol := isSubvolume(share.Path)
		if isSubvol && !*force {
			log.Printf("  ⏭️  SKIP: Already a subvolume")
			skipCount++
			continue
		}

		if *dryRun {
			if isSubvol {
				log.Printf("  [DRY RUN] Would update quota to %dGB", quotaGB)
			} else {
				log.Printf("  [DRY RUN] Would migrate to subvolume with %dGB quota", quotaGB)
			}
			successCount++
			continue
		}

		// Perform migration
		if isSubvol {
			// Just update quota
			log.Printf("  🔄 Updating quota to %dGB...", quotaGB)
			if err := qm.UpdateQuota(share.Path, quotaGB); err != nil {
				log.Printf("  ❌ ERROR: Failed to update quota: %v", err)
				errorCount++
				continue
			}
			log.Printf("  ✅ Quota updated successfully")
		} else {
			// Full migration: regular dir → subvolume with quota
			log.Printf("  🔄 Migrating to subvolume with %dGB quota...", quotaGB)

			backupPath := share.Path + ".backup"

			// Step 1: Rename original directory to .backup
			if err := os.Rename(share.Path, backupPath); err != nil {
				log.Printf("  ❌ ERROR: Failed to rename original: %v", err)
				errorCount++
				continue
			}

			// Step 2: Create subvolume with quota
			if err := qm.CreateQuotaDir(share.Path, quotaGB); err != nil {
				log.Printf("  ❌ ERROR: Failed to create subvolume: %v", err)
				// Rollback
				os.Rename(backupPath, share.Path)
				errorCount++
				continue
			}

			// Step 3: Copy data from backup to new subvolume
			log.Printf("  📦 Copying data...")
			cmd := exec.Command("cp", "-a", backupPath+"/.", share.Path+"/")
			if output, err := cmd.CombinedOutput(); err != nil {
				log.Printf("  ❌ ERROR: Failed to copy data: %v\nOutput: %s", err, output)
				// Rollback
				qm.RemoveQuotaDir(share.Path)
				os.Rename(backupPath, share.Path)
				errorCount++
				continue
			}

			// Step 4: Verify ownership
			cmd = exec.Command("sudo", "chown", "-R", fmt.Sprintf("%s:%s", user.Username, user.Username), share.Path)
			if err := cmd.Run(); err != nil {
				log.Printf("  ⚠️  WARNING: Failed to set ownership: %v", err)
			}

			// Step 5: Keep backup for safety
			log.Printf("  💾 Original data backed up to: %s", backupPath)
			log.Printf("  ⚠️  IMPORTANT: Verify the share works, then manually remove: sudo rm -rf %s", backupPath)

			log.Printf("  ✅ Migration completed successfully")
		}

		successCount++
	}

	// Summary
	log.Printf("\n" + strings.Repeat("=", 50))
	log.Printf("Migration Summary:")
	log.Printf("  ✅ Success: %d", successCount)
	log.Printf("  ⏭️  Skipped: %d", skipCount)
	log.Printf("  ❌ Errors: %d", errorCount)
	log.Printf(strings.Repeat("=", 50))

	if *dryRun {
		log.Printf("\n⚠️  This was a DRY RUN. No changes were made.")
		log.Printf("Run without --dry-run to perform the migration.")
	} else if successCount > 0 {
		log.Printf("\n✅ Migration completed!")
		log.Printf("\nNext steps:")
		log.Printf("  1. Test SMB access to verify shares work correctly")
		log.Printf("  2. Once verified, remove backup directories:")
		log.Printf("     sudo rm -rf %s/*/.backup", sharesDir)
	}
}

// isSubvolume checks if a path is a Btrfs subvolume
func isSubvolume(path string) bool {
	cmd := exec.Command("btrfs", "subvolume", "show", path)
	return cmd.Run() == nil
}
