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
	ID        int
	Name      string
	Address   string
	Port      int
	PublicKey string
	Enabled   bool
	Status    string // "online", "offline", "error", "unknown"
	LastSeen  *time.Time
	LastSync  *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Create creates a new peer
func Create(db *sql.DB, peer *Peer) error {
	query := `INSERT INTO peers (name, address, port, public_key, enabled, status, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	result, err := db.Exec(query, peer.Name, peer.Address, peer.Port, peer.PublicKey, peer.Enabled, peer.Status)
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
	query := `SELECT id, name, address, port, public_key, enabled, status, last_seen, last_sync, created_at, updated_at
	          FROM peers WHERE id = ?`

	err := db.QueryRow(query, id).Scan(
		&peer.ID, &peer.Name, &peer.Address, &peer.Port, &peer.PublicKey,
		&peer.Enabled, &peer.Status, &peer.LastSeen, &peer.LastSync,
		&peer.CreatedAt, &peer.UpdatedAt,
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
	query := `SELECT id, name, address, port, public_key, enabled, status, last_seen, last_sync, created_at, updated_at
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
			&peer.ID, &peer.Name, &peer.Address, &peer.Port, &peer.PublicKey,
			&peer.Enabled, &peer.Status, &peer.LastSeen, &peer.LastSync,
			&peer.CreatedAt, &peer.UpdatedAt,
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
	query := `UPDATE peers SET name = ?, address = ?, port = ?, public_key = ?,
	          enabled = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`

	_, err := db.Exec(query, peer.Name, peer.Address, peer.Port, peer.PublicKey,
		peer.Enabled, peer.Status, peer.ID)
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

// TestConnection tests if a peer is reachable
func TestConnection(peer *Peer) (bool, error) {
	// Create HTTPS client that accepts self-signed certificates
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   5 * time.Second,
	}

	// Try to reach the health endpoint
	url := fmt.Sprintf("https://%s:%d/health", peer.Address, peer.Port)
	resp, err := client.Get(url)
	if err != nil {
		return false, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// URL returns the full HTTPS URL of the peer
func (p *Peer) URL() string {
	return fmt.Sprintf("https://%s:%d", p.Address, p.Port)
}
