// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// This file provides CSRF protection using the double-submit cookie pattern.

package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

const (
	// CSRFCookieName is the name of the CSRF cookie
	CSRFCookieName = "anemone_csrf"
	// CSRFFieldName is the name of the hidden form field
	CSRFFieldName = "csrf_token"
	// CSRFHeaderName is the header name for AJAX requests
	CSRFHeaderName = "X-CSRF-Token"
	// csrfTokenLength is the byte length of the token (32 bytes = 256 bits)
	csrfTokenLength = 32
)

// GenerateCSRFToken generates a random CSRF token
func GenerateCSRFToken() (string, error) {
	b := make([]byte, csrfTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// SetCSRFCookie sets the CSRF token cookie on the response
func SetCSRFCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // JS must read it for AJAX requests
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

// GetCSRFFromRequest returns the CSRF token from the cookie
func GetCSRFFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(CSRFCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// ValidateCSRF checks that the form/header token matches the cookie token
func ValidateCSRF(r *http.Request) bool {
	cookieToken := GetCSRFFromRequest(r)
	if cookieToken == "" {
		return false
	}

	// Check form field first, then header (for AJAX)
	formToken := r.FormValue(CSRFFieldName)
	if formToken == "" {
		formToken = r.Header.Get(CSRFHeaderName)
	}

	return formToken != "" && formToken == cookieToken
}
