// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package rclone

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/users"
)

// SyncResult contains the result of a sync operation
type SyncResult struct {
	FilesTransferred int
	BytesTransferred int64
	Errors           []string
}

// IsRcloneInstalled checks if rclone is available on the system
func IsRcloneInstalled() bool {
	_, err := exec.LookPath("rclone")
	return err == nil
}

// GetRcloneVersion returns the installed rclone version
func GetRcloneVersion() (string, error) {
	cmd := exec.Command("rclone", "version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get rclone version: %w", err)
	}

	// Parse first line: "rclone v1.67.0"
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		parts := strings.Fields(lines[0])
		if len(parts) >= 2 {
			return parts[1], nil
		}
	}
	return "unknown", nil
}

// TestConnection tests the SFTP connection for a backup configuration
func TestConnection(backup *RcloneBackup) error {
	// Build rclone remote string
	remote := buildRemoteString(backup)

	// Use rclone lsd to test connection (list directories)
	args := []string{"lsd", remote + backup.RemotePath, "--max-depth", "1"}
	cmd := exec.Command("rclone", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("connection test failed: %s", stderr.String())
	}

	return nil
}

// Sync synchronizes all users' backup directories to the SFTP server
func Sync(db *sql.DB, backup *RcloneBackup, dataDir string) (*SyncResult, error) {
	if !IsRcloneInstalled() {
		return nil, fmt.Errorf("rclone is not installed")
	}

	// Update status to running
	UpdateSyncStatus(db, backup.ID, "running", "", 0, 0)

	result := &SyncResult{}

	// Get all users
	allUsers, err := users.GetAllUsers(db)
	if err != nil {
		UpdateSyncStatus(db, backup.ID, "error", err.Error(), 0, 0)
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	// Build rclone remote string
	remote := buildRemoteString(backup)

	// Sync each user's backup directory
	for _, user := range allUsers {
		// Source: user's backup directory
		sourceDir := filepath.Join(dataDir, "shares", user.Username, "backup")

		// Check if backup directory exists
		if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
			logger.Info("📂 Rclone: No backup directory for user %s, skipping", user.Username)
			continue
		}

		// Destination: remote path / username
		destPath := filepath.Join(backup.RemotePath, "backup", user.Username)

		logger.Info("📤 Rclone: Syncing %s to %s%s", user.Username, backup.SFTPHost, destPath)

		// Run rclone sync
		userResult, err := runRcloneSync(remote, sourceDir, destPath)
		if err != nil {
			errMsg := fmt.Sprintf("user %s: %v", user.Username, err)
			result.Errors = append(result.Errors, errMsg)
			logger.Info("⚠️  Rclone: Sync failed for %s: %v", user.Username, err)
			continue
		}

		result.FilesTransferred += userResult.FilesTransferred
		result.BytesTransferred += userResult.BytesTransferred

		logger.Info("✅ Rclone: Synced %s - %d files, %s",
			user.Username, userResult.FilesTransferred, FormatBytes(userResult.BytesTransferred))
	}

	// Update final status
	if len(result.Errors) > 0 {
		errMsg := strings.Join(result.Errors, "; ")
		UpdateSyncStatus(db, backup.ID, "error", errMsg, result.FilesTransferred, result.BytesTransferred)
	} else {
		UpdateSyncStatus(db, backup.ID, "success", "", result.FilesTransferred, result.BytesTransferred)
	}

	logger.Info("📦 Rclone backup completed: %d files, %s",
		result.FilesTransferred, FormatBytes(result.BytesTransferred))

	return result, nil
}

// SyncUser synchronizes a single user's backup directory
func SyncUser(db *sql.DB, backup *RcloneBackup, dataDir string, username string) (*SyncResult, error) {
	if !IsRcloneInstalled() {
		return nil, fmt.Errorf("rclone is not installed")
	}

	result := &SyncResult{}

	// Source: user's backup directory
	sourceDir := filepath.Join(dataDir, "shares", username, "backup")

	// Check if backup directory exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("backup directory does not exist: %s", sourceDir)
	}

	// Build rclone remote string
	remote := buildRemoteString(backup)

	// Destination: remote path / username
	destPath := filepath.Join(backup.RemotePath, "backup", username)

	logger.Info("📤 Rclone: Syncing %s to %s%s", username, backup.SFTPHost, destPath)

	// Run rclone sync
	userResult, err := runRcloneSync(remote, sourceDir, destPath)
	if err != nil {
		return nil, fmt.Errorf("sync failed: %w", err)
	}

	result.FilesTransferred = userResult.FilesTransferred
	result.BytesTransferred = userResult.BytesTransferred

	return result, nil
}

// buildRemoteString builds the rclone SFTP remote connection string
func buildRemoteString(backup *RcloneBackup) string {
	// Format: :sftp,host=HOST,user=USER,port=PORT[,key_file=KEY][,pass=PASS]:
	parts := []string{
		fmt.Sprintf("host=%s", backup.SFTPHost),
		fmt.Sprintf("user=%s", backup.SFTPUser),
		fmt.Sprintf("port=%d", backup.SFTPPort),
	}

	if backup.SFTPKeyPath != "" {
		parts = append(parts, fmt.Sprintf("key_file=%s", backup.SFTPKeyPath))
	}

	// Note: For password auth, rclone expects the password to be obscured
	// For simplicity, we prioritize key-based auth. Password can be added later.

	return fmt.Sprintf(":sftp,%s:", strings.Join(parts, ","))
}

// runRcloneSync executes rclone sync command and parses the output
func runRcloneSync(remote, sourceDir, destPath string) (*SyncResult, error) {
	result := &SyncResult{}

	// Build rclone command
	// --stats-one-line: compact output for parsing
	// --progress: show progress (useful for logs)
	// --transfers 4: parallel transfers
	// --checkers 8: parallel checkers
	args := []string{
		"sync",
		sourceDir,
		remote + destPath,
		"--stats-one-line",
		"--stats", "0",
		"--transfers", "4",
		"--checkers", "8",
		"-v",
	}

	cmd := exec.Command("rclone", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Parse output for stats (even on error, we might have partial results)
	output := stdout.String() + stderr.String()
	result.FilesTransferred, result.BytesTransferred = parseRcloneStats(output)

	if err != nil {
		// Check if it's a real error or just warnings
		if strings.Contains(stderr.String(), "error") || strings.Contains(stderr.String(), "failed") {
			return result, fmt.Errorf("rclone error: %s", stderr.String())
		}
	}

	return result, nil
}

// parseRcloneStats parses rclone output for transfer statistics
func parseRcloneStats(output string) (files int, bytes int64) {
	// Look for patterns like:
	// "Transferred:   1.234 MiB / 1.234 MiB, 100%, 0 B/s, ETA -"
	// "Transferred:        5 / 5, 100%"

	// Pattern for bytes transferred
	bytesPattern := regexp.MustCompile(`Transferred:\s*([\d.]+)\s*([KMGTi]*B)`)
	if match := bytesPattern.FindStringSubmatch(output); len(match) >= 3 {
		value, _ := strconv.ParseFloat(match[1], 64)
		unit := match[2]
		bytes = convertToBytes(value, unit)
	}

	// Pattern for files transferred
	filesPattern := regexp.MustCompile(`Transferred:\s*(\d+)\s*/\s*\d+`)
	if match := filesPattern.FindStringSubmatch(output); len(match) >= 2 {
		files, _ = strconv.Atoi(match[1])
	}

	// Alternative: look for "Checks:" line
	checksPattern := regexp.MustCompile(`Checks:\s*(\d+)`)
	if match := checksPattern.FindStringSubmatch(output); len(match) >= 2 && files == 0 {
		files, _ = strconv.Atoi(match[1])
	}

	return files, bytes
}

// convertToBytes converts a value with unit to bytes
func convertToBytes(value float64, unit string) int64 {
	multipliers := map[string]float64{
		"B":   1,
		"KiB": 1024,
		"KB":  1000,
		"MiB": 1024 * 1024,
		"MB":  1000 * 1000,
		"GiB": 1024 * 1024 * 1024,
		"GB":  1000 * 1000 * 1000,
		"TiB": 1024 * 1024 * 1024 * 1024,
		"TB":  1000 * 1000 * 1000 * 1000,
	}

	if mult, ok := multipliers[unit]; ok {
		return int64(value * mult)
	}
	return int64(value)
}

// ListRemoteDir lists the contents of a remote directory (for testing/debugging)
func ListRemoteDir(backup *RcloneBackup, path string) ([]string, error) {
	remote := buildRemoteString(backup)

	args := []string{"lsd", remote + path}
	cmd := exec.Command("rclone", args...)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list remote: %w", err)
	}

	var dirs []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			// Parse: "-1 2024-01-01 00:00:00        -1 dirname"
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				dirs = append(dirs, parts[len(parts)-1])
			}
		}
	}

	return dirs, nil
}
