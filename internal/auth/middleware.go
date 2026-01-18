// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package auth provides authentication middleware and session management.
package auth

import (
	"context"
	"database/sql"
	"net/http"
)

// contextKey is a custom type for context keys
type contextKey string

const (
	// SessionContextKey is the key for storing session in context
	SessionContextKey contextKey = "session"
)

// RequireAuth is a middleware that requires authentication
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := GetSessionFromRequest(r)
		if err != nil {
			// Not authenticated, redirect to login
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Add session to context
		ctx := context.WithValue(r.Context(), SessionContextKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RequireAdmin is a middleware that requires admin authentication
func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := GetSessionFromRequest(r)
		if err != nil {
			// Not authenticated, redirect to login
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if !session.IsAdmin {
			// Not admin, return forbidden
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}

		// Add session to context
		ctx := context.WithValue(r.Context(), SessionContextKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GetSessionFromContext retrieves the session from the request context
func GetSessionFromContext(r *http.Request) (*Session, bool) {
	session, ok := r.Context().Value(SessionContextKey).(*Session)
	return session, ok
}

// RedirectIfAuthenticated redirects authenticated users to dashboard
func RedirectIfAuthenticated(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if IsAuthenticated(r) {
			// Already authenticated, redirect to dashboard
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// RequireRestoreCheck checks if user needs to acknowledge server restore
func RequireRestoreCheck(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip check for restore-warning page itself and logout
		if r.URL.Path == "/restore-warning" || r.URL.Path == "/restore-warning/acknowledge" ||
		   r.URL.Path == "/restore-warning/bulk" || r.URL.Path == "/logout" {
			next.ServeHTTP(w, r)
			return
		}

		session, ok := GetSessionFromContext(r)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		// Check if server has been restored
		var serverRestored string
		err := db.QueryRow("SELECT value FROM system_config WHERE key = 'server_restored'").Scan(&serverRestored)
		if err != nil || serverRestored != "1" {
			// No restoration or error, continue normally
			next.ServeHTTP(w, r)
			return
		}

		// Check if user has acknowledged the restore
		var restoreAcknowledged bool
		err = db.QueryRow("SELECT restore_acknowledged FROM users WHERE id = ?", session.UserID).Scan(&restoreAcknowledged)
		if err != nil {
			// Error reading, continue normally
			next.ServeHTTP(w, r)
			return
		}

		if !restoreAcknowledged {
			// User needs to acknowledge restore, redirect to warning page
			http.Redirect(w, r, "/restore-warning", http.StatusSeeOther)
			return
		}

		// User has acknowledged, continue normally
		next.ServeHTTP(w, r)
	}
}
