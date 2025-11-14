// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

// GenerateEncryptionKey generates a 32-byte random encryption key
func GenerateEncryptionKey() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// HashKey returns SHA-256 hash of the key (for verification)
func HashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// EncryptKey encrypts an encryption key with a master key
func EncryptKey(plainKey, masterKey string) (string, error) {
	// Derive a proper 32-byte key from master key
	hash := sha256.Sum256([]byte(masterKey))
	key := hash[:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plainKey), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptKey decrypts an encryption key with a master key
func DecryptKey(encryptedKey, masterKey string) (string, error) {
	// Derive a proper 32-byte key from master key
	hash := sha256.Sum256([]byte(masterKey))
	key := hash[:]

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword compares a password with a hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// EncryptStream encrypts data from reader and writes to writer using AES-256-GCM
// The encryption key must be base64-encoded 32-byte key
// Format: [nonce (12 bytes)][encrypted data with auth tag]
func EncryptStream(reader io.Reader, writer io.Writer, encryptionKey string) error {
	// Decode the base64 key
	key, err := base64.StdEncoding.DecodeString(encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to decode encryption key: %w", err)
	}

	if len(key) != 32 {
		return fmt.Errorf("encryption key must be 32 bytes, got %d", len(key))
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Write nonce first (needed for decryption)
	if _, err := writer.Write(nonce); err != nil {
		return fmt.Errorf("failed to write nonce: %w", err)
	}

	// Read all data (for GCM we need the full plaintext)
	plaintext, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	// Encrypt data
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Write encrypted data
	if _, err := writer.Write(ciphertext); err != nil {
		return fmt.Errorf("failed to write encrypted data: %w", err)
	}

	return nil
}

// DecryptStream decrypts data from reader and writes to writer using AES-256-GCM
// The encryption key must be base64-encoded 32-byte key
// Expected format: [nonce (12 bytes)][encrypted data with auth tag]
func DecryptStream(reader io.Reader, writer io.Writer, encryptionKey string) error {
	// Decode the base64 key
	key, err := base64.StdEncoding.DecodeString(encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to decode encryption key: %w", err)
	}

	if len(key) != 32 {
		return fmt.Errorf("encryption key must be 32 bytes, got %d", len(key))
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	// Read nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(reader, nonce); err != nil {
		return fmt.Errorf("failed to read nonce: %w", err)
	}

	// Read all encrypted data
	ciphertext, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read encrypted data: %w", err)
	}

	// Decrypt data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w (invalid key or corrupted data)", err)
	}

	// Write decrypted data
	if _, err := writer.Write(plaintext); err != nil {
		return fmt.Errorf("failed to write decrypted data: %w", err)
	}

	return nil
}

// EncryptPassword encrypts a plaintext password using the master key
// Returns base64-encoded encrypted password suitable for database storage
// Used to securely store passwords for SMB restoration after backup/restore
func EncryptPassword(password, masterKey string) ([]byte, error) {
	if password == "" {
		return nil, errors.New("password cannot be empty")
	}
	if masterKey == "" {
		return nil, errors.New("master key cannot be empty")
	}

	// Derive a proper 32-byte key from master key
	hash := sha256.Sum256([]byte(masterKey))
	key := hash[:]

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt password
	ciphertext := gcm.Seal(nonce, nonce, []byte(password), nil)
	return ciphertext, nil
}

// DecryptPassword decrypts an encrypted password using the master key
// Returns the plaintext password
// Used to restore SMB passwords after backup/restore
func DecryptPassword(encryptedPassword []byte, masterKey string) (string, error) {
	if len(encryptedPassword) == 0 {
		return "", errors.New("encrypted password cannot be empty")
	}
	if masterKey == "" {
		return "", errors.New("master key cannot be empty")
	}

	// Derive a proper 32-byte key from master key
	hash := sha256.Sum256([]byte(masterKey))
	key := hash[:]

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce and ciphertext
	nonceSize := gcm.NonceSize()
	if len(encryptedPassword) < nonceSize {
		return "", errors.New("encrypted password is too short")
	}

	nonce := encryptedPassword[:nonceSize]
	ciphertext := encryptedPassword[nonceSize:]

	// Decrypt password
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	return string(plaintext), nil
}
