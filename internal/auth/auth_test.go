// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionManagerCreateSession(t *testing.T) {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	session, err := sm.CreateSession(1, "testuser", false)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Verify session fields
	if session.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", session.UserID)
	}
	if session.Username != "testuser" {
		t.Errorf("Expected Username 'testuser', got %q", session.Username)
	}
	if session.IsAdmin != false {
		t.Error("Expected IsAdmin false")
	}
	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}
	if session.ExpiresAt.Before(time.Now()) {
		t.Error("Session should not be expired immediately")
	}
}

func TestSessionManagerGetSession(t *testing.T) {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	// Create a session
	session, _ := sm.CreateSession(1, "testuser", false)

	// Get the session
	retrieved, err := sm.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Error("Retrieved session should match original")
	}

	// Try to get non-existent session
	_, err = sm.GetSession("nonexistent")
	if err == nil {
		t.Error("GetSession should fail for non-existent session")
	}
}

func TestSessionManagerGetExpiredSession(t *testing.T) {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	// Create an already expired session
	session := &Session{
		ID:        "test-expired",
		UserID:    1,
		Username:  "testuser",
		CreatedAt: time.Now().Add(-3 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}

	sm.mu.Lock()
	sm.sessions[session.ID] = session
	sm.mu.Unlock()

	// Try to get expired session
	_, err := sm.GetSession(session.ID)
	if err == nil {
		t.Error("GetSession should fail for expired session")
	}
}

func TestSessionManagerDeleteSession(t *testing.T) {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	// Create a session
	session, _ := sm.CreateSession(1, "testuser", false)

	// Delete it
	sm.DeleteSession(session.ID)

	// Try to get it
	_, err := sm.GetSession(session.ID)
	if err == nil {
		t.Error("GetSession should fail after deletion")
	}
}

func TestSessionManagerRenewSession(t *testing.T) {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	// Create a session
	session, _ := sm.CreateSession(1, "testuser", false)
	originalExpiry := session.ExpiresAt

	// Wait a tiny bit
	time.Sleep(10 * time.Millisecond)

	// Renew it
	err := sm.RenewSession(session.ID)
	if err != nil {
		t.Fatalf("RenewSession failed: %v", err)
	}

	// Get the session and check new expiry
	retrieved, _ := sm.GetSession(session.ID)
	if !retrieved.ExpiresAt.After(originalExpiry) {
		t.Error("Session expiry should be extended after renewal")
	}
}

func TestSessionManagerRenewNonExistentSession(t *testing.T) {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	err := sm.RenewSession("nonexistent")
	if err == nil {
		t.Error("RenewSession should fail for non-existent session")
	}
}

func TestGetSessionFromContext(t *testing.T) {
	// Create a request with session in context
	session := &Session{
		ID:       "test",
		UserID:   1,
		Username: "testuser",
		IsAdmin:  true,
	}

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), SessionContextKey, session)
	req = req.WithContext(ctx)

	// Get session from context
	retrieved, ok := GetSessionFromContext(req)
	if !ok {
		t.Fatal("GetSessionFromContext should return ok=true")
	}
	if retrieved.UserID != session.UserID {
		t.Error("Retrieved session should match")
	}

	// Test with no session in context
	req2 := httptest.NewRequest("GET", "/", nil)
	_, ok = GetSessionFromContext(req2)
	if ok {
		t.Error("GetSessionFromContext should return ok=false when no session")
	}
}

func TestRequireAuthMiddleware(t *testing.T) {
	// Test handler that records if it was called
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RequireAuth(handler)

	// Test without session - should redirect
	req := httptest.NewRequest("GET", "/protected", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Expected redirect status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if called {
		t.Error("Handler should not be called without valid session")
	}
}

func TestRequireAdminMiddleware(t *testing.T) {
	// Test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RequireAdmin(handler)

	// Test without session - should redirect
	req := httptest.NewRequest("GET", "/admin", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Expected redirect status %d, got %d", http.StatusSeeOther, rr.Code)
	}
}

func TestRedirectIfAuthenticatedMiddleware(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RedirectIfAuthenticated(handler)

	// Test without authentication - should call handler
	req := httptest.NewRequest("GET", "/login", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if !called {
		t.Error("Handler should be called when not authenticated")
	}
}

func TestSessionCookieSecurityAttributes(t *testing.T) {
	rr := httptest.NewRecorder()
	SetSessionCookie(rr, "test-session-id")

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]

	// Check security attributes
	if cookie.Name != SessionCookieName {
		t.Errorf("Expected cookie name %q, got %q", SessionCookieName, cookie.Name)
	}
	if !cookie.HttpOnly {
		t.Error("Cookie should be HttpOnly")
	}
	if !cookie.Secure {
		t.Error("Cookie should be Secure")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Error("Cookie should have SameSite=Strict")
	}
}

func TestClearSessionCookie(t *testing.T) {
	rr := httptest.NewRecorder()
	ClearSessionCookie(rr)

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.MaxAge != -1 {
		t.Errorf("Cookie MaxAge should be -1, got %d", cookie.MaxAge)
	}
}

func TestGenerateSessionIDUniqueness(t *testing.T) {
	ids := make(map[string]bool)

	// Generate 100 session IDs and verify uniqueness
	for i := 0; i < 100; i++ {
		id, err := generateSessionID()
		if err != nil {
			t.Fatalf("generateSessionID failed: %v", err)
		}
		if ids[id] {
			t.Errorf("Duplicate session ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestSessionDuration(t *testing.T) {
	// Session duration should be 2 hours
	expected := 2 * time.Hour
	if SessionDuration != expected {
		t.Errorf("SessionDuration should be %v, got %v", expected, SessionDuration)
	}
}
