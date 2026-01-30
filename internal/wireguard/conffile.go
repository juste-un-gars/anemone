// Package wireguard manages WireGuard VPN client configuration.
//
// This file handles generating WireGuard .conf files for wg-quick.
package wireguard

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// execCommand is a wrapper for exec.Command (allows mocking in tests)
var execCommand = exec.Command

// GenerateConfFile generates the content of a WireGuard .conf file.
func GenerateConfFile(cfg *Config) string {
	content := "[Interface]\n"
	content += fmt.Sprintf("PrivateKey = %s\n", cfg.PrivateKey)
	content += fmt.Sprintf("Address = %s\n", cfg.Address)
	if cfg.DNS != "" {
		content += fmt.Sprintf("DNS = %s\n", cfg.DNS)
	}

	content += "\n[Peer]\n"
	content += fmt.Sprintf("PublicKey = %s\n", cfg.PeerPublicKey)
	content += fmt.Sprintf("Endpoint = %s\n", cfg.PeerEndpoint)
	content += fmt.Sprintf("AllowedIPs = %s\n", cfg.AllowedIPs)
	if cfg.PersistentKeepalive > 0 {
		content += fmt.Sprintf("PersistentKeepalive = %d\n", cfg.PersistentKeepalive)
	}

	return content
}

// WriteConfFile writes the WireGuard configuration to /etc/wireguard/{name}.conf
// Uses sudo tee to write with root permissions.
func WriteConfFile(cfg *Config) error {
	confPath := filepath.Join("/etc/wireguard", cfg.Name+".conf")
	content := GenerateConfFile(cfg)

	// Use sudo tee to write the file with root permissions
	cmd := execCommand("sudo", "tee", confPath)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = nil // Suppress tee output

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Set proper permissions (readable only by root)
	chmodCmd := execCommand("sudo", "chmod", "600", confPath)
	if err := chmodCmd.Run(); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return nil
}

// RemoveConfFile removes the WireGuard configuration file.
func RemoveConfFile(name string) error {
	confPath := filepath.Join("/etc/wireguard", name+".conf")
	cmd := execCommand("sudo", "rm", "-f", confPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove config file: %w", err)
	}
	return nil
}

// Connect starts the WireGuard VPN connection.
func Connect(cfg *Config) error {
	// Write the config file
	if err := WriteConfFile(cfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Start the interface
	cmd := execCommand("sudo", "wg-quick", "up", cfg.Name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg-quick up failed: %s: %s", err, string(output))
	}

	return nil
}

// Disconnect stops the WireGuard VPN connection.
func Disconnect(name string) error {
	cmd := execCommand("sudo", "wg-quick", "down", name)
	_ = cmd.Run() // Ignore error - interface might already be down
	return nil
}
