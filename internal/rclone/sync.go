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
func TestConnection(backup *RcloneBackup, dataDir string) error {
	// Build rclone remote string
	remote := buildRemoteString(backup, dataDir)

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

	// Sync each user's backup directory
	for _, user := range allUsers {
		// Source: user's backup directory
		sourceDir := filepath.Join(dataDir, "shares", user.Username, "backup")

		// Check if backup directory exists
		if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
			logger.Info(fmt.Sprintf("Rclone: No backup directory for user %s, skipping", user.Username))
			continue
		}

		// Destination: remote path / username (with optional crypt wrapping)
		destPath := filepath.Join(backup.RemotePath, "backup", user.Username)
		dest := buildDestination(backup, dataDir, destPath)

		logger.Info(fmt.Sprintf("Rclone: Syncing %s to %s%s", user.Username, backup.DisplayHost(), destPath))

		// Run rclone sync
		userResult, err := runRcloneSyncDest(sourceDir, dest)
		if err != nil {
			errMsg := fmt.Sprintf("user %s: %v", user.Username, err)
			result.Errors = append(result.Errors, errMsg)
			logger.Info(fmt.Sprintf("Rclone: Sync failed for %s: %v", user.Username, err))
			continue
		}

		result.FilesTransferred += userResult.FilesTransferred
		result.BytesTransferred += userResult.BytesTransferred

		logger.Info(fmt.Sprintf("Rclone: Synced %s - %d files, %s",
			user.Username, userResult.FilesTransferred, FormatBytes(userResult.BytesTransferred)))
	}

	// Update final status
	if len(result.Errors) > 0 {
		errMsg := strings.Join(result.Errors, "; ")
		UpdateSyncStatus(db, backup.ID, "error", errMsg, result.FilesTransferred, result.BytesTransferred)
	} else {
		UpdateSyncStatus(db, backup.ID, "success", "", result.FilesTransferred, result.BytesTransferred)
	}

	logger.Info(fmt.Sprintf("Rclone backup completed: %d files, %s",
		result.FilesTransferred, FormatBytes(result.BytesTransferred)))

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

	// Destination: remote path / username (with optional crypt wrapping)
	destPath := filepath.Join(backup.RemotePath, "backup", username)
	dest := buildDestination(backup, dataDir, destPath)

	logger.Info(fmt.Sprintf("Rclone: Syncing %s to %s%s", username, backup.DisplayHost(), destPath))

	// Run rclone sync
	userResult, err := runRcloneSyncDest(sourceDir, dest)
	if err != nil {
		return nil, fmt.Errorf("sync failed: %w", err)
	}

	result.FilesTransferred = userResult.FilesTransferred
	result.BytesTransferred = userResult.BytesTransferred

	return result, nil
}

// buildRemoteString builds the rclone remote connection string based on provider type.
func buildRemoteString(backup *RcloneBackup, dataDir string) string {
	switch backup.ProviderType {
	case ProviderS3:
		return buildS3Remote(backup)
	case ProviderWebDAV:
		return buildWebDAVRemote(backup)
	case ProviderRemote:
		return buildNamedRemote(backup)
	default:
		return buildSFTPRemote(backup, dataDir)
	}
}

// buildSFTPRemote builds an rclone SFTP remote string.
func buildSFTPRemote(backup *RcloneBackup, dataDir string) string {
	parts := []string{
		fmt.Sprintf("host=%s", backup.SFTPHost),
		fmt.Sprintf("user=%s", backup.SFTPUser),
		fmt.Sprintf("port=%d", backup.SFTPPort),
	}

	if backup.SFTPKeyPath != "" {
		resolvedKeyPath := ResolveKeyPath(backup.SFTPKeyPath, dataDir)
		parts = append(parts, fmt.Sprintf("key_file=%s", resolvedKeyPath))
	}

	return fmt.Sprintf(":sftp,%s:", strings.Join(parts, ","))
}

// buildS3Remote builds an rclone S3 remote string.
func buildS3Remote(backup *RcloneBackup) string {
	cfg := backup.ProviderConfig
	parts := []string{
		fmt.Sprintf("provider=%s", cfgGet(cfg, "s3_provider", "Other")),
	}

	if ep := cfgGet(cfg, "endpoint", ""); ep != "" {
		parts = append(parts, fmt.Sprintf("endpoint=%s", quoteValue(ep)))
	}
	if region := cfgGet(cfg, "region", ""); region != "" {
		parts = append(parts, fmt.Sprintf("region=%s", region))
	}
	if ak := cfgGet(cfg, "access_key_id", ""); ak != "" {
		parts = append(parts, fmt.Sprintf("access_key_id=%s", ak))
	}
	if sk := cfgGet(cfg, "secret_access_key", ""); sk != "" {
		parts = append(parts, fmt.Sprintf("secret_access_key=%s", quoteValue(sk)))
	}

	return fmt.Sprintf(":s3,%s:", strings.Join(parts, ","))
}

// quoteValue quotes an rclone connection string value if it contains : or ,
func quoteValue(v string) string {
	if strings.ContainsAny(v, ":,") {
		return "'" + strings.ReplaceAll(v, "'", "''") + "'"
	}
	return v
}

// buildWebDAVRemote builds an rclone WebDAV remote string.
func buildWebDAVRemote(backup *RcloneBackup) string {
	cfg := backup.ProviderConfig
	parts := []string{}

	if url := cfgGet(cfg, "url", ""); url != "" {
		parts = append(parts, fmt.Sprintf("url=%s", quoteValue(url)))
	}
	if vendor := cfgGet(cfg, "vendor", ""); vendor != "" {
		parts = append(parts, fmt.Sprintf("vendor=%s", vendor))
	}
	if user := cfgGet(cfg, "user", ""); user != "" {
		parts = append(parts, fmt.Sprintf("user=%s", quoteValue(user)))
	}
	if pass := cfgGet(cfg, "pass", ""); pass != "" {
		parts = append(parts, fmt.Sprintf("pass=%s", quoteValue(pass)))
	}

	return fmt.Sprintf(":webdav,%s:", strings.Join(parts, ","))
}

// buildNamedRemote builds an rclone named remote string.
func buildNamedRemote(backup *RcloneBackup) string {
	name := cfgGet(backup.ProviderConfig, "remote_name", "")
	if name == "" {
		return ":"
	}
	// Named remotes use "remotename:" format
	return name + ":"
}

// cfgGet returns a value from config map with a default fallback.
func cfgGet(cfg map[string]string, key, defaultVal string) string {
	if v, ok := cfg[key]; ok && v != "" {
		return v
	}
	return defaultVal
}

// buildDestination builds the full rclone destination string, wrapping with crypt if enabled.
func buildDestination(backup *RcloneBackup, dataDir, destPath string) string {
	remote := buildRemoteString(backup, dataDir)
	cryptPass := cfgGet(backup.ProviderConfig, "crypt_password", "")
	if cryptPass == "" {
		return remote + destPath
	}
	// Wrap with crypt: the inner remote+path becomes the crypt backend
	return fmt.Sprintf(":crypt,remote=%s%s,password=%s,filename_encryption=standard:",
		remote, destPath, cryptPass)
}

// runRcloneSyncDest executes rclone sync command with a pre-built destination and parses the output.
func runRcloneSyncDest(sourceDir, dest string) (*SyncResult, error) {
	result := &SyncResult{}

	args := []string{
		"sync",
		sourceDir,
		dest,
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
func ListRemoteDir(backup *RcloneBackup, dataDir string, path string) ([]string, error) {
	remote := buildRemoteString(backup, dataDir)

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
