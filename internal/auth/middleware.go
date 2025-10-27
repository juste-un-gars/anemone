// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package auth

import (
	"context"
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
