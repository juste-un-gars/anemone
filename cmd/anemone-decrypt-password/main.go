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
		fmt.Fprintf(os.Stderr, "Usage: %s <base64_encrypted_password> <master_key>\n", os.Args[0])
		os.Exit(1)
	}

	encryptedPasswordB64 := os.Args[1]
	masterKey := os.Args[2]

	// Decode base64
	encryptedPassword, err := base64.StdEncoding.DecodeString(encryptedPasswordB64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding base64: %v\n", err)
		os.Exit(1)
	}

	// Decrypt password
	password, err := crypto.DecryptPassword(encryptedPassword, masterKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decrypting password: %v\n", err)
		os.Exit(1)
	}

	// Output password to stdout
	fmt.Print(password)
}
