package utils_test

import (
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/utils"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	aesKey := make([]byte, 32)
	for index := range aesKey {
		aesKey[index] = byte(index + 1)
	}
	originalText := "ghp_supersecretgithubtoken1234567890"
	encrypted, err := utils.EncryptToken(originalText, aesKey)
	if err != nil {
		t.Fatalf("expected no error encrypting, got: %v", err)
	}
	decrypted, err := utils.DecryptToken(encrypted, aesKey)
	if err != nil {
		t.Fatalf("expected no error decrypting, got: %v", err)
	}
	if decrypted != originalText {
		t.Fatalf("round-trip failed: got %q, want %q", decrypted, originalText)
	}
}

func TestEncryptProducesDistinctCiphertexts(t *testing.T) {
	aesKey := make([]byte, 32)
	for index := range aesKey {
		aesKey[index] = byte(index + 7)
	}
	originalText := "same_plaintext_token"
	firstEncrypted, err := utils.EncryptToken(originalText, aesKey)
	if err != nil {
		t.Fatalf("expected no error on first encrypt: %v", err)
	}
	secondEncrypted, err := utils.EncryptToken(originalText, aesKey)
	if err != nil {
		t.Fatalf("expected no error on second encrypt: %v", err)
	}
	if firstEncrypted == secondEncrypted {
		t.Fatal("two encryptions of same plaintext must produce distinct ciphertexts (random nonce)")
	}
}

func TestDecryptWithWrongKeyReturnsError(t *testing.T) {
	correctKey := make([]byte, 32)
	for index := range correctKey {
		correctKey[index] = byte(index + 1)
	}
	wrongKey := make([]byte, 32)
	for index := range wrongKey {
		wrongKey[index] = byte(index + 200)
	}
	encrypted, err := utils.EncryptToken("secret_token", correctKey)
	if err != nil {
		t.Fatalf("expected no error encrypting: %v", err)
	}
	_, decryptErr := utils.DecryptToken(encrypted, wrongKey)
	if decryptErr == nil {
		t.Fatal("expected error decrypting with wrong key, got nil")
	}
}

func TestEncryptWithShortKeyReturnsError(t *testing.T) {
	shortKey := make([]byte, 16)
	_, err := utils.EncryptToken("any text", shortKey)
	if err == nil {
		t.Fatal("expected error for 16-byte key, got nil")
	}
}

func TestDecryptInvalidBase64ReturnsError(t *testing.T) {
	aesKey := make([]byte, 32)
	_, err := utils.DecryptToken("not-valid-base64!!!", aesKey)
	if err == nil {
		t.Fatal("expected error for invalid base64 input, got nil")
	}
}
