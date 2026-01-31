// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package rclone

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// SSHKeyName is the default name for the generated SSH key
	SSHKeyName = "rclone_key"
	// SSHKeyRelativePath is the relative path to the SSH key from dataDir
	SSHKeyRelativePath = "certs/rclone_key"
)

// SSHKeyInfo contains information about an SSH key
type SSHKeyInfo struct {
	Exists     bool   `json:"exists"`
	KeyPath    string `json:"key_path"`    // Absolute path to private key
	PublicKey  string `json:"public_key"`  // Content of public key
	RelativePath string `json:"relative_path"` // Relative path for storing in DB
}

// GetSSHKeyInfo returns information about the Anemone SSH key
func GetSSHKeyInfo(dataDir string) (*SSHKeyInfo, error) {
	keyPath := filepath.Join(dataDir, SSHKeyRelativePath)
	pubKeyPath := keyPath + ".pub"

	info := &SSHKeyInfo{
		KeyPath:      keyPath,
		RelativePath: SSHKeyRelativePath,
	}

	// Check if private key exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		info.Exists = false
		return info, nil
	}

	info.Exists = true

	// Read public key content
	pubKeyData, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return info, fmt.Errorf("failed to read public key: %w", err)
	}

	info.PublicKey = strings.TrimSpace(string(pubKeyData))
	return info, nil
}

// GenerateSSHKey generates a new ed25519 SSH key pair for rclone SFTP connections
// Returns the key info including the public key content
func GenerateSSHKey(dataDir string) (*SSHKeyInfo, error) {
	keyPath := filepath.Join(dataDir, SSHKeyRelativePath)
	pubKeyPath := keyPath + ".pub"

	// Ensure certs directory exists
	certsDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(certsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create certs directory: %w", err)
	}

	// Remove existing keys if they exist
	os.Remove(keyPath)
	os.Remove(pubKeyPath)

	// Generate new key using ssh-keygen
	// -t ed25519: Use Ed25519 algorithm (modern, secure, fast)
	// -f <path>: Output file path
	// -N "": Empty passphrase (for automation)
	// -C "anemone-rclone": Comment to identify the key
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-C", "anemone-rclone")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to generate SSH key: %s", stderr.String())
	}

	// Set proper permissions
	if err := os.Chmod(keyPath, 0600); err != nil {
		return nil, fmt.Errorf("failed to set key permissions: %w", err)
	}

	// Read and return key info
	return GetSSHKeyInfo(dataDir)
}

// ResolveKeyPath resolves a key path that may be relative to dataDir
// - If keyPath is empty, returns empty string
// - If keyPath is absolute (starts with /), returns as-is
// - If keyPath is relative, joins with dataDir
func ResolveKeyPath(keyPath, dataDir string) string {
	if keyPath == "" {
		return ""
	}

	// If it's an absolute path, return as-is
	if filepath.IsAbs(keyPath) {
		return keyPath
	}

	// Relative path: resolve against dataDir
	return filepath.Join(dataDir, keyPath)
}

// IsRelativePath checks if a path is relative (not absolute)
func IsRelativePath(path string) bool {
	return path != "" && !filepath.IsAbs(path)
}
