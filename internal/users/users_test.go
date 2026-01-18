// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package users

import (
	"strings"
	"testing"
	"time"

	"github.com/juste-un-gars/anemone/internal/crypto"
)

func TestValidateUsername(t *testing.T) {
	testCases := []struct {
		name     string
		username string
		wantErr  bool
		errMsg   string
	}{
		// Valid usernames
		{"valid_simple", "john", false, ""},
		{"valid_with_numbers", "john123", false, ""},
		{"valid_with_underscore", "john_doe", false, ""},
		{"valid_with_hyphen", "john-doe", false, ""},
		{"valid_mixed", "John_Doe-123", false, ""},
		{"valid_min_length", "ab", false, ""},
		{"valid_max_length", strings.Repeat("a", 32), false, ""},

		// Invalid usernames
		{"empty", "", true, "cannot be empty"},
		{"too_short", "a", true, "at least 2 characters"},
		{"too_long", strings.Repeat("a", 33), true, "not exceed 32"},
		{"with_space", "john doe", true, "only contain"},
		{"with_special", "john@doe", true, "only contain"},
		{"with_dot", "john.doe", true, "only contain"},
		{"command_injection", "john;rm -rf /", true, "only contain"},
		{"path_traversal", "../etc/passwd", true, "only contain"},
		{"shell_expansion", "$(whoami)", true, "only contain"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateUsername(tc.username)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ValidateUsername(%q) should return error", tc.username)
				} else if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("Error should contain %q, got %q", tc.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("ValidateUsername(%q) returned unexpected error: %v", tc.username, err)
				}
			}
		})
	}
}

func TestUserIsActivated(t *testing.T) {
	// User with nil ActivatedAt is not activated
	userNotActivated := &User{
		ID:          1,
		Username:    "test",
		ActivatedAt: nil,
	}
	if userNotActivated.IsActivated() {
		t.Error("User with nil ActivatedAt should not be activated")
	}

	// User with ActivatedAt set is activated
	now := time.Now()
	userActivated := &User{
		ID:          2,
		Username:    "test2",
		ActivatedAt: &now,
	}
	if !userActivated.IsActivated() {
		t.Error("User with ActivatedAt set should be activated")
	}
}

func TestUserCheckPassword(t *testing.T) {
	// Create a user with a known password hash
	password := "SecurePassword123!"
	hash, err := crypto.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	user := &User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: hash,
	}

	// Correct password should return true
	if !user.CheckPassword(password) {
		t.Error("CheckPassword should return true for correct password")
	}

	// Wrong password should return false
	if user.CheckPassword("WrongPassword") {
		t.Error("CheckPassword should return false for wrong password")
	}

	// Empty password should return false
	if user.CheckPassword("") {
		t.Error("CheckPassword should return false for empty password")
	}
}

func TestUserCheckPasswordWithEmptyHash(t *testing.T) {
	// User with empty password hash (pending activation)
	user := &User{
		ID:           1,
		Username:     "pending",
		PasswordHash: "",
	}

	// Any password should fail for user with empty hash
	if user.CheckPassword("anypassword") {
		t.Error("CheckPassword should return false when hash is empty")
	}
}

// TestValidateUsernameSecurityCases tests security-specific username validation
func TestValidateUsernameSecurityCases(t *testing.T) {
	securityCases := []string{
		// Command injection attempts
		"; rm -rf /",
		"| cat /etc/passwd",
		"&& whoami",
		"`id`",
		"$(id)",

		// Path traversal
		"../",
		"..\\",
		"....//",

		// Special characters that could cause issues
		"user\x00name", // null byte
		"user\nname",   // newline
		"user\tname",   // tab

		// Unicode tricks
		"user\u0000name", // unicode null
	}

	for _, username := range securityCases {
		err := ValidateUsername(username)
		if err == nil {
			t.Errorf("Security case %q should be rejected", username)
		}
	}
}
