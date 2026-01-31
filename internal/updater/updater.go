// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package updater handles version checking and update notifications from GitHub releases.
package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// Current version of Anemone
	Version = "0.13.5-beta"

	// GitHub API endpoint for releases (includes pre-releases)
	GitHubAPIURL = "https://api.github.com/repos/juste-un-gars/anemone/releases"
)

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	CurrentVersion string    `json:"current_version"`
	LatestVersion  string    `json:"latest_version"`
	Available      bool      `json:"available"`
	ReleaseURL     string    `json:"release_url"`
	ReleaseNotes   string    `json:"release_notes"`
	PublishedAt    time.Time `json:"published_at"`
}

// GitHubRelease represents the GitHub API response
type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
	Prerelease  bool      `json:"prerelease"`
}

// CheckUpdate checks GitHub API for a new version
func CheckUpdate() (*UpdateInfo, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make request to GitHub API
	req, err := http.NewRequest("GET", GitHubAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent header (required by GitHub API)
	req.Header.Set("User-Agent", "Anemone-NAS/"+Version)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response (array of releases)
	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	// Find the latest release (first in the list, including pre-releases)
	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found")
	}
	release := releases[0]

	// Compare versions
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(Version, "v")

	updateAvailable := compareVersions(latestVersion, currentVersion)

	return &UpdateInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		Available:      updateAvailable,
		ReleaseURL:     release.HTMLURL,
		ReleaseNotes:   release.Body,
		PublishedAt:    release.PublishedAt,
	}, nil
}

// compareVersions returns true if latest > current
// Simple comparison for semantic versioning (e.g., "1.0.0" > "0.9.0-beta")
func compareVersions(latest, current string) bool {
	// Remove any suffix like "-beta", "-alpha", etc.
	latest = strings.Split(latest, "-")[0]
	current = strings.Split(current, "-")[0]

	// Split into parts
	latestParts := strings.Split(latest, ".")
	currentParts := strings.Split(current, ".")

	// Ensure both have at least 3 parts (major.minor.patch)
	for len(latestParts) < 3 {
		latestParts = append(latestParts, "0")
	}
	for len(currentParts) < 3 {
		currentParts = append(currentParts, "0")
	}

	// Compare each part
	for i := 0; i < 3; i++ {
		var latestNum, currentNum int
		fmt.Sscanf(latestParts[i], "%d", &latestNum)
		fmt.Sscanf(currentParts[i], "%d", &currentNum)

		if latestNum > currentNum {
			return true
		} else if latestNum < currentNum {
			return false
		}
	}

	// Versions are equal
	return false
}

// GetCurrentVersion returns the current version of Anemone
func GetCurrentVersion() string {
	return Version
}
