// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package main

import (
	"fmt"
	"os"

	"github.com/juste-un-gars/anemone/internal/crypto"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "Usage: %s <base64_encrypted_key> <old_master_key> <new_master_key>\n", os.Args[0])
		os.Exit(1)
	}

	encryptedKeyB64 := os.Args[1]
	oldMasterKey := os.Args[2]
	newMasterKey := os.Args[3]

	// Decrypt with old master key
	plainKey, err := crypto.DecryptKey(encryptedKeyB64, oldMasterKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decrypting with old master key: %v\n", err)
		os.Exit(1)
	}

	// Re-encrypt with new master key
	newEncryptedKey, err := crypto.EncryptKey(plainKey, newMasterKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encrypting with new master key: %v\n", err)
		os.Exit(1)
	}

	// Output base64-encoded encrypted key to stdout
	fmt.Print(newEncryptedKey)
}
