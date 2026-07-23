// Package crypto provides AES-256-GCM encryption for sensitive data such as API keys.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	// keyHexLen is the expected length of the hex-encoded ENCRYPTION_KEY (32 bytes = 64 hex chars).
	keyHexLen = 64
	// nonceSize is the standard GCM nonce size (12 bytes).
	nonceSize = 12
)

// EncryptionKey holds the 32-byte AES-256 key.
type EncryptionKey [32]byte

// LoadEncryptionKey reads the AES-256 key from the ENCRYPTION_KEY environment variable.
// The key must be a 64-character hex string (32 bytes).
func LoadEncryptionKey() (EncryptionKey, error) {
	hexKey := os.Getenv("ENCRYPTION_KEY")
	if hexKey == "" {
		return EncryptionKey{}, errors.New("ENCRYPTION_KEY environment variable is not set")
	}
	if len(hexKey) != keyHexLen {
		return EncryptionKey{}, fmt.Errorf("ENCRYPTION_KEY must be %d hex characters (32 bytes), got %d chars", keyHexLen, len(hexKey))
	}
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return EncryptionKey{}, fmt.Errorf("ENCRYPTION_KEY is not valid hex: %w", err)
	}
	var key EncryptionKey
	copy(key[:], keyBytes)
	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM and returns a hex-encoded string.
// Format: nonce_hex + ciphertext_hex + tag_hex
func Encrypt(key EncryptionKey, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", fmt.Errorf("aes new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("cipher new gcm: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	// Encrypt the plaintext. Seal appends ciphertext+tag to a fresh buffer.
	// We use nil as the destination and manually prepend the nonce afterwards
	// to avoid overwriting the nonce buffer.
	ciphertext := aesGCM.Seal(nil, nonce, plaintext, nil)
	result := make([]byte, 0, nonceSize+len(ciphertext))
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	return hex.EncodeToString(result), nil
}

// Decrypt decrypts a hex-encoded AES-256-GCM ciphertext and returns the plaintext.
// Expects format: nonce_hex + ciphertext_hex + tag_hex
func Decrypt(key EncryptionKey, encryptedHex string) ([]byte, error) {
	ciphertext, err := hex.DecodeString(encryptedHex)
	if err != nil {
		return nil, fmt.Errorf("decode hex: %w", err)
	}

	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("aes new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cipher new gcm: %w", err)
	}

	nonce := ciphertext[:nonceSize]
	encryptedData := ciphertext[nonceSize:]

	plaintext, err := aesGCM.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

// MaskAPIKey returns a masked version of the API key showing only the first 4
// and last 4 characters (e.g., "sk-34ab****cdef").
func MaskAPIKey(apiKey string) string {
	runes := []rune(apiKey)
	l := len(runes)
	if l == 0 {
		return "****"
	}
	if l <= 4 {
		return string(runes[:1]) + "****" + string(runes[l-1:])
	}
	if l <= 8 {
		return string(runes[:2]) + "****" + string(runes[l-2:])
	}
	return string(runes[:4]) + "****" + string(runes[l-4:])
}

// MustLoadEncryptionKey loads the encryption key from ENCRYPTION_KEY env var and panics if not set.
func MustLoadEncryptionKey() EncryptionKey {
	key, err := LoadEncryptionKey()
	if err != nil {
		panic("crypto: " + err.Error())
	}
	return key
}
