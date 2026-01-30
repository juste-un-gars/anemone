// Package wireguard manages WireGuard VPN client configuration.
//
// This package handles storing and retrieving WireGuard configuration
// from the database, and generating .conf files for wg-quick.
package wireguard

import (
	"database/sql"
	"fmt"
	"os/exec"
	"time"
)

// Config represents a WireGuard client configuration
type Config struct {
	ID        int
	Name      string // Interface name (default: wg0)
	CreatedAt time.Time
	UpdatedAt time.Time

	// Interface section
	PrivateKey string
	Address    string // Client IP (e.g., 10.0.0.5/32)
	DNS        string // Optional DNS server

	// Peer section (server)
	PeerPublicKey       string
	PeerEndpoint        string // Server address (e.g., vpn.example.com:51820)
	AllowedIPs          string // Routes through VPN (e.g., 0.0.0.0/0 or 10.0.0.0/24)
	PersistentKeepalive int    // Keepalive interval in seconds

	// Options
	Enabled   bool // Is VPN connection active
	AutoStart bool // Start VPN on Anemone boot
}

// Get retrieves the WireGuard configuration from database.
// Returns nil if no configuration exists.
func Get(db *sql.DB) (*Config, error) {
	query := `SELECT id, name, private_key, address, dns,
		peer_public_key, peer_endpoint, allowed_ips, persistent_keepalive,
		enabled, auto_start, created_at, updated_at
		FROM wireguard_config LIMIT 1`

	cfg := &Config{}
	var privateKey, address, dns, peerPublicKey, peerEndpoint, allowedIPs sql.NullString
	var createdAt, updatedAt sql.NullTime

	err := db.QueryRow(query).Scan(
		&cfg.ID, &cfg.Name, &privateKey, &address, &dns,
		&peerPublicKey, &peerEndpoint, &allowedIPs, &cfg.PersistentKeepalive,
		&cfg.Enabled, &cfg.AutoStart, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get wireguard config: %w", err)
	}

	cfg.PrivateKey = privateKey.String
	cfg.Address = address.String
	cfg.DNS = dns.String
	cfg.PeerPublicKey = peerPublicKey.String
	cfg.PeerEndpoint = peerEndpoint.String
	cfg.AllowedIPs = allowedIPs.String
	if createdAt.Valid {
		cfg.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		cfg.UpdatedAt = updatedAt.Time
	}

	return cfg, nil
}

// Save creates or updates the WireGuard configuration.
func Save(db *sql.DB, cfg *Config) error {
	// Check if config exists
	existing, err := Get(db)
	if err != nil {
		return err
	}

	if existing == nil {
		// Insert new config
		query := `INSERT INTO wireguard_config
			(name, private_key, address, dns, peer_public_key, peer_endpoint,
			allowed_ips, persistent_keepalive, enabled, auto_start, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

		result, err := db.Exec(query,
			cfg.Name, cfg.PrivateKey, cfg.Address, cfg.DNS,
			cfg.PeerPublicKey, cfg.PeerEndpoint, cfg.AllowedIPs,
			cfg.PersistentKeepalive, cfg.Enabled, cfg.AutoStart,
		)
		if err != nil {
			return fmt.Errorf("failed to insert wireguard config: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get insert id: %w", err)
		}
		cfg.ID = int(id)
	} else {
		// Update existing config
		query := `UPDATE wireguard_config SET
			name = ?, private_key = ?, address = ?, dns = ?,
			peer_public_key = ?, peer_endpoint = ?, allowed_ips = ?,
			persistent_keepalive = ?, enabled = ?, auto_start = ?,
			updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`

		_, err := db.Exec(query,
			cfg.Name, cfg.PrivateKey, cfg.Address, cfg.DNS,
			cfg.PeerPublicKey, cfg.PeerEndpoint, cfg.AllowedIPs,
			cfg.PersistentKeepalive, cfg.Enabled, cfg.AutoStart,
			existing.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to update wireguard config: %w", err)
		}
		cfg.ID = existing.ID
	}

	return nil
}

// Delete removes the WireGuard configuration from database.
func Delete(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM wireguard_config")
	if err != nil {
		return fmt.Errorf("failed to delete wireguard config: %w", err)
	}
	return nil
}

// SetEnabled updates only the enabled status.
func SetEnabled(db *sql.DB, enabled bool) error {
	_, err := db.Exec("UPDATE wireguard_config SET enabled = ?, updated_at = CURRENT_TIMESTAMP", enabled)
	if err != nil {
		return fmt.Errorf("failed to update enabled status: %w", err)
	}
	return nil
}

// SetAutoStart updates only the auto_start status.
func SetAutoStart(db *sql.DB, autoStart bool) error {
	_, err := db.Exec("UPDATE wireguard_config SET auto_start = ?, updated_at = CURRENT_TIMESTAMP", autoStart)
	if err != nil {
		return fmt.Errorf("failed to update auto_start status: %w", err)
	}
	return nil
}

// IsConfigured returns true if a WireGuard configuration exists with required fields.
func IsConfigured(db *sql.DB) (bool, error) {
	cfg, err := Get(db)
	if err != nil {
		return false, err
	}
	if cfg == nil {
		return false, nil
	}
	// Minimum required: private key, address, peer public key, endpoint
	return cfg.PrivateKey != "" && cfg.Address != "" &&
		cfg.PeerPublicKey != "" && cfg.PeerEndpoint != "", nil
}

// AutoConnect connects to VPN if auto_start is enabled.
// Should be called at application startup.
// Returns nil if no config exists or auto_start is disabled.
func AutoConnect(db *sql.DB) error {
	cfg, err := Get(db)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	// No config or auto_start disabled
	if cfg == nil || !cfg.AutoStart {
		return nil
	}

	// Check if already connected
	if IsConnected(cfg.Name) {
		return nil
	}

	// Connect
	if err := Connect(cfg); err != nil {
		return fmt.Errorf("auto-connect failed: %w", err)
	}

	// Update enabled status
	if err := SetEnabled(db, true); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// IsConnected checks if a WireGuard interface is currently active.
func IsConnected(name string) bool {
	cmd := exec.Command("sudo", "wg", "show", name)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(output) > 0
}
