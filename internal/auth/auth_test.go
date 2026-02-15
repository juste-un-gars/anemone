// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package auth

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create sessions table
	_, err = db.Exec(`
		CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			is_admin BOOLEAN DEFAULT 0,
			remember_me BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			last_activity DATETIME DEFAULT CURRENT_TIMESTAMP,
			user_agent TEXT,
			ip_address TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sessions table: %v", err)
	}

	return db
}

func TestSessionManagerCreateSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sm := &SessionManager{db: db}

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

func TestSessionManagerCreateSessionWithOptions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sm := &SessionManager{db: db}

	// Test with remember me
	session, err := sm.CreateSessionWithOptions(1, "testuser", true, true, "Mozilla/5.0", "192.168.1.1")
	if err != nil {
		t.Fatalf("CreateSessionWithOptions failed: %v", err)
	}

	if !session.RememberMe {
		t.Error("Expected RememberMe true")
	}
	if session.UserAgent != "Mozilla/5.0" {
		t.Errorf("Expected UserAgent 'Mozilla/5.0', got %q", session.UserAgent)
	}
	if session.IPAddress != "192.168.1.1" {
		t.Errorf("Expected IPAddress '192.168.1.1', got %q", session.IPAddress)
	}

	// Check that expiry is ~30 days for remember me
	expectedExpiry := time.Now().Add(RememberMeDuration)
	if session.ExpiresAt.Before(expectedExpiry.Add(-time.Minute)) {
		t.Error("Remember me session should expire in ~30 days")
	}
}

func TestSessionManagerGetSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sm := &SessionManager{db: db}

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
	db := setupTestDB(t)
	defer db.Close()

	sm := &SessionManager{db: db}

	// Insert an already expired session directly
	_, err := db.Exec(`
		INSERT INTO sessions (id, user_id, username, is_admin, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "test-expired", 1, "testuser", false,
		time.Now().Add(-3*time.Hour), time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("Failed to insert expired session: %v", err)
	}

	// Try to get expired session
	_, err = sm.GetSession("test-expired")
	if err == nil {
		t.Error("GetSession should fail for expired session")
	}
}

func TestSessionManagerDeleteSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sm := &SessionManager{db: db}

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

func TestSessionManagerDeleteUserSessions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sm := &SessionManager{db: db}

	// Create multiple sessions for the same user
	session1, _ := sm.CreateSession(1, "testuser", false)
	session2, _ := sm.CreateSession(1, "testuser", false)

	// Delete all sessions for user
	err := sm.DeleteUserSessions(1)
	if err != nil {
		t.Fatalf("DeleteUserSessions failed: %v", err)
	}

	// Both sessions should be gone
	_, err1 := sm.GetSession(session1.ID)
	_, err2 := sm.GetSession(session2.ID)
	if err1 == nil || err2 == nil {
		t.Error("All user sessions should be deleted")
	}
}

func TestSessionManagerRenewSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sm := &SessionManager{db: db}

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
	db := setupTestDB(t)
	defer db.Close()

	sm := &SessionManager{db: db}

	err := sm.RenewSession("nonexistent")
	if err == nil {
		t.Error("RenewSession should fail for non-existent session")
	}
}

func TestSessionManagerGetUserSessions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sm := &SessionManager{db: db}

	// Create multiple sessions for the same user
	sm.CreateSession(1, "testuser", false)
	sm.CreateSession(1, "testuser", false)
	sm.CreateSession(2, "otheruser", false)

	// Get sessions for user 1
	sessions, err := sm.GetUserSessions(1)
	if err != nil {
		t.Fatalf("GetUserSessions failed: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
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
	SetSessionCookie(rr, "test-session-id", false)

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

func TestSessionCookieRememberMe(t *testing.T) {
	// Test without remember me
	rr1 := httptest.NewRecorder()
	SetSessionCookie(rr1, "test-session-id", false)

	cookie1 := rr1.Result().Cookies()[0]
	expectedMaxAge := int(SessionDuration.Seconds())
	if cookie1.MaxAge != expectedMaxAge {
		t.Errorf("Expected MaxAge %d, got %d", expectedMaxAge, cookie1.MaxAge)
	}

	// Test with remember me
	rr2 := httptest.NewRecorder()
	SetSessionCookie(rr2, "test-session-id", true)

	cookie2 := rr2.Result().Cookies()[0]
	expectedMaxAgeRemember := int(RememberMeDuration.Seconds())
	if cookie2.MaxAge != expectedMaxAgeRemember {
		t.Errorf("Expected RememberMe MaxAge %d, got %d", expectedMaxAgeRemember, cookie2.MaxAge)
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

func TestSessionDurations(t *testing.T) {
	// Session duration should be 2 hours
	expected := 2 * time.Hour
	if SessionDuration != expected {
		t.Errorf("SessionDuration should be %v, got %v", expected, SessionDuration)
	}

	// Remember me duration should be 14 days
	expectedRemember := 14 * 24 * time.Hour
	if RememberMeDuration != expectedRemember {
		t.Errorf("RememberMeDuration should be %v, got %v", expectedRemember, RememberMeDuration)
	}
}
