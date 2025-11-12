// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/juste-un-gars/anemone/internal/backup"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <encrypted_backup_file> <passphrase>\n", os.Args[0])
		os.Exit(1)
	}

	encryptedFile := os.Args[1]
	passphrase := os.Args[2]

	// Read encrypted file
	encryptedData, err := os.ReadFile(encryptedFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading encrypted file: %v\n", err)
		os.Exit(1)
	}

	// Decrypt backup
	backupData, err := backup.DecryptBackup(encryptedData, passphrase)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decrypting backup: %v\n", err)
		os.Exit(1)
	}

	// Output JSON to stdout
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(backupData); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}
