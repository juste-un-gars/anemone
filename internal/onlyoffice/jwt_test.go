package onlyoffice

import (
	"testing"
)

func TestSignAndVerifyFileToken(t *testing.T) {
	secret := "test-secret-key-12345"

	token, err := SignFileToken(secret, 42, "documents", "report.docx")
	if err != nil {
		t.Fatalf("SignFileToken failed: %v", err)
	}

	claims, err := VerifyFileToken(secret, token)
	if err != nil {
		t.Fatalf("VerifyFileToken failed: %v", err)
	}

	if claims.UserID != 42 {
		t.Errorf("expected UserID 42, got %d", claims.UserID)
	}
	if claims.ShareName != "documents" {
		t.Errorf("expected ShareName 'documents', got %s", claims.ShareName)
	}
	if claims.FilePath != "report.docx" {
		t.Errorf("expected FilePath 'report.docx', got %s", claims.FilePath)
	}
}

func TestVerifyFileTokenWrongSecret(t *testing.T) {
	token, _ := SignFileToken("secret-a", 1, "share", "file.txt")

	_, err := VerifyFileToken("secret-b", token)
	if err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
}

func TestSignEditorConfig(t *testing.T) {
	secret := "test-secret"
	config := map[string]interface{}{
		"document": map[string]interface{}{
			"title": "test.docx",
		},
	}

	token, err := SignEditorConfig(secret, config)
	if err != nil {
		t.Fatalf("SignEditorConfig failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}
