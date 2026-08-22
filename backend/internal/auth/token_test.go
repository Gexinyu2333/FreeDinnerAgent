package auth

import (
	"testing"
	"time"
)

func TestGenerateAndParseAccessToken(t *testing.T) {
	token, err := GenerateAccessToken("secret", "user-1", "alice", time.Hour)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	claims, err := ParseAccessToken("secret", token)
	if err != nil {
		t.Fatalf("ParseAccessToken() error = %v", err)
	}
	if claims.UserID != "user-1" || claims.Username != "alice" || claims.Subject != "user-1" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestParseAccessTokenRejectsWrongSecretAndExpiredToken(t *testing.T) {
	token, err := GenerateAccessToken("secret", "user-1", "alice", time.Hour)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}
	if _, err := ParseAccessToken("other-secret", token); err == nil {
		t.Fatal("ParseAccessToken should reject tokens signed with another secret")
	}

	expired, err := GenerateAccessToken("secret", "user-1", "alice", -time.Hour)
	if err != nil {
		t.Fatalf("GenerateAccessToken(expired) error = %v", err)
	}
	if _, err := ParseAccessToken("secret", expired); err == nil {
		t.Fatal("ParseAccessToken should reject expired tokens")
	}
}

func TestGenerateRefreshTokenReturnsHashOnly(t *testing.T) {
	token, hash, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}
	if token == "" || hash == "" {
		t.Fatal("token and hash should be non-empty")
	}
	if token == hash {
		t.Fatal("refresh token hash should not equal raw token")
	}
}
