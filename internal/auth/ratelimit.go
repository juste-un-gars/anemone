// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// This file provides rate limiting for login attempts, tracked by both IP and username.

package auth

import (
	"strings"
	"sync"
	"time"
)

const (
	// MaxLoginAttempts is the maximum number of failed login attempts before lockout
	MaxLoginAttempts = 5
	// LoginWindow is the time window for counting attempts
	LoginWindow = 15 * time.Minute
	// LockoutDuration is the initial lockout after exceeding max attempts
	LockoutDuration = 15 * time.Minute
	// RateLimitCleanupInterval is how often to clean old entries
	RateLimitCleanupInterval = 10 * time.Minute
)

// loginAttempt tracks failed login attempts for an IP
type loginAttempt struct {
	Count    int
	FirstAt  time.Time
	LockedAt time.Time
}

// LoginRateLimiter provides rate limiting for login attempts by IP and username
type LoginRateLimiter struct {
	mu           sync.Mutex
	attempts     map[string]*loginAttempt // keyed by IP
	userAttempts map[string]*loginAttempt // keyed by username (lowercase)
}

var (
	defaultRateLimiter *LoginRateLimiter
	rlOnce             sync.Once
)

// InitLoginRateLimiter initializes the global login rate limiter
func InitLoginRateLimiter() *LoginRateLimiter {
	rlOnce.Do(func() {
		defaultRateLimiter = &LoginRateLimiter{
			attempts:     make(map[string]*loginAttempt),
			userAttempts: make(map[string]*loginAttempt),
		}
		go defaultRateLimiter.cleanup()
	})
	return defaultRateLimiter
}

// GetLoginRateLimiter returns the global login rate limiter
func GetLoginRateLimiter() *LoginRateLimiter {
	return defaultRateLimiter
}

// IsBlocked returns true if the IP is currently locked out, and the remaining duration
func (rl *LoginRateLimiter) IsBlocked(ip string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	a, ok := rl.attempts[ip]
	if !ok {
		return false, 0
	}

	// Check if locked out
	if !a.LockedAt.IsZero() {
		remaining := time.Until(a.LockedAt.Add(LockoutDuration))
		if remaining > 0 {
			return true, remaining
		}
		// Lockout expired, reset
		delete(rl.attempts, ip)
		return false, 0
	}

	// Check if window expired
	if time.Since(a.FirstAt) > LoginWindow {
		delete(rl.attempts, ip)
		return false, 0
	}

	return false, 0
}

// RecordFailure records a failed login attempt. Returns true if now locked out.
func (rl *LoginRateLimiter) RecordFailure(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	a, ok := rl.attempts[ip]
	if !ok {
		rl.attempts[ip] = &loginAttempt{
			Count:   1,
			FirstAt: time.Now(),
		}
		return false
	}

	// Reset if window expired
	if time.Since(a.FirstAt) > LoginWindow {
		rl.attempts[ip] = &loginAttempt{
			Count:   1,
			FirstAt: time.Now(),
		}
		return false
	}

	a.Count++
	if a.Count >= MaxLoginAttempts {
		a.LockedAt = time.Now()
		return true
	}

	return false
}

// RecordSuccess clears the attempt counter for an IP after successful login
func (rl *LoginRateLimiter) RecordSuccess(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.attempts, ip)
}

// RemainingAttempts returns how many attempts are left for an IP
func (rl *LoginRateLimiter) RemainingAttempts(ip string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	a, ok := rl.attempts[ip]
	if !ok {
		return MaxLoginAttempts
	}

	if time.Since(a.FirstAt) > LoginWindow {
		return MaxLoginAttempts
	}

	remaining := MaxLoginAttempts - a.Count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsBlockedUser returns true if a username is currently locked out
func (rl *LoginRateLimiter) IsBlockedUser(username string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	key := strings.ToLower(username)
	a, ok := rl.userAttempts[key]
	if !ok {
		return false, 0
	}

	if !a.LockedAt.IsZero() {
		remaining := time.Until(a.LockedAt.Add(LockoutDuration))
		if remaining > 0 {
			return true, remaining
		}
		delete(rl.userAttempts, key)
		return false, 0
	}

	if time.Since(a.FirstAt) > LoginWindow {
		delete(rl.userAttempts, key)
		return false, 0
	}

	return false, 0
}

// RecordFailureUser records a failed login attempt for a username. Returns true if now locked.
func (rl *LoginRateLimiter) RecordFailureUser(username string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	key := strings.ToLower(username)
	a, ok := rl.userAttempts[key]
	if !ok {
		rl.userAttempts[key] = &loginAttempt{
			Count:   1,
			FirstAt: time.Now(),
		}
		return false
	}

	if time.Since(a.FirstAt) > LoginWindow {
		rl.userAttempts[key] = &loginAttempt{
			Count:   1,
			FirstAt: time.Now(),
		}
		return false
	}

	a.Count++
	if a.Count >= MaxLoginAttempts {
		a.LockedAt = time.Now()
		return true
	}

	return false
}

// RecordSuccessUser clears the attempt counter for a username after successful login
func (rl *LoginRateLimiter) RecordSuccessUser(username string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.userAttempts, strings.ToLower(username))
}

// RemainingAttemptsUser returns how many attempts are left for a username
func (rl *LoginRateLimiter) RemainingAttemptsUser(username string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	key := strings.ToLower(username)
	a, ok := rl.userAttempts[key]
	if !ok {
		return MaxLoginAttempts
	}

	if time.Since(a.FirstAt) > LoginWindow {
		return MaxLoginAttempts
	}

	remaining := MaxLoginAttempts - a.Count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ClearUserLockout clears the rate limit state for a specific username (admin unlock)
func (rl *LoginRateLimiter) ClearUserLockout(username string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.userAttempts, strings.ToLower(username))
}

// GetLockedUsers returns a set of usernames that are currently locked out
func (rl *LoginRateLimiter) GetLockedUsers() map[string]bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	locked := make(map[string]bool)
	now := time.Now()
	for username, a := range rl.userAttempts {
		if !a.LockedAt.IsZero() && now.Before(a.LockedAt.Add(LockoutDuration)) {
			locked[username] = true
		}
	}
	return locked
}

// GetLockedIPs returns a map of IPs currently locked out with their remaining duration
func (rl *LoginRateLimiter) GetLockedIPs() map[string]time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	locked := make(map[string]time.Duration)
	now := time.Now()
	for ip, a := range rl.attempts {
		if !a.LockedAt.IsZero() {
			remaining := a.LockedAt.Add(LockoutDuration).Sub(now)
			if remaining > 0 {
				locked[ip] = remaining
			}
		}
	}
	return locked
}

// ClearIPLockout clears the rate limit state for a specific IP (admin unlock)
func (rl *LoginRateLimiter) ClearIPLockout(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.attempts, ip)
}

// GetLockedUsersWithDuration returns locked usernames with remaining lockout duration
func (rl *LoginRateLimiter) GetLockedUsersWithDuration() map[string]time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	locked := make(map[string]time.Duration)
	now := time.Now()
	for username, a := range rl.userAttempts {
		if !a.LockedAt.IsZero() {
			remaining := a.LockedAt.Add(LockoutDuration).Sub(now)
			if remaining > 0 {
				locked[username] = remaining
			}
		}
	}
	return locked
}

// cleanupMap removes expired entries from a map
func cleanupMap(m map[string]*loginAttempt, now time.Time) {
	for key, a := range m {
		if !a.LockedAt.IsZero() && now.After(a.LockedAt.Add(LockoutDuration)) {
			delete(m, key)
			continue
		}
		if a.LockedAt.IsZero() && now.Sub(a.FirstAt) > LoginWindow {
			delete(m, key)
		}
	}
}

// cleanup periodically removes expired entries
func (rl *LoginRateLimiter) cleanup() {
	ticker := time.NewTicker(RateLimitCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		cleanupMap(rl.attempts, now)
		cleanupMap(rl.userAttempts, now)
		rl.mu.Unlock()
	}
}
