// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package shares

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Share represents a file share (SMB, NFS, etc.)
type Share struct {
	ID          int
	UserID      int
	Name        string
	Path        string
	Protocol    string // "smb", "nfs", etc.
	SyncEnabled bool
	CreatedAt   time.Time
}

// Create creates a new share for a user
// The directory at share.Path must already exist (caller's responsibility to create with or without quota)
func Create(db *sql.DB, share *Share, username string) error {
	// Ensure the share directory exists
	if _, err := os.Stat(share.Path); os.IsNotExist(err) {
		// Create if it doesn't exist (fallback for compatibility)
		if err := os.MkdirAll(share.Path, 0775); err != nil {
			return fmt.Errorf("failed to create share directory: %w", err)
		}
	}

	// Pre-create .trash directory with correct permissions (755)
	// This prevents Samba VFS recycle from creating it with restrictive permissions (700)
	// The VFS module ignores force_directory_mode for internally-created directories
	if username != "" {
		trashDir := filepath.Join(share.Path, ".trash", username)
		// Use sudo mkdir because share.Path may already be owned by username
		cmd := exec.Command("sudo", "/usr/bin/mkdir", "-p", trashDir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create trash directory: %w", err)
		}

		// Set correct permissions (755) on .trash directories
		trashRoot := filepath.Join(share.Path, ".trash")
		chmodCmd := exec.Command("sudo", "/usr/bin/chmod", "-R", "755", trashRoot)
		if err := chmodCmd.Run(); err != nil {
			return fmt.Errorf("failed to set trash directory permissions: %w", err)
		}

		// Set ownership of .trash to the share user
		chownCmd := exec.Command("sudo", "/usr/bin/chown", "-R", fmt.Sprintf("%s:%s", username, username), trashRoot)
		if err := chownCmd.Run(); err != nil {
			return fmt.Errorf("failed to set trash directory ownership: %w", err)
		}
	}

	query := `INSERT INTO shares (user_id, name, path, protocol, sync_enabled, created_at)
	          VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`
	result, err := db.Exec(query, share.UserID, share.Name, share.Path, share.Protocol, share.SyncEnabled)
	if err != nil {
		return fmt.Errorf("failed to create share: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get share ID: %w", err)
	}
	share.ID = int(id)

	// Change owner to the share user (requires sudo)
	// This allows the SMB user to access their own directories
	if username != "" {
		cmd := exec.Command("sudo", "/usr/bin/chown", "-R", fmt.Sprintf("%s:%s", username, username), share.Path)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set directory ownership: %w", err)
		}
	}

	// Configure SELinux for Samba (only on RHEL/Fedora)
	if err := configureSELinux(share.Path); err != nil {
		// Log error but don't fail - SELinux might not be installed
		fmt.Printf("Warning: SELinux configuration failed: %v\n", err)
	}

	return nil
}

// configureSELinux sets the appropriate SELinux context for Samba shares
func configureSELinux(sharePath string) error {
	// Check if SELinux is available
	if _, err := exec.LookPath("restorecon"); err != nil {
		// SELinux not available (Debian/Ubuntu)
		return nil
	}

	// Apply Samba context to the share directory
	cmd := exec.Command("sudo", "restorecon", "-Rv", sharePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply SELinux context: %w", err)
	}

	return nil
}

// GetByID retrieves a share by its ID
func GetByID(db *sql.DB, id int) (*Share, error) {
	share := &Share{}
	query := `SELECT id, user_id, name, path, protocol, sync_enabled, created_at
	          FROM shares WHERE id = ?`
	err := db.QueryRow(query, id).Scan(
		&share.ID, &share.UserID, &share.Name, &share.Path,
		&share.Protocol, &share.SyncEnabled, &share.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("share not found")
		}
		return nil, fmt.Errorf("failed to get share: %w", err)
	}
	return share, nil
}

// GetByUser retrieves all shares for a specific user
func GetByUser(db *sql.DB, userID int) ([]*Share, error) {
	query := `SELECT id, user_id, name, path, protocol, sync_enabled, created_at
	          FROM shares WHERE user_id = ? ORDER BY created_at DESC`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query shares: %w", err)
	}
	defer rows.Close()

	var shares []*Share
	for rows.Next() {
		share := &Share{}
		err := rows.Scan(
			&share.ID, &share.UserID, &share.Name, &share.Path,
			&share.Protocol, &share.SyncEnabled, &share.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan share: %w", err)
		}
		shares = append(shares, share)
	}
	return shares, nil
}

// GetAll retrieves all shares (admin function)
func GetAll(db *sql.DB) ([]*Share, error) {
	query := `SELECT id, user_id, name, path, protocol, sync_enabled, created_at
	          FROM shares ORDER BY created_at DESC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query shares: %w", err)
	}
	defer rows.Close()

	var shares []*Share
	for rows.Next() {
		share := &Share{}
		err := rows.Scan(
			&share.ID, &share.UserID, &share.Name, &share.Path,
			&share.Protocol, &share.SyncEnabled, &share.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan share: %w", err)
		}
		shares = append(shares, share)
	}
	return shares, nil
}

// Update updates a share
func Update(db *sql.DB, share *Share) error {
	query := `UPDATE shares SET name = ?, path = ?, protocol = ?, sync_enabled = ?
	          WHERE id = ?`
	_, err := db.Exec(query, share.Name, share.Path, share.Protocol, share.SyncEnabled, share.ID)
	if err != nil {
		return fmt.Errorf("failed to update share: %w", err)
	}
	return nil
}

// Delete deletes a share
func Delete(db *sql.DB, id int) error {
	query := `DELETE FROM shares WHERE id = ?`
	_, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete share: %w", err)
	}
	return nil
}

// CreateDefaultShare creates the default backup share for a user
func CreateDefaultShare(db *sql.DB, userID int, username, sharesDir string) error {
	sharePath := filepath.Join(sharesDir, username, "backup")
	share := &Share{
		UserID:      userID,
		Name:        "backup",
		Path:        sharePath,
		Protocol:    "smb",
		SyncEnabled: true,
	}
	return Create(db, share, username)
}

// GetSharePath returns the full path to a share
func (s *Share) GetSharePath() string {
	return s.Path
}

// GetSizeMB calculates the current size of a share in MB
// Uses Btrfs quotas if available (much faster), falls back to filepath.Walk
func (s *Share) GetSizeMB() (int64, error) {
	// Try to use Btrfs quotas first (fast and accurate)
	if qm, err := newQuotaManager(s.Path); err == nil {
		if used, _, err := qm.GetUsage(s.Path); err == nil {
			return used / (1024 * 1024), nil // Convert bytes to MB
		}
	}

	// Fallback to filepath.Walk for non-quota filesystems
	var size int64
	err := filepath.Walk(s.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip permission errors (e.g., .trash directories)
			if os.IsPermission(err) {
				return nil
			}
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to calculate share size: %w", err)
	}
	return size / (1024 * 1024), nil // Convert to MB
}

// newQuotaManager creates a quota manager (local helper)
func newQuotaManager(path string) (quotaManager, error) {
	// Detect filesystem type
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, err
	}

	// Only Btrfs supported for now
	if stat.Type == 0x9123683E { // BTRFS_SUPER_MAGIC
		return &btrfsQuotaHelper{}, nil
	}

	return nil, fmt.Errorf("filesystem quota not supported")
}

// quotaManager interface for size calculation
type quotaManager interface {
	GetUsage(path string) (used, limit int64, err error)
}

// btrfsQuotaHelper implements quotaManager for Btrfs
type btrfsQuotaHelper struct{}

func (h *btrfsQuotaHelper) GetUsage(path string) (int64, int64, error) {
	// Get subvolume ID
	cmd := exec.Command("btrfs", "subvolume", "show", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, err
	}

	// Parse subvolume ID
	var subvolID string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Subvolume ID:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				subvolID = parts[2]
				break
			}
		}
	}

	if subvolID == "" {
		return 0, 0, fmt.Errorf("failed to parse subvolume ID")
	}

	// Get qgroup info
	cmd = exec.Command("btrfs", "qgroup", "show", "-r", "--raw", path)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return 0, 0, err
	}

	// Parse qgroup output for exclusive usage
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "0/"+subvolID) {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				// Format: qgroupid rfer excl [max_rfer max_excl]
				used, _ := strconv.ParseInt(fields[2], 10, 64) // excl (exclusive bytes)
				limit := int64(0)
				if len(fields) >= 5 {
					limit, _ = strconv.ParseInt(fields[4], 10, 64)
				}
				return used, limit, nil
			}
		}
	}

	return 0, 0, fmt.Errorf("qgroup not found")
}
