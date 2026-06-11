package auth

import (
	"testing"
	"time"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("supersecret123")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "supersecret123" {
		t.Fatal("hash should not equal plaintext password")
	}
	if !CheckPassword(hash, "supersecret123") {
		t.Fatal("CheckPassword should return true for correct password")
	}
	if CheckPassword(hash, "wrongpassword") {
		t.Fatal("CheckPassword should return false for incorrect password")
	}
}

func TestGenerateAndParseToken(t *testing.T) {
	secret := "test-secret"
	token, err := GenerateToken(secret, "user-123", "user@example.com", "user", time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	claims, err := ParseToken(secret, token)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("expected UserID 'user-123', got %q", claims.UserID)
	}
	if claims.Email != "user@example.com" {
		t.Errorf("expected Email 'user@example.com', got %q", claims.Email)
	}
	if claims.Role != "user" {
		t.Errorf("expected Role 'user', got %q", claims.Role)
	}
}

func TestParseTokenInvalid(t *testing.T) {
	if _, err := ParseToken("test-secret", "not-a-valid-token"); err == nil {
		t.Fatal("expected error for invalid token, got nil")
	}

	token, err := GenerateToken("secret-a", "user-123", "user@example.com", "user", time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	if _, err := ParseToken("secret-b", token); err == nil {
		t.Fatal("expected error when parsing token with wrong secret, got nil")
	}
}

func TestParseExpiredToken(t *testing.T) {
	token, err := GenerateToken("test-secret", "user-123", "user@example.com", "user", -time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	if _, err := ParseToken("test-secret", token); err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}
