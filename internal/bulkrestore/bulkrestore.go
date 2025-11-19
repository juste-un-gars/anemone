// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package bulkrestore

import (
	"bytes"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/juste-un-gars/anemone/internal/crypto"
	"github.com/juste-un-gars/anemone/internal/peers"
	"github.com/juste-un-gars/anemone/internal/users"
)

// FileEntry represents a file in the manifest
type FileEntry struct {
	Path         string `json:"path"`
	Size         int64  `json:"size"`
	ModifiedTime int64  `json:"modified_time"`
	IsDir        bool   `json:"is_dir"`
	Checksum     string `json:"checksum"`
}

// Manifest represents the backup manifest
type Manifest struct {
	Files map[string]FileEntry `json:"files"` // Map indexed by file path
}

// RestoreProgress holds the restoration progress
type RestoreProgress struct {
	TotalFiles     int
	ProcessedFiles int
	TotalBytes     int64
	ProcessedBytes int64
	CurrentFile    string
	Errors         []string
}

// BulkRestoreFromPeer restores all files from a peer backup to local shares
func BulkRestoreFromPeer(db *sql.DB, userID int, peerID int, shareName string, sourceServer string, dataDir string, progressChan chan<- RestoreProgress) error {
	progress := RestoreProgress{}

	// Get user info
	user, err := users.GetByID(db, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Get master key from system_config
	var masterKey string
	err = db.QueryRow("SELECT value FROM system_config WHERE key = 'master_key'").Scan(&masterKey)
	if err != nil {
		return fmt.Errorf("failed to get master key: %w", err)
	}

	// Decrypt user's encryption key
	encryptedKey := string(user.EncryptionKeyEncrypted)
	userKey, err := crypto.DecryptKey(encryptedKey, masterKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt user key: %w", err)
	}

	// Get peer info
	peer, err := peers.GetByID(db, peerID)
	if err != nil {
		return fmt.Errorf("failed to get peer: %w", err)
	}

	// Download and decrypt manifest
	baseURL := fmt.Sprintf("https://%s:%d", peer.Address, peer.Port)
	manifestURL := fmt.Sprintf("%s/api/sync/download-encrypted-manifest?user_id=%d&share_name=%s&source_server=%s", baseURL, userID, shareName, sourceServer)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("GET", manifestURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if peer has password
	if peer.Password != nil && *peer.Password != "" {
		req.Header.Set("X-Sync-Password", *peer.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download manifest: status %d", resp.StatusCode)
	}

	// Read encrypted manifest
	encryptedManifest, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// Decrypt manifest
	decryptedManifest, err := decryptData(encryptedManifest, userKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt manifest: %w", err)
	}

	// Parse manifest
	var manifest Manifest
	if err := json.Unmarshal(decryptedManifest, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Calculate total size and files
	progress.TotalFiles = len(manifest.Files)
	for _, file := range manifest.Files {
		if !file.IsDir {
			progress.TotalBytes += file.Size
		}
	}

	if progressChan != nil {
		progressChan <- progress
	}

	log.Printf("Starting bulk restore for user %d from peer %s: %d files, %d bytes",
		userID, peer.Name, progress.TotalFiles, progress.TotalBytes)

	// Determine target directory based on share name
	// For standard shares (backup/data), use the share name directly
	// For custom shares, verify they exist in the database
	var targetDir string
	var share struct {
		Path string
	}
	err = db.QueryRow("SELECT path FROM shares WHERE user_id = ? AND name = ?", userID, shareName).Scan(&share.Path)
	if err != nil {
		return fmt.Errorf("share not found for user %d: %s (%w)", userID, shareName, err)
	}

	// Use the path from database
	targetDir = share.Path

	// Restore each file
	// IMPORTANT: Iterate over map to get both the key (file path) and value (file entry)
	// The manifest stores files with the path as the key, but the Path field in FileEntry may be empty
	for filePath, file := range manifest.Files {
		progress.CurrentFile = filePath
		progress.ProcessedFiles++

		if progressChan != nil {
			progressChan <- progress
		}

		if file.IsDir {
			// Create directory
			dirPath := filepath.Join(targetDir, filePath)
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				errMsg := fmt.Sprintf("Failed to create directory %s: %v", filePath, err)
				progress.Errors = append(progress.Errors, errMsg)
				log.Printf("Error: %s", errMsg)
				continue
			}

			// Set directory ownership to user
			if err := setOwnership(dirPath, user.Username); err != nil {
				log.Printf("Warning: Failed to set ownership for directory %s: %v", filePath, err)
			}
		} else {
			// Download and decrypt file
			fileURL := fmt.Sprintf("%s/api/sync/download-encrypted-file?user_id=%d&share_name=%s&path=%s&source_server=%s",
				baseURL, userID, shareName, buildURL(filePath), buildURL(sourceServer))

			req, err := http.NewRequest("GET", fileURL, nil)
			if err != nil {
				errMsg := fmt.Sprintf("Failed to create request for %s: %v", filePath, err)
				progress.Errors = append(progress.Errors, errMsg)
				log.Printf("Error: %s", errMsg)
				continue
			}

			if peer.Password != nil && *peer.Password != "" {
				req.Header.Set("X-Sync-Password", *peer.Password)
			}

			resp, err := client.Do(req)
			if err != nil {
				errMsg := fmt.Sprintf("Failed to download %s: %v", filePath, err)
				progress.Errors = append(progress.Errors, errMsg)
				log.Printf("Error: %s", errMsg)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				errMsg := fmt.Sprintf("Failed to download %s: status %d", filePath, resp.StatusCode)
				progress.Errors = append(progress.Errors, errMsg)
				log.Printf("Error: %s", errMsg)
				continue
			}

			// Read encrypted file
			encryptedData, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				errMsg := fmt.Sprintf("Failed to read %s: %v", filePath, err)
				progress.Errors = append(progress.Errors, errMsg)
				log.Printf("Error: %s", errMsg)
				continue
			}

			// Decrypt file
			decryptedData, err := decryptData(encryptedData, userKey)
			if err != nil {
				errMsg := fmt.Sprintf("Failed to decrypt %s: %v", filePath, err)
				progress.Errors = append(progress.Errors, errMsg)
				log.Printf("Error: %s", errMsg)
				continue
			}

			// Write file to disk
			targetFilePath := filepath.Join(targetDir, filePath)

			// Ensure parent directory exists
			parentDir := filepath.Dir(targetFilePath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				errMsg := fmt.Sprintf("Failed to create parent directory for %s: %v", filePath, err)
				progress.Errors = append(progress.Errors, errMsg)
				log.Printf("Error: %s", errMsg)
				continue
			}

			// Set parent directory ownership (important for subdirectories)
			if err := setOwnership(parentDir, user.Username); err != nil {
				log.Printf("Warning: Failed to set ownership for parent directory of %s: %v", filePath, err)
			}

			if err := os.WriteFile(targetFilePath, decryptedData, 0644); err != nil {
				errMsg := fmt.Sprintf("Failed to write %s: %v", filePath, err)
				progress.Errors = append(progress.Errors, errMsg)
				log.Printf("Error: %s", errMsg)
				continue
			}

			// Set file ownership to user
			if err := setOwnership(targetFilePath, user.Username); err != nil {
				log.Printf("Warning: Failed to set ownership for %s: %v", filePath, err)
			}

			progress.ProcessedBytes += file.Size
			log.Printf("Restored file: %s (%d bytes)", filePath, file.Size)
		}

		if progressChan != nil {
			progressChan <- progress
		}
	}

	log.Printf("Bulk restore completed for user %d: %d files, %d bytes, %d errors",
		userID, progress.ProcessedFiles, progress.ProcessedBytes, len(progress.Errors))

	return nil
}

// buildURL properly encodes the path for URL
func buildURL(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		parts[i] = strings.ReplaceAll(part, " ", "%20")
	}
	return strings.Join(parts, "/")
}

// decryptData is a helper function to decrypt a []byte using crypto.DecryptStream
func decryptData(encryptedData []byte, encryptionKey string) ([]byte, error) {
	reader := bytes.NewReader(encryptedData)
	writer := &bytes.Buffer{}

	if err := crypto.DecryptStream(reader, writer, encryptionKey); err != nil {
		return nil, err
	}

	return writer.Bytes(), nil
}

// setOwnership changes the ownership of a file or directory to the specified user
func setOwnership(path, username string) error {
	// Lookup user to get UID and GID
	u, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("user lookup failed: %w", err)
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return fmt.Errorf("invalid UID: %w", err)
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return fmt.Errorf("invalid GID: %w", err)
	}

	// Change ownership (requires root privileges)
	if err := os.Chown(path, uid, gid); err != nil {
		return fmt.Errorf("chown failed: %w", err)
	}

	return nil
}
