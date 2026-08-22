package secret

import "testing"

func TestCryptoEncryptDecryptRoundTrip(t *testing.T) {
	crypto := NewCrypto("top-secret")

	encrypted, err := crypto.Encrypt("ak_test_secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if encrypted == "" || encrypted == "ak_test_secret" {
		t.Fatalf("encrypted payload should be non-empty ciphertext, got %q", encrypted)
	}

	decrypted, err := crypto.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if decrypted != "ak_test_secret" {
		t.Fatalf("decrypted = %q", decrypted)
	}
}

func TestCryptoRejectsWrongKeyAndInvalidPayload(t *testing.T) {
	encrypted, err := NewCrypto("right-key").Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if _, err := NewCrypto("wrong-key").Decrypt(encrypted); err == nil {
		t.Fatal("Decrypt should reject ciphertext encrypted with another key")
	}
	if _, err := NewCrypto("right-key").Decrypt("not-base64"); err == nil {
		t.Fatal("Decrypt should reject invalid base64 payload")
	}
}
