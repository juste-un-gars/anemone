// Package onlyoffice manages the OnlyOffice Document Server Docker container.
//
// This file handles Docker operations: pull, start, stop, restart, remove,
// and status checks for the OnlyOffice container.
package onlyoffice

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/juste-un-gars/anemone/internal/logger"
)

const ContainerName = "onlyoffice-docs"
const ImageName = "onlyoffice/documentserver"

// IsDockerInstalled checks if docker CLI is available.
func IsDockerInstalled() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

// IsImagePresent checks if the OnlyOffice image has been pulled.
func IsImagePresent() bool {
	cmd := exec.Command("docker", "image", "inspect", ImageName)
	return cmd.Run() == nil
}

// ContainerStatus returns the status of the OnlyOffice container.
// Returns: "running", "exited", "paused", "created", or "" if not found.
func ContainerStatus() string {
	cmd := exec.Command("docker", "inspect", "--format", "{{.State.Status}}", ContainerName)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// PullImage pulls the OnlyOffice Document Server image.
func PullImage() error {
	cmd := exec.Command("docker", "pull", ImageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker pull failed: %s", string(output))
	}
	logger.Info("OnlyOffice image pulled successfully")
	return nil
}

// StartContainer starts the OnlyOffice container.
// If container doesn't exist, it creates a new one.
// If container exists but is stopped, it starts it.
func StartContainer(secret, ooURL string) error {
	status := ContainerStatus()

	if status == "running" {
		return nil
	}

	if status != "" {
		// Container exists but not running — start it
		cmd := exec.Command("docker", "start", ContainerName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("docker start failed: %s", string(output))
		}
		logger.Info("OnlyOffice container started")
		return nil
	}

	// Container doesn't exist — create and start
	port, err := extractPort(ooURL)
	if err != nil {
		return fmt.Errorf("invalid OnlyOffice URL: %w", err)
	}

	args := []string{
		"run", "-d",
		"--name", ContainerName,
		"-p", fmt.Sprintf("%s:80", port),
		"-e", fmt.Sprintf("JWT_SECRET=%s", secret),
		"--restart=always",
		ImageName,
	}

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker run failed: %s", string(output))
	}
	logger.Info("OnlyOffice container created and started", "port", port)
	return nil
}

// StopContainer stops the OnlyOffice container.
func StopContainer() error {
	cmd := exec.Command("docker", "stop", ContainerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker stop failed: %s", string(output))
	}
	logger.Info("OnlyOffice container stopped")
	return nil
}

// RestartContainer restarts the OnlyOffice container.
func RestartContainer() error {
	cmd := exec.Command("docker", "restart", ContainerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker restart failed: %s", string(output))
	}
	logger.Info("OnlyOffice container restarted")
	return nil
}

// RemoveContainer stops and removes the OnlyOffice container.
func RemoveContainer() error {
	exec.Command("docker", "stop", ContainerName).Run()
	cmd := exec.Command("docker", "rm", ContainerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker rm failed: %s", string(output))
	}
	logger.Info("OnlyOffice container removed")
	return nil
}

// extractPort extracts the port from a URL string.
func extractPort(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			return "443", nil
		}
		return "80", nil
	}
	return port, nil
}
