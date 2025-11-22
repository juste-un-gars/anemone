// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/juste-un-gars/anemone/internal/crypto"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <plain_password> <master_key>\n", os.Args[0])
		os.Exit(1)
	}

	plainPassword := os.Args[1]
	masterKey := os.Args[2]

	// Encrypt password with master key
	encryptedBytes, err := crypto.EncryptPassword(plainPassword, masterKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encrypting password: %v\n", err)
		os.Exit(1)
	}

	// Output base64-encoded encrypted password to stdout
	fmt.Print(base64.StdEncoding.EncodeToString(encryptedBytes))
}
