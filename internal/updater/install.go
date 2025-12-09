// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package updater

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// PerformAutoUpdate launches the auto-update script in the background
// The script will continue running even after the server restarts
// targetVersion should be the version to install (e.g., "0.9.2-beta")
// Note: Requires sudo NOPASSWD for systemctl restart anemone
func PerformAutoUpdate(targetVersion string) error {
	// Get the project root directory (where the anemone binary is)
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	projectDir := filepath.Dir(execPath)

	// Path to the auto-update script
	scriptPath := filepath.Join(projectDir, "scripts", "auto-update.sh")

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("auto-update script not found at %s", scriptPath)
	}

	// Prepare the command to run the script in background with nohup
	// The script will log to /tmp/anemone-update.log
	// Arguments: scriptPath, targetVersion
	cmd := exec.Command("nohup", scriptPath, targetVersion)
	cmd.Dir = projectDir

	// Redirect stdout and stderr to /dev/null (script handles its own logging)
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Start the script in the background
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start auto-update script: %w", err)
	}

	log.Printf("🚀 Auto-update process started (PID: %d)", cmd.Process.Pid)
	log.Printf("📝 Update log: /tmp/anemone-update.log")

	// Don't wait for the command to finish - let it run independently
	// The script will handle the git pull, build, and restart

	return nil
}

// GetUpdateLogPath returns the path to the auto-update log file
func GetUpdateLogPath() string {
	return "/tmp/anemone-update.log"
}
