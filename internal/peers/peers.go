// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package peers

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

// Peer represents a remote Anemone instance for P2P synchronization
type Peer struct {
	ID                 int
	Name               string
	Address            string
	Port               int
	PublicKey          *string // Can be NULL
	Password           *string // Can be NULL - password for peer authentication
	Enabled            bool
	Status             string // "online", "offline", "error", "unknown"
	LastSeen           *time.Time
	LastSync           *time.Time
	SyncEnabled        bool
	SyncFrequency      string // "daily", "weekly", "monthly", "interval"
	SyncTime           string // "HH:MM" format
	SyncDayOfWeek      *int   // 0-6 (0=Sunday), NULL if not weekly
	SyncDayOfMonth     *int   // 1-31, NULL if not monthly
	SyncIntervalMinutes int    // Interval in minutes for "interval" frequency
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Create creates a new peer
func Create(db *sql.DB, peer *Peer) error {
	query := `INSERT INTO peers (name, address, port, public_key, password, enabled, status,
	          sync_enabled, sync_frequency, sync_time, sync_day_of_week, sync_day_of_month,
	          sync_interval_minutes, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	result, err := db.Exec(query, peer.Name, peer.Address, peer.Port, peer.PublicKey, peer.Password,
		peer.Enabled, peer.Status, peer.SyncEnabled, peer.SyncFrequency, peer.SyncTime,
		peer.SyncDayOfWeek, peer.SyncDayOfMonth, peer.SyncIntervalMinutes)
	if err != nil {
		return fmt.Errorf("failed to create peer: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get peer ID: %w", err)
	}

	peer.ID = int(id)
	return nil
}

// GetByID retrieves a peer by ID
func GetByID(db *sql.DB, id int) (*Peer, error) {
	peer := &Peer{}
	query := `SELECT id, name, address, port, public_key, password, enabled, status, last_seen, last_sync,
	          sync_enabled, sync_frequency, sync_time, sync_day_of_week, sync_day_of_month,
	          sync_interval_minutes, created_at, updated_at
	          FROM peers WHERE id = ?`

	err := db.QueryRow(query, id).Scan(
		&peer.ID, &peer.Name, &peer.Address, &peer.Port, &peer.PublicKey, &peer.Password,
		&peer.Enabled, &peer.Status, &peer.LastSeen, &peer.LastSync,
		&peer.SyncEnabled, &peer.SyncFrequency, &peer.SyncTime, &peer.SyncDayOfWeek, &peer.SyncDayOfMonth,
		&peer.SyncIntervalMinutes, &peer.CreatedAt, &peer.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("peer not found")
		}
		return nil, fmt.Errorf("failed to get peer: %w", err)
	}

	return peer, nil
}

// GetAll retrieves all peers
func GetAll(db *sql.DB) ([]*Peer, error) {
	query := `SELECT id, name, address, port, public_key, password, enabled, status, last_seen, last_sync,
	          sync_enabled, sync_frequency, sync_time, sync_day_of_week, sync_day_of_month,
	          sync_interval_minutes, created_at, updated_at
	          FROM peers ORDER BY created_at DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query peers: %w", err)
	}
	defer rows.Close()

	var peers []*Peer
	for rows.Next() {
		peer := &Peer{}
		err := rows.Scan(
			&peer.ID, &peer.Name, &peer.Address, &peer.Port, &peer.PublicKey, &peer.Password,
			&peer.Enabled, &peer.Status, &peer.LastSeen, &peer.LastSync,
			&peer.SyncEnabled, &peer.SyncFrequency, &peer.SyncTime, &peer.SyncDayOfWeek, &peer.SyncDayOfMonth,
			&peer.SyncIntervalMinutes, &peer.CreatedAt, &peer.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan peer: %w", err)
		}
		peers = append(peers, peer)
	}

	return peers, nil
}

// Update updates a peer
func Update(db *sql.DB, peer *Peer) error {
	query := `UPDATE peers SET name = ?, address = ?, port = ?, public_key = ?, password = ?,
	          enabled = ?, status = ?, sync_enabled = ?, sync_frequency = ?, sync_time = ?,
	          sync_day_of_week = ?, sync_day_of_month = ?, sync_interval_minutes = ?,
	          updated_at = CURRENT_TIMESTAMP
	          WHERE id = ?`

	_, err := db.Exec(query, peer.Name, peer.Address, peer.Port, peer.PublicKey, peer.Password,
		peer.Enabled, peer.Status, peer.SyncEnabled, peer.SyncFrequency, peer.SyncTime,
		peer.SyncDayOfWeek, peer.SyncDayOfMonth, peer.SyncIntervalMinutes, peer.ID)
	if err != nil {
		return fmt.Errorf("failed to update peer: %w", err)
	}

	return nil
}

// Delete deletes a peer
func Delete(db *sql.DB, id int) error {
	query := `DELETE FROM peers WHERE id = ?`
	_, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete peer: %w", err)
	}

	return nil
}

// UpdateStatus updates the status and last_seen timestamp of a peer
func UpdateStatus(db *sql.DB, id int, status string) error {
	query := `UPDATE peers SET status = ?, last_seen = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update peer status: %w", err)
	}

	return nil
}

// TestConnection tests if a peer is reachable and validates authentication if password is set
func TestConnection(peer *Peer) (bool, error) {
	// Create HTTPS client that accepts self-signed certificates
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   5 * time.Second,
	}

	// First, check basic connectivity with health endpoint
	healthURL := fmt.Sprintf("https://%s:%d/health", peer.Address, peer.Port)
	resp, err := client.Get(healthURL)
	if err != nil {
		return false, fmt.Errorf("connection failed: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	// ALWAYS test authentication on a protected endpoint, even if password is empty
	// This detects if the remote server requires a password but we don't have one configured
	testURL := fmt.Sprintf("https://%s:%d/api/sync/manifest?user_id=1&share_name=test", peer.Address, peer.Port)
	req, err := http.NewRequest(http.MethodGet, testURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create auth test request: %w", err)
	}

	// Add authentication header if password is configured
	if peer.Password != nil && *peer.Password != "" {
		req.Header.Set("X-Sync-Password", *peer.Password)
	}

	resp, err = client.Do(req)
	if err != nil {
		return false, fmt.Errorf("auth test request failed: %w", err)
	}
	resp.Body.Close()

	// Check response status
	if resp.StatusCode == http.StatusUnauthorized {
		return false, fmt.Errorf("authentication required: server expects a password")
	}
	if resp.StatusCode == http.StatusForbidden {
		return false, fmt.Errorf("authentication failed: invalid password")
	}
	// 200 (manifest exists) or 404 (no manifest) are both OK - authentication succeeded
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return false, fmt.Errorf("unexpected auth test response: status %d", resp.StatusCode)
	}

	return true, nil
}

// URL returns the full HTTPS URL of the peer
func (p *Peer) URL() string {
	return fmt.Sprintf("https://%s:%d", p.Address, p.Port)
}

// UpdateLastSync updates the last_sync timestamp of a peer
func UpdateLastSync(db *sql.DB, peerID int) error {
	query := `UPDATE peers SET last_sync = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Exec(query, peerID)
	if err != nil {
		return fmt.Errorf("failed to update peer last_sync: %w", err)
	}
	return nil
}

// ShouldSyncPeer determines if a peer should be synchronized based on its configuration
func ShouldSyncPeer(peer *Peer) bool {
	// Check if sync is enabled for this peer
	if !peer.SyncEnabled || !peer.Enabled {
		return false
	}

	// First sync ever
	if peer.LastSync == nil {
		return true
	}

	now := time.Now()
	lastSync := *peer.LastSync

	// Parse sync time (format: "HH:MM")
	var syncHour, syncMinute int
	fmt.Sscanf(peer.SyncTime, "%d:%d", &syncHour, &syncMinute)

	switch peer.SyncFrequency {
	case "interval":
		// Interval-based sync: check if enough time has passed since last sync
		if peer.SyncIntervalMinutes <= 0 {
			return false
		}

		interval := time.Duration(peer.SyncIntervalMinutes) * time.Minute
		return now.Sub(lastSync) >= interval

	case "daily":
		// Daily sync: check if we've passed the sync time today and haven't synced today
		lastSyncDate := lastSync.Format("2006-01-02")
		todayDate := now.Format("2006-01-02")

		// If last sync was on a different day and we've passed the sync time
		if lastSyncDate != todayDate && (now.Hour() > syncHour || (now.Hour() == syncHour && now.Minute() >= syncMinute)) {
			return true
		}
		return false

	case "weekly":
		// Weekly sync: check if we're on the right day of week and past sync time
		if peer.SyncDayOfWeek == nil {
			return false
		}

		currentDayOfWeek := int(now.Weekday()) // 0=Sunday, 1=Monday, ..., 6=Saturday
		if currentDayOfWeek != *peer.SyncDayOfWeek {
			return false
		}

		// Check if we've passed the sync time today
		if now.Hour() < syncHour || (now.Hour() == syncHour && now.Minute() < syncMinute) {
			return false
		}

		// Check if last sync was before today
		lastSyncDate := lastSync.Format("2006-01-02")
		todayDate := now.Format("2006-01-02")
		return lastSyncDate != todayDate

	case "monthly":
		// Monthly sync: check if we're on the right day of month and past sync time
		if peer.SyncDayOfMonth == nil {
			return false
		}

		if now.Day() != *peer.SyncDayOfMonth {
			return false
		}

		// Check if we've passed the sync time today
		if now.Hour() < syncHour || (now.Hour() == syncHour && now.Minute() < syncMinute) {
			return false
		}

		// Check if last sync was before today
		lastSyncDate := lastSync.Format("2006-01-02")
		todayDate := now.Format("2006-01-02")
		return lastSyncDate != todayDate

	default:
		return false
	}
}
