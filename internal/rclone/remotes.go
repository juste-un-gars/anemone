// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package rclone

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// RemoteInfo describes a named remote configured in rclone.
type RemoteInfo struct {
	Name string // Remote name (without trailing colon)
}

// ListRemotes returns the list of named remotes configured via "rclone config".
func ListRemotes() ([]RemoteInfo, error) {
	cmd := exec.Command("rclone", "listremotes")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("rclone listremotes failed: %s", stderr.String())
	}

	var remotes []RemoteInfo
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// rclone listremotes outputs "remotename:" per line
		name := strings.TrimSuffix(line, ":")
		if name != "" {
			remotes = append(remotes, RemoteInfo{Name: name})
		}
	}

	return remotes, nil
}

// ObscurePassword obscures a plaintext password using "rclone obscure" for safe storage.
func ObscurePassword(plain string) (string, error) {
	cmd := exec.Command("rclone", "obscure", plain)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("rclone obscure failed: %s", stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}
