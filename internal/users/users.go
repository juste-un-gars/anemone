// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package users handles user account management including creation, authentication,
// and Linux system user integration.
package users

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/juste-un-gars/anemone/internal/btrfs"
	"github.com/juste-un-gars/anemone/internal/crypto"
	"github.com/juste-un-gars/anemone/internal/smb"
)

// usernameRegex validates username format to prevent command injection
// Only allows: letters (a-z, A-Z), numbers (0-9), underscore (_), and hyphen (-)
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateUsername checks if a username has a valid format
// Returns an error if the username is invalid
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if len(username) < 2 {
		return fmt.Errorf("username must be at least 2 characters")
	}
	if len(username) > 32 {
		return fmt.Errorf("username must not exceed 32 characters")
	}
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("username can only contain letters, numbers, underscore (_) and hyphen (-)")
	}
	return nil
}

// User represents a user account
type User struct {
	ID                     int
	Username               string
	PasswordHash           string
	Email                  string
	EncryptionKeyHash      string
	EncryptionKeyEncrypted string
	IsAdmin                bool
	QuotaTotalGB           int
	QuotaBackupGB          int
	Language               string
	CreatedAt              time.Time
	ActivatedAt            *time.Time
	LastLogin              *time.Time
}

// CreateFirstAdmin creates the first administrator user during setup
func CreateFirstAdmin(db *sql.DB, username, password, email, masterKey, language string) (*User, string, error) {
	// Validate username format (prevent command injection)
	if err := ValidateUsername(username); err != nil {
		return nil, "", fmt.Errorf("invalid username: %w", err)
	}

	// Hash password
	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Encrypt password for SMB restoration
	passwordEncrypted, err := crypto.EncryptPassword(password, masterKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encrypt password: %w", err)
	}

	// Generate encryption key
	encryptionKey, err := crypto.GenerateEncryptionKey()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Hash the key (for verification)
	keyHash := crypto.HashKey(encryptionKey)

	// Encrypt the key with master key
	encryptedKey, err := crypto.EncryptKey(encryptionKey, masterKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encrypt key: %w", err)
	}

	// Insert user
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO users (
			username, password_hash, password_encrypted, email,
			encryption_key_hash, encryption_key_encrypted,
			is_admin, quota_total_gb, quota_backup_gb,
			created_at, activated_at, language
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, username, passwordHash, passwordEncrypted, email, keyHash, encryptedKey, true, 100, 50, now, now, language)

	if err != nil {
		return nil, "", fmt.Errorf("failed to insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user ID: %w", err)
	}

	user := &User{
		ID:                     int(id),
		Username:               username,
		PasswordHash:           passwordHash,
		Email:                  email,
		EncryptionKeyHash:      keyHash,
		EncryptionKeyEncrypted: encryptedKey,
		IsAdmin:                true,
		QuotaTotalGB:           100,
		QuotaBackupGB:          50,
		CreatedAt:              now,
		ActivatedAt:            &now,
	}

	// Return user and plaintext encryption key (only time it's available)
	return user, encryptionKey, nil
}

// GetByUsername retrieves a user by username
func GetByUsername(db *sql.DB, username string) (*User, error) {
	user := &User{}
	var activatedAt, lastLogin sql.NullTime

	err := db.QueryRow(`
		SELECT id, username, password_hash, email,
		       encryption_key_hash, encryption_key_encrypted,
		       is_admin, quota_total_gb, quota_backup_gb, language,
		       created_at, activated_at, last_login
		FROM users WHERE username = ?
	`, username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Email,
		&user.EncryptionKeyHash, &user.EncryptionKeyEncrypted,
		&user.IsAdmin, &user.QuotaTotalGB, &user.QuotaBackupGB, &user.Language,
		&user.CreatedAt, &activatedAt, &lastLogin,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if activatedAt.Valid {
		user.ActivatedAt = &activatedAt.Time
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	return user, nil
}

// CheckPassword verifies a user's password
func (u *User) CheckPassword(password string) bool {
	return crypto.CheckPassword(password, u.PasswordHash)
}

// UpdateLastLogin updates the user's last login timestamp
func (u *User) UpdateLastLogin(db *sql.DB) error {
	now := time.Now()
	_, err := db.Exec("UPDATE users SET last_login = ? WHERE id = ?", now, u.ID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	u.LastLogin = &now
	return nil
}

// CreatePendingUser creates a new user account (pending activation)
// This is used by admins to create new users who will activate their account later
func CreatePendingUser(db *sql.DB, username, email string, isAdmin bool, quotaTotalGB, quotaBackupGB int) (*User, error) {
	now := time.Now()

	// Create user with placeholder password (will be set during activation)
	result, err := db.Exec(`
		INSERT INTO users (
			username, password_hash, email,
			encryption_key_hash, encryption_key_encrypted,
			is_admin, quota_total_gb, quota_backup_gb,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, username, "", email, "", "", isAdmin, quotaTotalGB, quotaBackupGB, now)

	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	user := &User{
		ID:            int(id),
		Username:      username,
		Email:         email,
		IsAdmin:       isAdmin,
		QuotaTotalGB:  quotaTotalGB,
		QuotaBackupGB: quotaBackupGB,
		CreatedAt:     now,
	}

	return user, nil
}

// ActivateUser activates a pending user account with password and encryption key
func ActivateUser(db *sql.DB, userID int, password, masterKey string) (string, error) {
	// Hash password
	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Encrypt password for SMB restoration
	passwordEncrypted, err := crypto.EncryptPassword(password, masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt password: %w", err)
	}

	// Generate encryption key
	encryptionKey, err := crypto.GenerateEncryptionKey()
	if err != nil {
		return "", fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Hash the key (for verification)
	keyHash := crypto.HashKey(encryptionKey)

	// Encrypt the key with master key
	encryptedKey, err := crypto.EncryptKey(encryptionKey, masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt key: %w", err)
	}

	// Update user
	now := time.Now()
	_, err = db.Exec(`
		UPDATE users
		SET password_hash = ?,
		    password_encrypted = ?,
		    encryption_key_hash = ?,
		    encryption_key_encrypted = ?,
		    activated_at = ?
		WHERE id = ?
	`, passwordHash, passwordEncrypted, keyHash, encryptedKey, now, userID)

	if err != nil {
		return "", fmt.Errorf("failed to activate user: %w", err)
	}

	// Return plaintext encryption key (only time it's available)
	return encryptionKey, nil
}

// GetByID retrieves a user by ID
func GetByID(db *sql.DB, userID int) (*User, error) {
	user := &User{}
	var activatedAt, lastLogin sql.NullTime

	err := db.QueryRow(`
		SELECT id, username, password_hash, email,
		       encryption_key_hash, encryption_key_encrypted,
		       is_admin, quota_total_gb, quota_backup_gb, language,
		       created_at, activated_at, last_login
		FROM users WHERE id = ?
	`, userID).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Email,
		&user.EncryptionKeyHash, &user.EncryptionKeyEncrypted,
		&user.IsAdmin, &user.QuotaTotalGB, &user.QuotaBackupGB, &user.Language,
		&user.CreatedAt, &activatedAt, &lastLogin,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if activatedAt.Valid {
		user.ActivatedAt = &activatedAt.Time
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	return user, nil
}

// GetAllUsers retrieves all users
func GetAllUsers(db *sql.DB) ([]*User, error) {
	rows, err := db.Query(`
		SELECT id, username, password_hash, email,
		       encryption_key_hash, encryption_key_encrypted,
		       is_admin, quota_total_gb, quota_backup_gb, language,
		       created_at, activated_at, last_login
		FROM users
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		var activatedAt, lastLogin sql.NullTime

		err := rows.Scan(
			&user.ID, &user.Username, &user.PasswordHash, &user.Email,
			&user.EncryptionKeyHash, &user.EncryptionKeyEncrypted,
			&user.IsAdmin, &user.QuotaTotalGB, &user.QuotaBackupGB, &user.Language,
			&user.CreatedAt, &activatedAt, &lastLogin,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if activatedAt.Valid {
			user.ActivatedAt = &activatedAt.Time
		}
		if lastLogin.Valid {
			user.LastLogin = &lastLogin.Time
		}

		users = append(users, user)
	}

	return users, nil
}


// removeShareDirectory removes a share directory, handling Btrfs subvolumes properly
func removeShareDirectory(path string) error {
	// Check if it's a Btrfs subvolume
	if btrfs.IsSubvolume(path) {
		// Use btrfs subvolume delete for proper cleanup
		if output, err := btrfs.DeleteSubvolume(path); err != nil {
			return fmt.Errorf("failed to delete subvolume: %w\nOutput: %s", err, output)
		}
		return nil
	}

	// Regular directory, use standard removal
	return os.RemoveAll(path)
}

// DeleteUser deletes a user and their associated data
// This includes: database entries, SMB shares, system user, and all files on disk
func DeleteUser(db *sql.DB, userID int, dataDir string) error {
	// Get user info before deleting
	user, err := GetByID(db, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Get user's shares to delete files from disk
	var shares []struct {
		ID   int
		Path string
	}
	rows, err := db.Query("SELECT id, path FROM shares WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to query shares: %w", err)
	}
	for rows.Next() {
		var share struct {
			ID   int
			Path string
		}
		if err := rows.Scan(&share.ID, &share.Path); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan share: %w", err)
		}
		shares = append(shares, share)
	}
	rows.Close()

	// Delete user backups on all enabled peers (best-effort, don't fail if peer is down)
	deleteUserBackupsOnPeers(db, userID)

	// Start transaction for database cleanup
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete user from database (cascading deletes will handle related data)
	_, err = tx.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Delete files from disk for each share (handles Btrfs subvolumes properly)
	sharesDir := filepath.Join(dataDir, "shares")
	for _, share := range shares {
		if err := removeShareDirectory(share.Path); err != nil {
			fmt.Printf("Warning: failed to delete share directory %s: %v\n", share.Path, err)
		}
	}

	// Delete user's parent directory (e.g., /srv/anemone/shares/username/)
	// Use sudo because files may belong to the deleted system user
	userDir := filepath.Join(sharesDir, user.Username)
	rmCmd := exec.Command("sudo", "rm", "-rf", userDir)
	if output, err := rmCmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: failed to delete user directory %s: %v\nOutput: %s\n", userDir, err, output)
	}

	// Remove SMB user
	cmd := exec.Command("sudo", "smbpasswd", "-x", user.Username)
	if err := cmd.Run(); err != nil {
		// Log error but don't fail (user might not exist in SMB)
		fmt.Printf("Warning: failed to remove SMB user %s: %v\n", user.Username, err)
	}

	// Remove system user
	cmd = exec.Command("sudo", "userdel", user.Username)
	if err := cmd.Run(); err != nil {
		// Log error but don't fail (user might not exist)
		fmt.Printf("Warning: failed to remove system user %s: %v\n", user.Username, err)
	}

	// Regenerate SMB configuration (removes deleted user's shares from smb.conf)
	smbCfg := &smb.Config{
		ConfigPath: filepath.Join(dataDir, "smb", "smb.conf"),
		WorkGroup:  "WORKGROUP",
		ServerName: "Anemone NAS",
		SharesDir:  sharesDir,
		DfreePath:  "/usr/local/bin/anemone-dfree-wrapper.sh",
	}

	if err := smb.GenerateConfig(db, smbCfg); err != nil {
		fmt.Printf("Warning: failed to regenerate SMB config: %v\n", err)
	}

	// Copy config to /etc/samba/smb.conf and reload Samba
	copyCmd := exec.Command("sudo", "cp", smbCfg.ConfigPath, "/etc/samba/smb.conf")
	if err := copyCmd.Run(); err != nil {
		fmt.Printf("Warning: failed to copy SMB config: %v\n", err)
	}

	// Reload Samba service (try both smb and smbd for multi-distro support)
	reloadCmd := exec.Command("sudo", "systemctl", "reload", "smb")
	if err := reloadCmd.Run(); err != nil {
		reloadCmd = exec.Command("sudo", "systemctl", "reload", "smbd")
		if err := reloadCmd.Run(); err != nil {
			fmt.Printf("Warning: failed to reload Samba: %v\n", err)
		}
	}

	return nil
}

// deleteUserBackupsOnPeers deletes user backups on all enabled peers
// This is called when a user is deleted to comply with GDPR right to be forgotten
func deleteUserBackupsOnPeers(db *sql.DB, userID int) {
	// Get server name for source_server parameter
	var serverName string
	err := db.QueryRow("SELECT value FROM system_config WHERE key = 'nas_name'").Scan(&serverName)
	if err != nil {
		log.Printf("⚠️  Warning: failed to get server name: %v", err)
		serverName = "unknown"
	}

	// Get master key for password decryption
	var masterKey string
	err = db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		log.Printf("⚠️  Warning: failed to get master key: %v", err)
		return
	}

	// Get all enabled peers
	rows, err := db.Query("SELECT id, name, address, port, password FROM peers WHERE enabled = 1")
	if err != nil {
		log.Printf("⚠️  Warning: failed to query peers: %v", err)
		return
	}
	defer rows.Close()

	// Create HTTP client that accepts self-signed certs
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}

	// Delete user backup on each peer
	for rows.Next() {
		var peerID int
		var peerName, peerAddress string
		var peerPort int
		var encryptedPassword []byte

		if err := rows.Scan(&peerID, &peerName, &peerAddress, &peerPort, &encryptedPassword); err != nil {
			log.Printf("⚠️  Warning: failed to scan peer: %v", err)
			continue
		}

		// Build delete URL
		deleteURL := fmt.Sprintf("https://%s:%d/api/sync/delete-user-backup?source_server=%s&user_id=%d",
			peerAddress, peerPort, serverName, userID)

		req, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
		if err != nil {
			log.Printf("⚠️  Warning: failed to create delete request for peer %s: %v", peerName, err)
			continue
		}

		// Decrypt and add sync authentication header with the PEER's password
		if len(encryptedPassword) > 0 {
			peerPassword, err := crypto.DecryptPassword(encryptedPassword, masterKey)
			if err != nil {
				log.Printf("⚠️  Warning: failed to decrypt password for peer %s: %v", peerName, err)
				continue
			}
			req.Header.Set("X-Sync-Password", peerPassword)
		}

		// Send request
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("⚠️  Warning: failed to delete user %d backup on peer %s: %v", userID, peerName, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("⚠️  Warning: failed to delete user %d backup on peer %s: status %d", userID, peerName, resp.StatusCode)
			continue
		}

		log.Printf("✅ Successfully deleted user %d backup on peer %s", userID, peerName)
	}
}

// IsActivated checks if the user account is activated
func (u *User) IsActivated() bool {
	return u.ActivatedAt != nil
}

// UpdateUserLanguage updates the language preference for a user
func UpdateUserLanguage(db *sql.DB, userID int, language string) error {
	// Validate language code
	if language != "fr" && language != "en" {
		return fmt.Errorf("invalid language code: %s (must be 'fr' or 'en')", language)
	}

	_, err := db.Exec("UPDATE users SET language = ? WHERE id = ?", language, userID)
	if err != nil {
		return fmt.Errorf("failed to update user language: %w", err)
	}

	return nil
}

// ChangePassword changes a user's password (both in DB and SMB)
// IMPORTANT: The encryption key remains unchanged - password is only for authentication
func ChangePassword(db *sql.DB, userID int, oldPassword, newPassword, masterKey string) error {
	// Validate new password length
	if len(newPassword) < 8 {
		return fmt.Errorf("new password must be at least 8 characters")
	}

	// Get user from database
	user, err := GetByID(db, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify old password
	if !crypto.CheckPassword(oldPassword, user.PasswordHash) {
		return fmt.Errorf("incorrect current password")
	}

	// Hash new password
	newPasswordHash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Encrypt password for SMB restoration
	passwordEncrypted, err := crypto.EncryptPassword(newPassword, masterKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	// Update password in database
	_, err = db.Exec("UPDATE users SET password_hash = ?, password_encrypted = ? WHERE id = ?", newPasswordHash, passwordEncrypted, userID)
	if err != nil {
		return fmt.Errorf("failed to update password in database: %w", err)
	}

	// Update SMB password
	// Use smbpasswd with stdin to change password
	cmd := exec.Command("sudo", "smbpasswd", "-s", user.Username)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start smbpasswd: %w", err)
	}

	// Write new password twice (smbpasswd asks for password twice)
	_, err = fmt.Fprintf(stdin, "%s\n%s\n", newPassword, newPassword)
	stdin.Close()
	if err != nil {
		return fmt.Errorf("failed to write to smbpasswd: %w", err)
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed to update SMB password: %w", err)
	}

	fmt.Printf("Password changed successfully for user: %s\n", user.Username)
	return nil
}

// ResetPassword resets a user's password (used by admin for password reset)
// It updates both the database and SMB password, without verifying the old password
func ResetPassword(db *sql.DB, userID int, username, newPassword, masterKey string) error {
	// Validate new password length
	if len(newPassword) < 8 {
		return fmt.Errorf("new password must be at least 8 characters")
	}

	// Hash new password
	newPasswordHash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Encrypt password for SMB restoration
	passwordEncrypted, err := crypto.EncryptPassword(newPassword, masterKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	// Update password in database
	_, err = db.Exec("UPDATE users SET password_hash = ?, password_encrypted = ? WHERE id = ?", newPasswordHash, passwordEncrypted, userID)
	if err != nil {
		return fmt.Errorf("failed to update password in database: %w", err)
	}

	// Update SMB password
	// Use smbpasswd with stdin to change password
	cmd := exec.Command("sudo", "smbpasswd", "-s", username)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start smbpasswd: %w", err)
	}

	// Write new password twice (smbpasswd asks for password twice)
	_, err = fmt.Fprintf(stdin, "%s\n%s\n", newPassword, newPassword)
	stdin.Close()
	if err != nil {
		return fmt.Errorf("failed to write to smbpasswd: %w", err)
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed to update SMB password: %w", err)
	}

	fmt.Printf("Password reset successfully for user: %s\n", username)
	return nil
}
