// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/juste-un-gars/anemone/internal/logger"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/juste-un-gars/anemone/internal/btrfs"
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

	logger.Info("🪸 Anemone Share Migration Tool")
	logger.Info("Data dir", "data_dir", *dataDir)
	logger.Info("Database", "db_path", dbPath)
	logger.Info("Shares dir", "shares_dir", sharesDir)
	logger.Info("Dry run", "dry_run", *dryRun)
	logger.Info("")

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize quota manager
	qm, err := quota.NewQuotaManager(sharesDir)
	if err != nil {
		logger.Error("Failed to initialize quota manager", "error", err)
		os.Exit(1)
	}

	// Get all shares
	allShares, err := shares.GetAll(db)
	if err != nil {
		logger.Error("Failed to get shares", "error", err)
		os.Exit(1)
	}

	if len(allShares) == 0 {
		logger.Info("No shares found in database")
		return
	}

	logger.Info("Found shares to process", "all_shares", len(allShares))

	successCount := 0
	skipCount := 0
	errorCount := 0

	for _, share := range allShares {
		logger.Info("\n---")
		logger.Info("Processing share", "name", share.Name, "id", share.ID, "user_id", share.UserID)
		logger.Info("Path", "path", share.Path)

		// Get user info for quota
		user, err := users.GetByID(db, share.UserID)
		if err != nil {
			logger.Info("ERROR: Failed to get user info", "error", err)
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

		logger.Info("User: (Quota: GB)", "username", user.Username, "quota_gb", quotaGB)

		// Check if path exists
		info, err := os.Stat(share.Path)
		if err != nil {
			if os.IsNotExist(err) {
				logger.Info("  ⚠️  SKIP: Path does not exist")
				skipCount++
				continue
			}
			logger.Info("ERROR: Failed to stat path", "error", err)
			errorCount++
			continue
		}

		if !info.IsDir() {
			logger.Info("  ❌ ERROR: Path is not a directory")
			errorCount++
			continue
		}

		// Check if already a subvolume
		isSubvol := btrfs.IsSubvolume(share.Path)
		if isSubvol && !*force {
			logger.Info("  ⏭️  SKIP: Already a subvolume")
			skipCount++
			continue
		}

		if *dryRun {
			if isSubvol {
				logger.Info("[DRY RUN] Would update quota to GB", "quota_gb", quotaGB)
			} else {
				logger.Info("[DRY RUN] Would migrate to subvolume with GB quota", "quota_gb", quotaGB)
			}
			successCount++
			continue
		}

		// Perform migration
		if isSubvol {
			// Just update quota
			logger.Info("Updating quota to GB...", "quota_gb", quotaGB)
			if err := qm.UpdateQuota(share.Path, quotaGB); err != nil {
				logger.Info("ERROR: Failed to update quota", "error", err)
				errorCount++
				continue
			}
			logger.Info("  ✅ Quota updated successfully")
		} else {
			// Full migration: regular dir → subvolume with quota
			logger.Info("Migrating to subvolume with GB quota...", "quota_gb", quotaGB)

			backupPath := share.Path + ".backup"

			// Step 1: Rename original directory to .backup
			if err := os.Rename(share.Path, backupPath); err != nil {
				logger.Info("ERROR: Failed to rename original", "error", err)
				errorCount++
				continue
			}

			// Step 2: Create subvolume with quota (no owner - will be set by cp -a)
			if err := qm.CreateQuotaDir(share.Path, quotaGB, ""); err != nil {
				logger.Info("ERROR: Failed to create subvolume", "error", err)
				// Rollback
				os.Rename(backupPath, share.Path)
				errorCount++
				continue
			}

			// Step 3: Copy data from backup to new subvolume
			logger.Info("  📦 Copying data...")
			cmd := exec.Command("cp", "-a", backupPath+"/.", share.Path+"/")
			if output, err := cmd.CombinedOutput(); err != nil {
				logger.Info("ERROR: Failed to copy data: \nOutput", "error", err, "output", output)
				// Rollback
				qm.RemoveQuotaDir(share.Path)
				os.Rename(backupPath, share.Path)
				errorCount++
				continue
			}

			// Step 4: Verify ownership
			cmd = exec.Command("sudo", "/usr/bin/chown", "-R", fmt.Sprintf("%s:%s", user.Username, user.Username), share.Path)
			if err := cmd.Run(); err != nil {
				logger.Info("WARNING: Failed to set ownership", "error", err)
			}

			// Step 5: Keep backup for safety
			logger.Info("Original data backed up to", "backup_path", backupPath)
			logger.Info("IMPORTANT: Verify the share works, then manually remove: sudo rm -rf", "backup_path", backupPath)

			logger.Info("  ✅ Migration completed successfully")
		}

		successCount++
	}

	// Summary
	logger.Info("\n" + strings.Repeat("=", 50))
	logger.Info("Migration Summary:")
	logger.Info("Success", "success_count", successCount)
	logger.Info("Skipped", "skip_count", skipCount)
	logger.Info("Errors", "error_count", errorCount)
	logger.Info(strings.Repeat("=", 50))

	if *dryRun {
		logger.Info("\n⚠️  This was a DRY RUN. No changes were made.")
		logger.Info("Run without --dry-run to perform the migration.")
	} else if successCount > 0 {
		logger.Info("\n✅ Migration completed!")
		logger.Info("\nNext steps:")
		logger.Info("  1. Test SMB access to verify shares work correctly")
		logger.Info("  2. Once verified, remove backup directories:")
		logger.Info("sudo rm -rf /*/.backup", "shares_dir", sharesDir)
	}
}

