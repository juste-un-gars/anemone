// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package users

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/juste-un-gars/anemone/internal/crypto"
)

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
func CreateFirstAdmin(db *sql.DB, username, password, email, masterKey string) (*User, string, error) {
	// Hash password
	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
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
			username, password_hash, email,
			encryption_key_hash, encryption_key_encrypted,
			is_admin, quota_total_gb, quota_backup_gb,
			created_at, activated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, username, passwordHash, email, keyHash, encryptedKey, true, 100, 50, now, now)

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
		    encryption_key_hash = ?,
		    encryption_key_encrypted = ?,
		    activated_at = ?
		WHERE id = ?
	`, passwordHash, keyHash, encryptedKey, now, userID)

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

// DeleteUser deletes a user and their associated data
// This includes: database entries, SMB shares, system user, and all files on disk
func DeleteUser(db *sql.DB, userID int) error {
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

	// Delete files from disk for each share
	for _, share := range shares {
		if err := os.RemoveAll(share.Path); err != nil {
			// Log error but continue (don't fail if directory doesn't exist)
			fmt.Printf("Warning: failed to delete share directory %s: %v\n", share.Path, err)
		}
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

	return nil
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
func ChangePassword(db *sql.DB, userID int, oldPassword, newPassword string) error {
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

	// Update password in database
	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", newPasswordHash, userID)
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
func ResetPassword(db *sql.DB, userID int, username, newPassword string) error {
	// Validate new password length
	if len(newPassword) < 8 {
		return fmt.Errorf("new password must be at least 8 characters")
	}

	// Hash new password
	newPasswordHash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password in database
	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", newPasswordHash, userID)
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
