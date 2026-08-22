package auth

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("password hash should not equal plaintext")
	}
	if !CheckPassword("correct horse battery staple", hash) {
		t.Fatal("CheckPassword should accept the original password")
	}
	if CheckPassword("wrong password", hash) {
		t.Fatal("CheckPassword should reject a different password")
	}
}

func TestNormalizeDisplayName(t *testing.T) {
	blank := "   "
	if got := normalizeDisplayName(&blank); got != nil {
		t.Fatalf("blank display name should normalize to nil, got %q", *got)
	}

	name := "  小鱼  "
	got := normalizeDisplayName(&name)
	if got == nil || *got != "小鱼" {
		t.Fatalf("display name should be trimmed, got %#v", got)
	}
}
