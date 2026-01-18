// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package crypto

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateEncryptionKey(t *testing.T) {
	key, err := GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("GenerateEncryptionKey failed: %v", err)
	}

	// Key should be base64 encoded
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		t.Fatalf("Key is not valid base64: %v", err)
	}

	// Key should be 32 bytes when decoded
	if len(decoded) != 32 {
		t.Errorf("Expected 32 bytes, got %d", len(decoded))
	}

	// Generate another key - should be different (random)
	key2, err := GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("GenerateEncryptionKey failed: %v", err)
	}
	if key == key2 {
		t.Error("Two generated keys should not be identical")
	}
}

func TestHashKey(t *testing.T) {
	key := "test-encryption-key"
	hash1 := HashKey(key)
	hash2 := HashKey(key)

	// Same input should produce same hash
	if hash1 != hash2 {
		t.Error("HashKey should be deterministic")
	}

	// Different input should produce different hash
	hash3 := HashKey("different-key")
	if hash1 == hash3 {
		t.Error("Different keys should produce different hashes")
	}

	// Hash should be base64 encoded
	_, err := base64.StdEncoding.DecodeString(hash1)
	if err != nil {
		t.Errorf("Hash is not valid base64: %v", err)
	}
}

func TestEncryptDecryptKey(t *testing.T) {
	plainKey := "my-secret-encryption-key"
	masterKey := "master-password-123"

	// Encrypt the key
	encrypted, err := EncryptKey(plainKey, masterKey)
	if err != nil {
		t.Fatalf("EncryptKey failed: %v", err)
	}

	// Encrypted should be different from plain
	if encrypted == plainKey {
		t.Error("Encrypted key should differ from plain key")
	}

	// Decrypt the key
	decrypted, err := DecryptKey(encrypted, masterKey)
	if err != nil {
		t.Fatalf("DecryptKey failed: %v", err)
	}

	// Decrypted should match original
	if decrypted != plainKey {
		t.Errorf("Expected %q, got %q", plainKey, decrypted)
	}

	// Wrong master key should fail
	_, err = DecryptKey(encrypted, "wrong-master-key")
	if err == nil {
		t.Error("DecryptKey should fail with wrong master key")
	}
}

func TestEncryptKeyDifferentOutputs(t *testing.T) {
	plainKey := "test-key"
	masterKey := "master"

	// Encrypting same key twice should produce different ciphertexts (due to random nonce)
	enc1, _ := EncryptKey(plainKey, masterKey)
	enc2, _ := EncryptKey(plainKey, masterKey)

	if enc1 == enc2 {
		t.Error("Same plaintext should produce different ciphertexts due to random nonce")
	}

	// But both should decrypt to same value
	dec1, _ := DecryptKey(enc1, masterKey)
	dec2, _ := DecryptKey(enc2, masterKey)

	if dec1 != dec2 {
		t.Error("Both ciphertexts should decrypt to same plaintext")
	}
}

func TestHashPassword(t *testing.T) {
	password := "SecurePassword123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Hash should start with bcrypt prefix
	if !strings.HasPrefix(hash, "$2") {
		t.Error("Hash should be bcrypt format (starts with $2)")
	}

	// Hash should be different from password
	if hash == password {
		t.Error("Hash should differ from password")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "SecurePassword123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Correct password should match
	if !CheckPassword(password, hash) {
		t.Error("CheckPassword should return true for correct password")
	}

	// Wrong password should not match
	if CheckPassword("WrongPassword", hash) {
		t.Error("CheckPassword should return false for wrong password")
	}

	// Empty password should not match
	if CheckPassword("", hash) {
		t.Error("CheckPassword should return false for empty password")
	}
}

func TestEncryptDecryptStream(t *testing.T) {
	// Generate a valid encryption key
	key, err := GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("GenerateEncryptionKey failed: %v", err)
	}

	testCases := []struct {
		name string
		data string
	}{
		{"empty", ""},
		{"small", "Hello, World!"},
		{"medium", strings.Repeat("Test data. ", 1000)},
		{"with_unicode", "Données chiffrées avec des accents et émojis 🔐"},
		{"binary_like", string([]byte{0, 1, 2, 255, 254, 253})},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			var encryptedBuf bytes.Buffer
			err := EncryptStream(strings.NewReader(tc.data), &encryptedBuf, key)
			if err != nil {
				t.Fatalf("EncryptStream failed: %v", err)
			}

			// Encrypted data should differ from original (unless empty)
			if len(tc.data) > 0 && encryptedBuf.String() == tc.data {
				t.Error("Encrypted data should differ from original")
			}

			// Decrypt
			var decryptedBuf bytes.Buffer
			err = DecryptStream(&encryptedBuf, &decryptedBuf, key)
			if err != nil {
				t.Fatalf("DecryptStream failed: %v", err)
			}

			// Decrypted should match original
			if decryptedBuf.String() != tc.data {
				t.Errorf("Decryption mismatch: expected %d bytes, got %d bytes",
					len(tc.data), decryptedBuf.Len())
			}
		})
	}
}

func TestEncryptStreamInvalidKey(t *testing.T) {
	var buf bytes.Buffer

	// Invalid base64 key
	err := EncryptStream(strings.NewReader("test"), &buf, "not-valid-base64!!!")
	if err == nil {
		t.Error("EncryptStream should fail with invalid base64 key")
	}

	// Key too short
	shortKey := base64.StdEncoding.EncodeToString([]byte("short"))
	err = EncryptStream(strings.NewReader("test"), &buf, shortKey)
	if err == nil {
		t.Error("EncryptStream should fail with key that's not 32 bytes")
	}
}

func TestDecryptStreamWrongKey(t *testing.T) {
	key1, _ := GenerateEncryptionKey()
	key2, _ := GenerateEncryptionKey()

	// Encrypt with key1
	var encryptedBuf bytes.Buffer
	err := EncryptStream(strings.NewReader("secret data"), &encryptedBuf, key1)
	if err != nil {
		t.Fatalf("EncryptStream failed: %v", err)
	}

	// Try to decrypt with key2 - should fail
	var decryptedBuf bytes.Buffer
	err = DecryptStream(bytes.NewReader(encryptedBuf.Bytes()), &decryptedBuf, key2)
	if err == nil {
		t.Error("DecryptStream should fail with wrong key")
	}
}

func TestEncryptDecryptPassword(t *testing.T) {
	password := "MySecretSMBPassword123!"
	masterKey := "server-master-key"

	// Encrypt
	encrypted, err := EncryptPassword(password, masterKey)
	if err != nil {
		t.Fatalf("EncryptPassword failed: %v", err)
	}

	// Encrypted should be non-empty
	if len(encrypted) == 0 {
		t.Error("Encrypted password should not be empty")
	}

	// Decrypt
	decrypted, err := DecryptPassword(encrypted, masterKey)
	if err != nil {
		t.Fatalf("DecryptPassword failed: %v", err)
	}

	// Should match original
	if decrypted != password {
		t.Errorf("Expected %q, got %q", password, decrypted)
	}
}

func TestEncryptPasswordErrors(t *testing.T) {
	// Empty password
	_, err := EncryptPassword("", "masterkey")
	if err == nil {
		t.Error("EncryptPassword should fail with empty password")
	}

	// Empty master key
	_, err = EncryptPassword("password", "")
	if err == nil {
		t.Error("EncryptPassword should fail with empty master key")
	}
}

func TestDecryptPasswordErrors(t *testing.T) {
	// Empty encrypted password
	_, err := DecryptPassword([]byte{}, "masterkey")
	if err == nil {
		t.Error("DecryptPassword should fail with empty encrypted password")
	}

	// Empty master key
	_, err = DecryptPassword([]byte{1, 2, 3}, "")
	if err == nil {
		t.Error("DecryptPassword should fail with empty master key")
	}

	// Too short encrypted password
	_, err = DecryptPassword([]byte{1, 2, 3}, "masterkey")
	if err == nil {
		t.Error("DecryptPassword should fail with too short encrypted password")
	}

	// Wrong master key
	encrypted, _ := EncryptPassword("password", "correct-key")
	_, err = DecryptPassword(encrypted, "wrong-key")
	if err == nil {
		t.Error("DecryptPassword should fail with wrong master key")
	}
}

func TestDecryptKeyErrors(t *testing.T) {
	// Invalid base64
	_, err := DecryptKey("not-valid-base64!!!", "masterkey")
	if err == nil {
		t.Error("DecryptKey should fail with invalid base64")
	}

	// Too short ciphertext
	shortCiphertext := base64.StdEncoding.EncodeToString([]byte{1, 2, 3})
	_, err = DecryptKey(shortCiphertext, "masterkey")
	if err == nil {
		t.Error("DecryptKey should fail with too short ciphertext")
	}
}
