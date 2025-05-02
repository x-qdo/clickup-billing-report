package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// deriveKey derives a 32-byte key from the session secret using HKDF.
func deriveKey(secret string) ([]byte, error) {
	// Use HKDF to derive a stable encryption key from the potentially non-uniform secret
	hash := sha256.New
	// No salt needed here, the secret itself provides uniqueness
	// Info can be used to differentiate keys if needed (e.g., "session-token-encryption")
	hkdfReader := hkdf.New(hash, []byte(secret), nil, []byte("clickup-reporter-token-key"))
	key := make([]byte, 32) // AES-256 key size
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("failed to derive key using HKDF: %w", err)
	}
	return key, nil
}

// encrypt encrypts plaintext using AES-GCM with the derived key.
// Returns base64 encoded ciphertext (nonce + encrypted data).
func encrypt(plaintext string, secret string) (string, error) {
	key, err := deriveKey(secret)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts base64 encoded ciphertext using AES-GCM with the derived key.
func decrypt(ciphertextB64 string, secret string) (string, error) {
	key, err := deriveKey(secret)
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher block for decryption: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM cipher for decryption: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, encryptedMessage := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encryptedMessage, nil)
	if err != nil {
		// This often means the key is wrong or the ciphertext is corrupted
		return "", fmt.Errorf("failed to decrypt message (check secret key): %w", err)
	}

	return string(plaintext), nil
}
