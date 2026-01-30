// Package wireguard manages WireGuard VPN client configuration.
//
// This file handles retrieving WireGuard connection status.
package wireguard

import (
	"bufio"
	"os/exec"
	"strings"
)

// Status represents the current WireGuard connection status.
type Status struct {
	Connected       bool
	Interface       string
	PublicKey       string
	ListeningPort   string
	PeerPublicKey   string
	PeerEndpoint    string
	LatestHandshake string
	TransferRx      string
	TransferTx      string
}

// GetStatus retrieves the current WireGuard interface status.
func GetStatus(name string) *Status {
	status := &Status{
		Interface: name,
		Connected: false,
	}

	// Run wg show
	cmd := exec.Command("sudo", "wg", "show", name)
	output, err := cmd.Output()
	if err != nil {
		return status
	}

	// If we got output, interface is up
	if len(output) > 0 {
		status.Connected = true
		parseWgShowOutput(string(output), status)
	}

	return status
}

// parseWgShowOutput parses the output of `wg show` command.
func parseWgShowOutput(output string, status *Status) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	inPeerSection := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check for peer section
		if strings.HasPrefix(line, "peer:") {
			inPeerSection = true
			status.PeerPublicKey = strings.TrimSpace(strings.TrimPrefix(line, "peer:"))
			continue
		}

		// Parse key: value pairs
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if inPeerSection {
			switch key {
			case "endpoint":
				status.PeerEndpoint = value
			case "latest handshake":
				status.LatestHandshake = value
			case "transfer":
				// Format: "1.23 MiB received, 456.78 KiB sent"
				transferParts := strings.Split(value, ",")
				if len(transferParts) >= 1 {
					status.TransferRx = strings.TrimSpace(strings.TrimSuffix(transferParts[0], "received"))
				}
				if len(transferParts) >= 2 {
					status.TransferTx = strings.TrimSpace(strings.TrimSuffix(transferParts[1], "sent"))
				}
			}
		} else {
			switch key {
			case "public key":
				status.PublicKey = value
			case "listening port":
				status.ListeningPort = value
			}
		}
	}
}
