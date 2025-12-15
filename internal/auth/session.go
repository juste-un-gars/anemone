// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	SessionCookieName = "anemone_session"
	SessionDuration   = 2 * time.Hour // 2 hours for better security
)

// Session represents a user session
type Session struct {
	ID        string
	UserID    int
	Username  string
	IsAdmin   bool
	CreatedAt time.Time
	ExpiresAt time.Time
}

// SessionManager manages user sessions
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

var (
	defaultManager *SessionManager
	once           sync.Once
)

// GetSessionManager returns the default session manager
func GetSessionManager() *SessionManager {
	once.Do(func() {
		defaultManager = &SessionManager{
			sessions: make(map[string]*Session),
		}
		// Start cleanup goroutine
		go defaultManager.cleanupExpiredSessions()
	})
	return defaultManager
}

// generateSessionID generates a random session ID
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// CreateSession creates a new session for a user
func (sm *SessionManager) CreateSession(userID int, username string, isAdmin bool) (*Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	now := time.Now()
	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		Username:  username,
		IsAdmin:   isAdmin,
		CreatedAt: now,
		ExpiresAt: now.Add(SessionDuration),
	}

	sm.mu.Lock()
	sm.sessions[sessionID] = session
	sm.mu.Unlock()

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// DeleteSession deletes a session
func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, sessionID)
}

// RenewSession extends the session expiration
func (sm *SessionManager) RenewSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.ExpiresAt = time.Now().Add(SessionDuration)
	return nil
}

// cleanupExpiredSessions removes expired sessions periodically
func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		for id, session := range sm.sessions {
			if now.After(session.ExpiresAt) {
				delete(sm.sessions, id)
			}
		}
		sm.mu.Unlock()
	}
}

// SetSessionCookie sets the session cookie in the response
// Uses SameSite=Strict for maximum CSRF protection (prevents cross-origin requests)
// Uses Secure=true to enforce HTTPS only (works with HSTS header)
func SetSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Enforce HTTPS (required with HSTS header)
		SameSite: http.SameSiteStrictMode, // Strong CSRF protection
		MaxAge:   int(SessionDuration.Seconds()),
	})
}

// ClearSessionCookie clears the session cookie
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// GetSessionFromRequest retrieves the session from the request
func GetSessionFromRequest(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil, fmt.Errorf("no session cookie: %w", err)
	}

	sm := GetSessionManager()
	session, err := sm.GetSession(cookie.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	// Renew session on each request
	sm.RenewSession(cookie.Value)

	return session, nil
}

// IsAuthenticated checks if the request has a valid session
func IsAuthenticated(r *http.Request) bool {
	_, err := GetSessionFromRequest(r)
	return err == nil
}

// IsAdmin checks if the request has a valid admin session
func IsAdmin(r *http.Request) bool {
	session, err := GetSessionFromRequest(r)
	if err != nil {
		return false
	}
	return session.IsAdmin
}
