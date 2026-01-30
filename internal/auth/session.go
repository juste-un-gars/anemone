// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"github.com/juste-un-gars/anemone/internal/logger"
	"net/http"
	"sync"
	"time"
)

const (
	SessionCookieName       = "anemone_session"
	SessionDuration         = 2 * time.Hour  // Normal session: 2 hours
	RememberMeDuration      = 30 * 24 * time.Hour // Remember me: 30 days
	SessionCleanupInterval  = 1 * time.Hour
)

// Session represents a user session
type Session struct {
	ID           string
	UserID       int
	Username     string
	IsAdmin      bool
	RememberMe   bool
	CreatedAt    time.Time
	ExpiresAt    time.Time
	LastActivity time.Time
	UserAgent    string
	IPAddress    string
}

// SessionManager manages user sessions with database persistence
type SessionManager struct {
	db *sql.DB
	mu sync.RWMutex
}

var (
	defaultManager *SessionManager
	once           sync.Once
)

// InitSessionManager initializes the session manager with a database connection
func InitSessionManager(db *sql.DB) *SessionManager {
	once.Do(func() {
		defaultManager = &SessionManager{
			db: db,
		}
		// Start cleanup goroutine
		go defaultManager.cleanupExpiredSessions()
	})
	return defaultManager
}

// GetSessionManager returns the default session manager
// Note: InitSessionManager must be called first
func GetSessionManager() *SessionManager {
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
	return sm.CreateSessionWithOptions(userID, username, isAdmin, false, "", "")
}

// CreateSessionWithOptions creates a new session with additional options
func (sm *SessionManager) CreateSessionWithOptions(userID int, username string, isAdmin bool, rememberMe bool, userAgent string, ipAddress string) (*Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	now := time.Now()
	duration := SessionDuration
	if rememberMe {
		duration = RememberMeDuration
	}

	session := &Session{
		ID:           sessionID,
		UserID:       userID,
		Username:     username,
		IsAdmin:      isAdmin,
		RememberMe:   rememberMe,
		CreatedAt:    now,
		ExpiresAt:    now.Add(duration),
		LastActivity: now,
		UserAgent:    userAgent,
		IPAddress:    ipAddress,
	}

	// Insert into database
	_, err = sm.db.Exec(`
		INSERT INTO sessions (id, user_id, username, is_admin, remember_me, created_at, expires_at, last_activity, user_agent, ip_address)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, session.ID, session.UserID, session.Username, session.IsAdmin, session.RememberMe,
		session.CreatedAt, session.ExpiresAt, session.LastActivity, session.UserAgent, session.IPAddress)

	if err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	var session Session
	var rememberMe bool

	err := sm.db.QueryRow(`
		SELECT id, user_id, username, is_admin, remember_me, created_at, expires_at, last_activity,
		       COALESCE(user_agent, ''), COALESCE(ip_address, '')
		FROM sessions
		WHERE id = ?
	`, sessionID).Scan(
		&session.ID, &session.UserID, &session.Username, &session.IsAdmin, &rememberMe,
		&session.CreatedAt, &session.ExpiresAt, &session.LastActivity,
		&session.UserAgent, &session.IPAddress,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	session.RememberMe = rememberMe

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		// Delete expired session
		sm.DeleteSession(sessionID)
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

// DeleteSession deletes a session
func (sm *SessionManager) DeleteSession(sessionID string) {
	_, err := sm.db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	if err != nil {
		logger.Info("Failed to delete session: %v", err)
	}
}

// DeleteUserSessions deletes all sessions for a user
func (sm *SessionManager) DeleteUserSessions(userID int) error {
	_, err := sm.db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

// RenewSession extends the session expiration
func (sm *SessionManager) RenewSession(sessionID string) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	duration := SessionDuration
	if session.RememberMe {
		duration = RememberMeDuration
	}

	newExpiry := time.Now().Add(duration)
	_, err = sm.db.Exec(`
		UPDATE sessions
		SET expires_at = ?, last_activity = CURRENT_TIMESTAMP
		WHERE id = ?
	`, newExpiry, sessionID)

	return err
}

// UpdateLastActivity updates the last activity timestamp
func (sm *SessionManager) UpdateLastActivity(sessionID string) error {
	_, err := sm.db.Exec(`
		UPDATE sessions
		SET last_activity = CURRENT_TIMESTAMP
		WHERE id = ?
	`, sessionID)
	return err
}

// GetUserSessions returns all active sessions for a user
func (sm *SessionManager) GetUserSessions(userID int) ([]*Session, error) {
	rows, err := sm.db.Query(`
		SELECT id, user_id, username, is_admin, remember_me, created_at, expires_at, last_activity,
		       COALESCE(user_agent, ''), COALESCE(ip_address, '')
		FROM sessions
		WHERE user_id = ? AND expires_at > CURRENT_TIMESTAMP
		ORDER BY last_activity DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var s Session
		err := rows.Scan(
			&s.ID, &s.UserID, &s.Username, &s.IsAdmin, &s.RememberMe,
			&s.CreatedAt, &s.ExpiresAt, &s.LastActivity,
			&s.UserAgent, &s.IPAddress,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, &s)
	}

	return sessions, nil
}

// cleanupExpiredSessions removes expired sessions periodically
func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(SessionCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		result, err := sm.db.Exec("DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP")
		if err != nil {
			logger.Info("Failed to cleanup expired sessions: %v", err)
			continue
		}
		if count, _ := result.RowsAffected(); count > 0 {
			logger.Info("Cleaned up %d expired sessions", count)
		}
	}
}

// SetSessionCookie sets the session cookie in the response
func SetSessionCookie(w http.ResponseWriter, sessionID string, rememberMe bool) {
	maxAge := int(SessionDuration.Seconds())
	if rememberMe {
		maxAge = int(RememberMeDuration.Seconds())
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
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
	if sm == nil {
		return nil, fmt.Errorf("session manager not initialized")
	}

	session, err := sm.GetSession(cookie.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	// Update last activity (non-blocking)
	go sm.UpdateLastActivity(cookie.Value)

	// Renew session on each request for non-remember-me sessions
	// For remember-me sessions, only renew if more than 1 day until expiry
	if !session.RememberMe {
		sm.RenewSession(cookie.Value)
	} else if time.Until(session.ExpiresAt) < 24*time.Hour {
		sm.RenewSession(cookie.Value)
	}

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
