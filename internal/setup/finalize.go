// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package setup

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/juste-un-gars/anemone/internal/crypto"
	"github.com/juste-un-gars/anemone/internal/database"
	"github.com/juste-un-gars/anemone/internal/smb"
	syncpkg "github.com/juste-un-gars/anemone/internal/sync"
	"github.com/juste-un-gars/anemone/internal/syncauth"
	"github.com/juste-un-gars/anemone/internal/users"
)

// FinalizeOptions contains options for finalizing the setup
type FinalizeOptions struct {
	DataDir       string
	SharesDir     string
	IncomingDir   string
	AdminUsername string
	AdminPassword string
	AdminEmail    string
	ServerName    string
	Language      string
}

// FinalizeResult contains the results of the finalization
type FinalizeResult struct {
	AdminUserID   int    `json:"admin_user_id"`
	AdminUsername string `json:"admin_username"`
	EncryptionKey string `json:"encryption_key"`
	SyncPassword  string `json:"sync_password"`
	EnvFile       string `json:"env_file"`
}

// FinalizeSetup completes the Anemone setup
func FinalizeSetup(opts FinalizeOptions) (*FinalizeResult, error) {
	result := &FinalizeResult{}

	// 1. Initialize database
	dbPath := filepath.Join(opts.DataDir, "db", "anemone.db")
	dbDir := filepath.Dir(dbPath)

	// Create database directory with sudo (needed for restricted paths like /srv)
	if err := createDirectoryWithSudo(dbDir); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := database.Init(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// 2. Run migrations
	if err := database.Migrate(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// 3. Update sudoers if using custom data directory (MUST be done before creating users)
	defaultDataDir := "/srv/anemone"
	if opts.DataDir != defaultDataDir {
		if err := updateSudoersDataDir(defaultDataDir, opts.DataDir); err != nil {
			// Non-fatal: log warning but continue
			log.Printf("Warning: Failed to update sudoers for custom path: %v", err)
		}
	}

	// 4. Generate master key for encrypting user keys
	masterKeyBytes := make([]byte, 32)
	if _, err := rand.Read(masterKeyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate master key: %w", err)
	}
	masterKey := base64.StdEncoding.EncodeToString(masterKeyBytes)

	// 5. Create admin user
	language := opts.Language
	if language == "" {
		language = "fr"
	}
	adminUser, encryptionKey, err := users.CreateFirstAdmin(
		db,
		opts.AdminUsername,
		opts.AdminPassword,
		opts.AdminEmail,
		masterKey,
		language,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}
	result.AdminUserID = adminUser.ID
	result.AdminUsername = adminUser.Username
	result.EncryptionKey = encryptionKey

	// 6. Generate sync authentication password
	syncPasswordBytes := make([]byte, 24) // 24 bytes = 192 bits
	if _, err := rand.Read(syncPasswordBytes); err != nil {
		return nil, fmt.Errorf("failed to generate sync password: %w", err)
	}
	syncPassword := base64.URLEncoding.EncodeToString(syncPasswordBytes)
	result.SyncPassword = syncPassword

	// 7. Store sync password hash in database
	if err := syncauth.SetSyncAuthPassword(db, syncPassword); err != nil {
		return nil, fmt.Errorf("failed to store sync password: %w", err)
	}

	// 8. Save system configuration
	serverName := opts.ServerName
	if serverName == "" {
		serverName = "Anemone NAS"
	}
	if err := saveSystemConfig(db, masterKey, serverName, language); err != nil {
		return nil, fmt.Errorf("failed to save system config: %w", err)
	}

	// 9. Write environment file for service (always in /etc/anemone/)
	envFile := "/etc/anemone/anemone.env"
	if err := writeEnvFile(envFile, opts); err != nil {
		return nil, fmt.Errorf("failed to write environment file: %w", err)
	}
	result.EnvFile = envFile

	return result, nil
}

// saveSystemConfig saves essential system configuration
func saveSystemConfig(db *sql.DB, masterKey, serverName, language string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	configs := map[string]string{
		"master_key":      masterKey,
		"language":        language,
		"nas_name":        serverName,
		"timezone":        "Europe/Paris",
		"setup_completed": "true",
	}

	for key, value := range configs {
		_, err = tx.Exec(
			"INSERT OR REPLACE INTO system_config (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)",
			key, value,
		)
		if err != nil {
			return fmt.Errorf("failed to save config %s: %w", key, err)
		}
	}

	return tx.Commit()
}

// writeEnvFile writes the environment configuration file to /etc/anemone/
func writeEnvFile(path string, opts FinalizeOptions) error {
	content := fmt.Sprintf(`# Anemone NAS Configuration
# Generated during setup - do not edit manually

ANEMONE_DATA_DIR=%s
`, opts.DataDir)

	// Add optional overrides
	if opts.SharesDir != "" && opts.SharesDir != filepath.Join(opts.DataDir, "shares") {
		content += fmt.Sprintf("ANEMONE_SHARES_DIR=%s\n", opts.SharesDir)
	}

	if opts.IncomingDir != "" && opts.IncomingDir != filepath.Join(opts.DataDir, "backups", "incoming") {
		content += fmt.Sprintf("ANEMONE_INCOMING_DIR=%s\n", opts.IncomingDir)
	}

	// Create /etc/anemone/ directory if it doesn't exist
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Try without sudo first
		if err := os.MkdirAll(dir, 0755); err != nil {
			// Try with sudo
			cmd := exec.Command("sudo", "mkdir", "-p", dir)
			if err := cmd.Run(); err != nil {
				currentUser := os.Getenv("USER")
				if currentUser == "" {
					currentUser = os.Getenv("LOGNAME")
				}
				if currentUser == "" {
					currentUser = "YOUR_USER"
				}
				return fmt.Errorf("cannot create %s. Please create it manually:\n\nsudo mkdir -p %s\nsudo chown %s:%s %s", dir, dir, currentUser, currentUser, dir)
			}
		}
	}

	// Try to write directly first
	if err := os.WriteFile(path, []byte(content), 0644); err == nil {
		return nil
	}

	// Fall back to sudo tee
	cmd := exec.Command("sudo", "tee", path)
	cmd.Stdin = strings.NewReader(content)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot write to %s. Please create the directory manually:\n\nsudo mkdir -p %s\nsudo chown $(whoami):$(whoami) %s\n\nThen retry the setup.", path, dir, dir)
	}

	return nil
}

// updateSudoersDataDir updates /etc/sudoers.d/anemone to use the custom data directory
func updateSudoersDataDir(oldDir, newDir string) error {
	sudoersFile := "/etc/sudoers.d/anemone"

	// Read current sudoers file using sudo (file has 440 permissions)
	cmd := exec.Command("sudo", "cat", sudoersFile)
	content, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to read sudoers file: %w", err)
	}

	// Replace old path with new path
	newContent := strings.ReplaceAll(string(content), oldDir, newDir)

	// Update the comment to reflect the new path
	newContent = strings.Replace(newContent, "# Data directory: "+newDir, "# Data directory: "+newDir+" (updated by setup wizard)", 1)

	// Write back using sudo tee (sudoers file requires root)
	cmd = exec.Command("sudo", "tee", sudoersFile)
	cmd.Stdin = strings.NewReader(newContent)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to write sudoers file: %s", string(output))
	}

	return nil
}

// FinalizeImport completes the setup for an imported existing installation
// This is used when recovering an existing Anemone installation (e.g., after OS reinstall)
// Unlike FinalizeSetup, this does NOT:
// - Create a new database (uses existing)
// - Generate new keys (uses existing)
// - Create admin user (uses existing)
func FinalizeImport(opts FinalizeOptions) error {
	// 1. Verify the database exists
	dbPath := filepath.Join(opts.DataDir, "db", "anemone.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("database not found at %s - this doesn't appear to be a valid Anemone installation", dbPath)
	}

	// 2. Update sudoers if using custom data directory
	defaultDataDir := "/srv/anemone"
	if opts.DataDir != defaultDataDir {
		if err := updateSudoersDataDir(defaultDataDir, opts.DataDir); err != nil {
			// Non-fatal: log warning but continue
			log.Printf("Warning: Failed to update sudoers for custom path: %v", err)
		}
	}

	// 3. Write environment file for service (always in /etc/anemone/)
	envFile := "/etc/anemone/anemone.env"
	if err := writeEnvFile(envFile, opts); err != nil {
		return fmt.Errorf("failed to write environment file: %w", err)
	}

	// 4. Open existing database to regenerate Samba config
	db, err := database.Init(dbPath)
	if err != nil {
		log.Printf("Warning: Failed to open database for Samba config: %v", err)
	} else {
		defer db.Close()

		// Get server name from DB
		serverName, _ := syncpkg.GetServerName(db)
		if serverName == "" {
			serverName = "Anemone NAS"
		}

		// Determine shares directory
		sharesDir := opts.SharesDir
		if sharesDir == "" {
			sharesDir = filepath.Join(opts.DataDir, "shares")
		}

		// Regenerate Samba configuration
		smbCfg := &smb.Config{
			ConfigPath: filepath.Join(opts.DataDir, "smb", "smb.conf"),
			WorkGroup:  "WORKGROUP",
			ServerName: serverName,
			SharesDir:  sharesDir,
		}

		if err := smb.GenerateConfig(db, smbCfg); err != nil {
			log.Printf("Warning: Failed to generate Samba config: %v", err)
		} else {
			log.Printf("Generated Samba configuration")

			// Reload Samba
			if err := smb.ReloadConfig(); err != nil {
				log.Printf("Warning: Failed to reload Samba: %v", err)
			} else {
				log.Printf("Reloaded Samba configuration")
			}
		}

		// 5. Recreate system users and Samba accounts
		// After OS reinstall, users don't exist - we need to recreate them
		masterKey, err := getMasterKey(db)
		if err != nil {
			log.Printf("Warning: Failed to get master key: %v", err)
		} else {
			// Get all activated users with their encrypted passwords
			userRows, err := db.Query(`
				SELECT id, username, password_encrypted, activated_at
				FROM users
				WHERE activated_at IS NOT NULL AND password_encrypted IS NOT NULL
			`)
			if err != nil {
				log.Printf("Warning: Failed to query users: %v", err)
			} else {
				defer userRows.Close()
				for userRows.Next() {
					var userID int
					var username string
					var passwordEncrypted []byte
					var activatedAt *string
					if err := userRows.Scan(&userID, &username, &passwordEncrypted, &activatedAt); err != nil {
						log.Printf("Warning: Failed to scan user: %v", err)
						continue
					}

					// Decrypt password
					if len(passwordEncrypted) > 0 {
						plainPassword, err := crypto.DecryptPassword(passwordEncrypted, masterKey)
						if err != nil {
							log.Printf("Warning: Failed to decrypt password for user %s: %v", username, err)
							continue
						}

						// Create system user and set SMB password
						if err := smb.AddSMBUser(username, plainPassword); err != nil {
							log.Printf("Warning: Failed to create SMB user %s: %v", username, err)
						} else {
							log.Printf("Created system user and SMB account for: %s", username)
						}
					}
				}
			}
		}

		// 6. Fix ownership of share directories
		shareRows, err := db.Query("SELECT u.username, s.path FROM users u JOIN shares s ON u.id = s.user_id")
		if err != nil {
			log.Printf("Warning: Failed to query shares for ownership fix: %v", err)
		} else {
			defer shareRows.Close()
			for shareRows.Next() {
				var username, sharePath string
				if err := shareRows.Scan(&username, &sharePath); err != nil {
					continue
				}
				// Fix ownership of the share directory
				cmd := exec.Command("sudo", "chown", "-R", username+":"+username, sharePath)
				if err := cmd.Run(); err != nil {
					log.Printf("Warning: Failed to fix ownership for %s: %v", sharePath, err)
				} else {
					log.Printf("Fixed ownership for share: %s -> %s", sharePath, username)
				}
			}
		}
	}

	log.Printf("Import finalized: data directory = %s", opts.DataDir)
	return nil
}

// getMasterKey retrieves the master encryption key from the database
func getMasterKey(db *sql.DB) (string, error) {
	var masterKey string
	err := db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to get master_key: %w", err)
	}
	return masterKey, nil
}

// GenerateSystemdOverride generates a systemd override file for Anemone
func GenerateSystemdOverride(dataDir string) (string, error) {
	content := fmt.Sprintf(`[Service]
Environment="ANEMONE_DATA_DIR=%s"
`, dataDir)

	return content, nil
}
