// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// This file provides IP-based rate limiting for login attempts.

package auth

import (
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

// LoginRateLimiter provides IP-based rate limiting for login attempts
type LoginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string]*loginAttempt
}

var (
	defaultRateLimiter *LoginRateLimiter
	rlOnce             sync.Once
)

// InitLoginRateLimiter initializes the global login rate limiter
func InitLoginRateLimiter() *LoginRateLimiter {
	rlOnce.Do(func() {
		defaultRateLimiter = &LoginRateLimiter{
			attempts: make(map[string]*loginAttempt),
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

// cleanup periodically removes expired entries
func (rl *LoginRateLimiter) cleanup() {
	ticker := time.NewTicker(RateLimitCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, a := range rl.attempts {
			// Remove if lockout expired
			if !a.LockedAt.IsZero() && now.After(a.LockedAt.Add(LockoutDuration)) {
				delete(rl.attempts, ip)
				continue
			}
			// Remove if window expired and not locked
			if a.LockedAt.IsZero() && now.Sub(a.FirstAt) > LoginWindow {
				delete(rl.attempts, ip)
			}
		}
		rl.mu.Unlock()
	}
}
