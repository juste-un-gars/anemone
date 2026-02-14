// This file handles JWT token creation and verification for OnlyOffice integration.
// Tokens secure the communication between Anemone and the OnlyOffice container.
package onlyoffice

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// FileClaims holds the claims for a file download token.
// Used by the /api/oo/download endpoint to authorize the OnlyOffice container.
type FileClaims struct {
	UserID    int    `json:"uid"`
	ShareName string `json:"share"`
	FilePath  string `json:"path"`
	jwt.RegisteredClaims
}

// SignFileToken creates a JWT token authorizing access to a specific file.
// The token is short-lived (1 hour) and includes user, share, and path info.
func SignFileToken(secret string, userID int, shareName, filePath string) (string, error) {
	claims := FileClaims{
		UserID:    userID,
		ShareName: shareName,
		FilePath:  filePath,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "anemone",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// VerifyFileToken validates a file download JWT and returns the claims.
func VerifyFileToken(secret, tokenStr string) (*FileClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &FileClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*FileClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// SignEditorConfig signs the OnlyOffice editor configuration as a JWT payload.
// OnlyOffice expects the entire config to be signed when JWT is enabled.
func SignEditorConfig(secret string, payload map[string]interface{}) (string, error) {
	claims := jwt.MapClaims(payload)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// VerifyCallbackToken validates a JWT from OnlyOffice callback requests.
// Returns the claims map from the token payload.
func VerifyCallbackToken(secret, tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid callback token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid callback claims")
	}

	return claims, nil
}
