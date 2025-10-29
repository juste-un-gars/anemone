// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package smb

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/juste-un-gars/anemone/internal/shares"
)

// Config holds SMB server configuration
type Config struct {
	ConfigPath  string // Path to smb.conf
	WorkGroup   string // Workgroup name
	ServerName  string // Server name
	SharesDir   string // Base directory for shares
}

const smbConfigTemplate = `[global]
   workgroup = {{.WorkGroup}}
   server string = {{.ServerName}}
   security = user
   map to guest = never
   log file = /var/log/samba/log.%m
   max log size = 1000

   # Performance optimizations
   socket options = TCP_NODELAY IPTOS_LOWDELAY SO_RCVBUF=524288 SO_SNDBUF=524288
   read raw = yes
   write raw = yes
   oplocks = yes
   max xmit = 65535
   dead time = 15
   getwd cache = yes

   # Security
   client min protocol = SMB2
   client max protocol = SMB3
   server min protocol = SMB2
   server max protocol = SMB3

{{range .Shares}}
[{{.Name}}]
   path = {{.Path}}
   valid users = {{.Username}}
   read only = no
   browseable = yes
   create mask = 0664
   directory mask = 0775
   force user = {{.Username}}
   force group = {{.Username}}
{{end}}
`

// ShareConfig represents a share in the SMB configuration
type ShareConfig struct {
	Name     string
	Username string
	Path     string
}

// GenerateConfig generates the smb.conf file from database shares
func GenerateConfig(db *sql.DB, cfg *Config) error {
	// Get all shares from database
	allShares, err := shares.GetAll(db)
	if err != nil {
		return fmt.Errorf("failed to get shares: %w", err)
	}

	// Get usernames for each share
	shareConfigs := []ShareConfig{}
	for _, share := range allShares {
		// Get username from user_id
		var username string
		err := db.QueryRow("SELECT username FROM users WHERE id = ?", share.UserID).Scan(&username)
		if err != nil {
			return fmt.Errorf("failed to get username for share %d: %w", share.ID, err)
		}

		shareConfigs = append(shareConfigs, ShareConfig{
			Name:     share.Name,
			Username: username,
			Path:     share.Path,
		})
	}

	// Prepare template data
	data := struct {
		WorkGroup  string
		ServerName string
		Shares     []ShareConfig
	}{
		WorkGroup:  cfg.WorkGroup,
		ServerName: cfg.ServerName,
		Shares:     shareConfigs,
	}

	// Parse and execute template
	tmpl, err := template.New("smb").Parse(smbConfigTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Ensure config directory exists
	configDir := filepath.Dir(cfg.ConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create config file
	file, err := os.Create(cfg.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// AddSMBUser adds a user to Samba (creates system user and SMB password)
func AddSMBUser(username, password string) error {
	// Check if user already exists
	_, err := exec.Command("id", username).Output()
	if err != nil {
		// User doesn't exist, create it (requires sudo)
		cmd := exec.Command("sudo", "useradd", "-M", "-s", "/usr/sbin/nologin", username)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create system user: %w", err)
		}
	}

	// Set SMB password using smbpasswd (requires sudo)
	cmd := exec.Command("sudo", "smbpasswd", "-a", "-s", username)
	cmd.Stdin = strings.NewReader(password + "\n" + password + "\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set SMB password: %w (output: %s)", err, string(output))
	}

	// Enable the SMB user (requires sudo)
	cmd = exec.Command("sudo", "smbpasswd", "-e", username)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable SMB user: %w", err)
	}

	return nil
}

// RemoveSMBUser removes a user from Samba
func RemoveSMBUser(username string) error {
	// Remove from Samba
	cmd := exec.Command("smbpasswd", "-x", username)
	if err := cmd.Run(); err != nil {
		// Ignore error if user doesn't exist in Samba
		return nil
	}

	// Remove system user
	cmd = exec.Command("userdel", username)
	if err := cmd.Run(); err != nil {
		// Ignore error if user doesn't exist
		return nil
	}

	return nil
}

// ReloadConfig reloads the Samba configuration
func ReloadConfig() error {
	cmd := exec.Command("sudo", "systemctl", "reload", "smbd")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload smbd: %w", err)
	}
	return nil
}

// RestartService restarts the Samba service
func RestartService() error {
	cmd := exec.Command("systemctl", "restart", "smbd")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart smbd: %w", err)
	}
	return nil
}

// CheckSambaInstalled checks if Samba is installed
func CheckSambaInstalled() bool {
	_, err := exec.LookPath("smbd")
	return err == nil
}

// GetServiceStatus returns the status of the Samba service
func GetServiceStatus() (string, error) {
	cmd := exec.Command("systemctl", "is-active", "smbd")
	output, err := cmd.Output()
	status := strings.TrimSpace(string(output))
	if err != nil {
		// Service might be inactive
		return status, nil
	}
	return status, nil
}
