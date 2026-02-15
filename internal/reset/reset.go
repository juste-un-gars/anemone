// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package reset handles password reset token generation and validation.
package reset

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"
)

const (
	// TokenValidityDuration is how long password reset tokens are valid (24 hours)
	TokenValidityDuration = 24 * time.Hour
)

// Token represents a password reset token
type Token struct {
	ID        int
	UserID    int
	Token     string
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}

// GenerateToken generates a random password reset token
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// CreatePasswordResetToken creates a new password reset token for a user
func CreatePasswordResetToken(db *sql.DB, userID int) (*Token, error) {
	tokenString, err := GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(TokenValidityDuration)

	result, err := db.Exec(`
		INSERT INTO password_reset_tokens (user_id, token, expires_at, used, created_at)
		VALUES (?, ?, ?, 0, ?)
	`, userID, tokenString, expiresAt, now)

	if err != nil {
		return nil, fmt.Errorf("failed to insert token: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get token ID: %w", err)
	}

	token := &Token{
		ID:        int(id),
		UserID:    userID,
		Token:     tokenString,
		ExpiresAt: expiresAt,
		Used:      false,
		CreatedAt: now,
	}

	return token, nil
}

// GetTokenByString retrieves a token by its string value
func GetTokenByString(db *sql.DB, tokenString string) (*Token, error) {
	token := &Token{}

	err := db.QueryRow(`
		SELECT id, user_id, token, expires_at, used, created_at
		FROM password_reset_tokens
		WHERE token = ?
	`, tokenString).Scan(
		&token.ID, &token.UserID, &token.Token, &token.ExpiresAt, &token.Used, &token.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	return token, nil
}

// IsValid checks if the token is valid (not expired and not used)
func (t *Token) IsValid() bool {
	if t.Used {
		return false // Already used
	}
	return time.Now().Before(t.ExpiresAt)
}

// MarkAsUsed atomically marks the token as used.
// Returns false if the token was already consumed by another request (race condition protection).
func (t *Token) MarkAsUsed(db *sql.DB) (bool, error) {
	result, err := db.Exec("UPDATE password_reset_tokens SET used = 1 WHERE id = ? AND used = 0", t.ID)
	if err != nil {
		return false, fmt.Errorf("failed to mark token as used: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return false, nil // Already consumed by another request
	}
	t.Used = true
	return true, nil
}

// DeleteExpiredTokens removes expired tokens from the database
func DeleteExpiredTokens(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM password_reset_tokens WHERE expires_at < ?", time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired tokens: %w", err)
	}
	return nil
}

// GetPendingTokens retrieves all pending (not used) tokens
func GetPendingTokens(db *sql.DB) ([]*Token, error) {
	rows, err := db.Query(`
		SELECT id, user_id, token, expires_at, used, created_at
		FROM password_reset_tokens
		WHERE used = 0
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*Token
	for rows.Next() {
		token := &Token{}
		err := rows.Scan(
			&token.ID, &token.UserID, &token.Token, &token.ExpiresAt, &token.Used, &token.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan token: %w", err)
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}
