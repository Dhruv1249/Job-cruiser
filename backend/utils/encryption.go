package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var ErrInvalidAESKeyLength = errors.New("AES key must be exactly 32 bytes for AES-256")
var ErrInvalidCiphertextFormat = errors.New("ciphertext too short to contain nonce")

// EncryptToken encrypts plaintext using AES-256-GCM with the provided 32-byte key.
// Returns a base64-encoded string containing the random nonce prepended to the ciphertext.
func EncryptToken(plaintext string, aesKey []byte) (string, error) {
	if len(aesKey) != 32 {
		return "", ErrInvalidAESKeyLength
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("failed creating AES cipher block: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed creating GCM cipher: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed generating random nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptToken decrypts a base64-encoded AES-256-GCM ciphertext back to its original plaintext.
// The nonce is expected to be prepended to the ciphertext as produced by EncryptToken.
func DecryptToken(encryptedBase64 string, aesKey []byte) (string, error) {
	if len(aesKey) != 32 {
		return "", ErrInvalidAESKeyLength
	}
	combined, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return "", fmt.Errorf("failed base64-decoding ciphertext: %w", err)
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("failed creating AES cipher block: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed creating GCM cipher: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(combined) < nonceSize {
		return "", ErrInvalidCiphertextFormat
	}
	nonce, ciphertext := combined[:nonceSize], combined[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed decrypting AES-GCM ciphertext: %w", err)
	}
	return string(plaintext), nil
}
