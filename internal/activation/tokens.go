// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package activation handles user account activation tokens and email verification.
package activation

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"
)

const (
	// TokenValidityDuration is how long activation tokens are valid (24 hours)
	TokenValidityDuration = 24 * time.Hour
)

// Token represents an activation token
type Token struct {
	ID        int
	Token     string
	UserID    int
	Username  string
	Email     string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

// GenerateToken generates a random activation token
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// CreateActivationToken creates a new activation token for a user
func CreateActivationToken(db *sql.DB, userID int, username, email string) (*Token, error) {
	tokenString, err := GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(TokenValidityDuration)

	result, err := db.Exec(`
		INSERT INTO activation_tokens (token, user_id, username, email, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, tokenString, userID, username, email, now, expiresAt)

	if err != nil {
		return nil, fmt.Errorf("failed to insert token: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get token ID: %w", err)
	}

	token := &Token{
		ID:        int(id),
		Token:     tokenString,
		UserID:    userID,
		Username:  username,
		Email:     email,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	return token, nil
}

// GetTokenByString retrieves a token by its string value
func GetTokenByString(db *sql.DB, tokenString string) (*Token, error) {
	token := &Token{}
	var usedAt sql.NullTime

	err := db.QueryRow(`
		SELECT id, token, user_id, username, email, created_at, expires_at, used_at
		FROM activation_tokens
		WHERE token = ?
	`, tokenString).Scan(
		&token.ID, &token.Token, &token.UserID, &token.Username, &token.Email,
		&token.CreatedAt, &token.ExpiresAt, &usedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	if usedAt.Valid {
		token.UsedAt = &usedAt.Time
	}

	return token, nil
}

// IsValid checks if the token is valid (not expired and not used)
func (t *Token) IsValid() bool {
	if t.UsedAt != nil {
		return false // Already used
	}
	return time.Now().Before(t.ExpiresAt)
}

// MarkAsUsed atomically marks the token as used.
// Returns false if the token was already consumed by another request (race condition protection).
func (t *Token) MarkAsUsed(db *sql.DB) (bool, error) {
	now := time.Now()
	result, err := db.Exec("UPDATE activation_tokens SET used_at = ? WHERE id = ? AND used_at IS NULL", now, t.ID)
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
	t.UsedAt = &now
	return true, nil
}

// DeleteExpiredTokens removes expired tokens from the database
func DeleteExpiredTokens(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM activation_tokens WHERE expires_at < ?", time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired tokens: %w", err)
	}
	return nil
}

// GetPendingTokens retrieves all pending (not used) tokens
func GetPendingTokens(db *sql.DB) ([]*Token, error) {
	rows, err := db.Query(`
		SELECT id, token, user_id, username, email, created_at, expires_at
		FROM activation_tokens
		WHERE used_at IS NULL
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
			&token.ID, &token.Token, &token.UserID, &token.Username, &token.Email,
			&token.CreatedAt, &token.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan token: %w", err)
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}
