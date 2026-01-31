// Package wireguard manages WireGuard VPN client configuration.
//
// This file handles parsing WireGuard .conf files.
package wireguard

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// ParseConfig parses a WireGuard .conf file content and returns a Config.
func ParseConfig(content string) (*Config, error) {
	cfg := &Config{
		Name:                "wg0",
		PersistentKeepalive: 25,
		AllowedIPs:          "0.0.0.0/0",
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	var section string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for section headers
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.Trim(line, "[]"))
			continue
		}

		// Parse key = value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch section {
		case "interface":
			switch key {
			case "privatekey":
				cfg.PrivateKey = value
			case "address":
				cfg.Address = value
			case "dns":
				cfg.DNS = value
			}
		case "peer":
			switch key {
			case "publickey":
				cfg.PeerPublicKey = value
			case "presharedkey":
				cfg.PeerPresharedKey = value
			case "endpoint":
				cfg.PeerEndpoint = value
			case "allowedips":
				cfg.AllowedIPs = value
			case "persistentkeepalive":
				if v, err := strconv.Atoi(value); err == nil {
					cfg.PersistentKeepalive = v
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	// Validate required fields
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("missing PrivateKey in [Interface]")
	}
	if cfg.Address == "" {
		return nil, fmt.Errorf("missing Address in [Interface]")
	}
	if cfg.PeerPublicKey == "" {
		return nil, fmt.Errorf("missing PublicKey in [Peer]")
	}
	if cfg.PeerEndpoint == "" {
		return nil, fmt.Errorf("missing Endpoint in [Peer]")
	}

	// Derive public key from private key
	if cfg.PrivateKey != "" {
		if pubKey, err := DerivePublicKey(cfg.PrivateKey); err == nil {
			cfg.PublicKey = pubKey
		}
	}

	return cfg, nil
}
