// Package adminverify provides password verification for destructive admin operations.
//
// This package implements a security layer that requires admin password re-entry
// before executing destructive operations like pool destruction, disk formatting, etc.
// It includes rate limiting to prevent brute force attacks.
package adminverify

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/juste-un-gars/anemone/internal/users"
)

// Token represents a temporary verification token
type Token struct {
	Value     string
	UserID    int
	ExpiresAt time.Time
	Used      bool
}

// RateLimitEntry tracks failed attempts per IP
type RateLimitEntry struct {
	Attempts  int
	ResetAt   time.Time
}

// Verifier handles admin password verification
type Verifier struct {
	mu           sync.RWMutex
	tokens       map[string]*Token
	rateLimits   map[string]*RateLimitEntry
	tokenTTL     time.Duration
	maxAttempts  int
	rateLimitTTL time.Duration
}

// Config holds verifier configuration
type Config struct {
	TokenTTL     time.Duration // How long tokens are valid (default: 5 minutes)
	MaxAttempts  int           // Max attempts per minute per IP (default: 5)
	RateLimitTTL time.Duration // How long rate limit window lasts (default: 1 minute)
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		TokenTTL:     5 * time.Minute,
		MaxAttempts:  5,
		RateLimitTTL: 1 * time.Minute,
	}
}

// Global verifier instance
var (
	globalVerifier *Verifier
	once           sync.Once
)

// GetVerifier returns the global verifier instance
func GetVerifier() *Verifier {
	once.Do(func() {
		globalVerifier = NewVerifier(DefaultConfig())
	})
	return globalVerifier
}

// NewVerifier creates a new verifier with the given config
func NewVerifier(cfg Config) *Verifier {
	v := &Verifier{
		tokens:       make(map[string]*Token),
		rateLimits:   make(map[string]*RateLimitEntry),
		tokenTTL:     cfg.TokenTTL,
		maxAttempts:  cfg.MaxAttempts,
		rateLimitTTL: cfg.RateLimitTTL,
	}
	// Start cleanup goroutine
	go v.cleanupLoop()
	return v
}

// cleanupLoop periodically removes expired tokens and rate limit entries
func (v *Verifier) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		v.cleanup()
	}
}

// cleanup removes expired entries
func (v *Verifier) cleanup() {
	v.mu.Lock()
	defer v.mu.Unlock()

	now := time.Now()

	// Clean up expired tokens
	for key, token := range v.tokens {
		if now.After(token.ExpiresAt) || token.Used {
			delete(v.tokens, key)
		}
	}

	// Clean up expired rate limits
	for key, entry := range v.rateLimits {
		if now.After(entry.ResetAt) {
			delete(v.rateLimits, key)
		}
	}
}

// CheckRateLimit checks if an IP is rate limited
// Returns error if rate limited, nil otherwise
func (v *Verifier) CheckRateLimit(ip string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	entry, exists := v.rateLimits[ip]
	now := time.Now()

	if !exists {
		v.rateLimits[ip] = &RateLimitEntry{
			Attempts: 0,
			ResetAt:  now.Add(v.rateLimitTTL),
		}
		return nil
	}

	// Reset if window has passed
	if now.After(entry.ResetAt) {
		entry.Attempts = 0
		entry.ResetAt = now.Add(v.rateLimitTTL)
		return nil
	}

	// Check if rate limited
	if entry.Attempts >= v.maxAttempts {
		remaining := entry.ResetAt.Sub(now).Seconds()
		return fmt.Errorf("rate limited: too many attempts, try again in %.0f seconds", remaining)
	}

	return nil
}

// RecordAttempt records a failed attempt for rate limiting
func (v *Verifier) RecordAttempt(ip string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	entry, exists := v.rateLimits[ip]
	if !exists {
		v.rateLimits[ip] = &RateLimitEntry{
			Attempts: 1,
			ResetAt:  time.Now().Add(v.rateLimitTTL),
		}
		return
	}

	entry.Attempts++
}

// VerifyPassword verifies admin password and returns a token if valid
func (v *Verifier) VerifyPassword(db *sql.DB, userID int, password, ip string) (string, error) {
	// Check rate limit first
	if err := v.CheckRateLimit(ip); err != nil {
		return "", err
	}

	// Get user from database
	user, err := users.GetByID(db, userID)
	if err != nil {
		v.RecordAttempt(ip)
		return "", fmt.Errorf("user not found")
	}

	// Verify user is admin
	if !user.IsAdmin {
		v.RecordAttempt(ip)
		return "", fmt.Errorf("not an administrator")
	}

	// Verify password
	if !user.CheckPassword(password) {
		v.RecordAttempt(ip)
		return "", fmt.Errorf("incorrect password")
	}

	// Generate token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	tokenValue := hex.EncodeToString(tokenBytes)

	// Store token
	v.mu.Lock()
	v.tokens[tokenValue] = &Token{
		Value:     tokenValue,
		UserID:    userID,
		ExpiresAt: time.Now().Add(v.tokenTTL),
		Used:      false,
	}
	v.mu.Unlock()

	return tokenValue, nil
}

// ValidateToken validates a verification token
// Tokens are single-use and expire after the TTL
func (v *Verifier) ValidateToken(tokenValue string, userID int) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	token, exists := v.tokens[tokenValue]
	if !exists {
		return fmt.Errorf("invalid or expired token")
	}

	if time.Now().After(token.ExpiresAt) {
		delete(v.tokens, tokenValue)
		return fmt.Errorf("token expired")
	}

	if token.Used {
		return fmt.Errorf("token already used")
	}

	if token.UserID != userID {
		return fmt.Errorf("token does not belong to this user")
	}

	// Mark token as used (single-use)
	token.Used = true

	return nil
}

// InvalidateToken invalidates a token
func (v *Verifier) InvalidateToken(tokenValue string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.tokens, tokenValue)
}

// GetRemainingAttempts returns how many attempts are left for an IP
func (v *Verifier) GetRemainingAttempts(ip string) int {
	v.mu.RLock()
	defer v.mu.RUnlock()

	entry, exists := v.rateLimits[ip]
	if !exists {
		return v.maxAttempts
	}

	if time.Now().After(entry.ResetAt) {
		return v.maxAttempts
	}

	remaining := v.maxAttempts - entry.Attempts
	if remaining < 0 {
		return 0
	}
	return remaining
}
