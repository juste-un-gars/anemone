// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package shares

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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
func Create(db *sql.DB, share *Share) error {
	// Ensure the share directory exists
	if err := os.MkdirAll(share.Path, 0755); err != nil {
		return fmt.Errorf("failed to create share directory: %w", err)
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
	return Create(db, share)
}

// GetSharePath returns the full path to a share
func (s *Share) GetSharePath() string {
	return s.Path
}

// GetSizeMB calculates the current size of a share in MB
func (s *Share) GetSizeMB() (int64, error) {
	var size int64
	err := filepath.Walk(s.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
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
