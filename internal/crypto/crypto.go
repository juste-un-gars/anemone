// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package crypto provides cryptographic utilities including AES-256-GCM encryption,
// bcrypt password hashing, and secure key generation.
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

// HashPassword hashes a password using bcrypt with cost 12
// Cost 12 provides strong protection against brute-force attacks while maintaining reasonable performance
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
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

// EncryptStream encrypts data from reader and writes to writer using AES-256-GCM with chunking
// The encryption key must be base64-encoded 32-byte key
// Format: [magic "AECG" 4B][version 4B][chunk_size 4B][nonce 12B][encrypted_chunk + tag][...]
// Uses 128MB chunks to prevent OOM on systems with limited RAM (2GB)
func EncryptStream(reader io.Reader, writer io.Writer, encryptionKey string) error {
	const chunkSize = 128 * 1024 * 1024 // 128MB chunks

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

	// Write magic header and version
	magic := []byte("AECG") // Anemone Encrypted Chunked GCM
	if _, err := writer.Write(magic); err != nil {
		return fmt.Errorf("failed to write magic header: %w", err)
	}

	version := uint32(1)
	versionBytes := make([]byte, 4)
	versionBytes[0] = byte(version >> 24)
	versionBytes[1] = byte(version >> 16)
	versionBytes[2] = byte(version >> 8)
	versionBytes[3] = byte(version)
	if _, err := writer.Write(versionBytes); err != nil {
		return fmt.Errorf("failed to write version: %w", err)
	}

	// Process file in chunks
	buffer := make([]byte, chunkSize)
	for {
		// Read chunk
		n, err := io.ReadFull(reader, buffer)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return fmt.Errorf("failed to read chunk: %w", err)
		}
		if n == 0 {
			break // End of file
		}

		// Generate random nonce for this chunk
		nonce := make([]byte, gcm.NonceSize())
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			return fmt.Errorf("failed to generate nonce: %w", err)
		}

		// Encrypt chunk
		ciphertext := gcm.Seal(nil, nonce, buffer[:n], nil)

		// Write chunk size (4 bytes)
		chunkSizeBytes := make([]byte, 4)
		chunkLen := uint32(len(ciphertext))
		chunkSizeBytes[0] = byte(chunkLen >> 24)
		chunkSizeBytes[1] = byte(chunkLen >> 16)
		chunkSizeBytes[2] = byte(chunkLen >> 8)
		chunkSizeBytes[3] = byte(chunkLen)
		if _, err := writer.Write(chunkSizeBytes); err != nil {
			return fmt.Errorf("failed to write chunk size: %w", err)
		}

		// Write nonce
		if _, err := writer.Write(nonce); err != nil {
			return fmt.Errorf("failed to write nonce: %w", err)
		}

		// Write encrypted chunk
		if _, err := writer.Write(ciphertext); err != nil {
			return fmt.Errorf("failed to write encrypted chunk: %w", err)
		}

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
	}

	return nil
}

// DecryptStream decrypts data from reader and writes to writer using AES-256-GCM
// The encryption key must be base64-encoded 32-byte key
// Supports both chunked format (new) and legacy format (old) for backward compatibility
// New format: [magic "AECG" 4B][version 4B][chunk_size 4B][nonce 12B][encrypted_chunk + tag][...]
// Legacy format: [nonce (12 bytes)][encrypted data with auth tag]
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

	// Peek at first 4 bytes to detect format
	magic := make([]byte, 4)
	if _, err := io.ReadFull(reader, magic); err != nil {
		return fmt.Errorf("failed to read magic/nonce header: %w", err)
	}

	// Check if this is the new chunked format
	if string(magic) == "AECG" {
		return decryptStreamChunked(reader, writer, gcm)
	}

	// Legacy format - magic bytes are actually first 4 bytes of nonce
	return decryptStreamLegacy(reader, writer, gcm, magic)
}

// decryptStreamChunked handles the new chunked format
func decryptStreamChunked(reader io.Reader, writer io.Writer, gcm cipher.AEAD) error {
	// Read version
	versionBytes := make([]byte, 4)
	if _, err := io.ReadFull(reader, versionBytes); err != nil {
		return fmt.Errorf("failed to read version: %w", err)
	}
	version := uint32(versionBytes[0])<<24 | uint32(versionBytes[1])<<16 | uint32(versionBytes[2])<<8 | uint32(versionBytes[3])
	if version != 1 {
		return fmt.Errorf("unsupported encryption version: %d", version)
	}

	// Process chunks
	for {
		// Read chunk size
		chunkSizeBytes := make([]byte, 4)
		_, err := io.ReadFull(reader, chunkSizeBytes)
		if err == io.EOF {
			break // End of file
		}
		if err != nil {
			return fmt.Errorf("failed to read chunk size: %w", err)
		}
		chunkSize := uint32(chunkSizeBytes[0])<<24 | uint32(chunkSizeBytes[1])<<16 | uint32(chunkSizeBytes[2])<<8 | uint32(chunkSizeBytes[3])

		// Read nonce
		nonce := make([]byte, gcm.NonceSize())
		if _, err := io.ReadFull(reader, nonce); err != nil {
			return fmt.Errorf("failed to read nonce: %w", err)
		}

		// Read encrypted chunk
		ciphertext := make([]byte, chunkSize)
		if _, err := io.ReadFull(reader, ciphertext); err != nil {
			return fmt.Errorf("failed to read encrypted chunk: %w", err)
		}

		// Decrypt chunk
		plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return fmt.Errorf("failed to decrypt chunk: %w (invalid key or corrupted data)", err)
		}

		// Write decrypted chunk
		if _, err := writer.Write(plaintext); err != nil {
			return fmt.Errorf("failed to write decrypted chunk: %w", err)
		}
	}

	return nil
}

// decryptStreamLegacy handles the old non-chunked format (backward compatibility)
func decryptStreamLegacy(reader io.Reader, writer io.Writer, gcm cipher.AEAD, noncePrefix []byte) error {
	// Read rest of nonce (we already read first 4 bytes)
	nonceRest := make([]byte, gcm.NonceSize()-4)
	if _, err := io.ReadFull(reader, nonceRest); err != nil {
		return fmt.Errorf("failed to read nonce: %w", err)
	}
	nonce := append(noncePrefix, nonceRest...)

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
