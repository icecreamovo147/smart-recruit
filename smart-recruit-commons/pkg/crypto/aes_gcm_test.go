package crypto

import (
	"encoding/hex"
	"os"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := EncryptionKey{}
	for i := 0; i < 32; i++ {
		key[i] = byte(i)
	}

	plaintext := []byte("sk-test-api-key-1234567890abcdef")

	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if encrypted == "" {
		t.Fatal("Encrypt returned empty string")
	}

	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("Decrypt returned wrong plaintext: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	key := EncryptionKey{}
	for i := 0; i < 32; i++ {
		key[i] = byte(i)
	}

	plaintext := []byte("same-text")

	c1, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("First Encrypt failed: %v", err)
	}

	c2, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Second Encrypt failed: %v", err)
	}

	if c1 == c2 {
		t.Fatal("Two encryptions of the same plaintext produced the same ciphertext (nonce reuse?)")
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := EncryptionKey{}
	for i := 0; i < 32; i++ {
		key1[i] = byte(i)
	}
	key2 := EncryptionKey{}
	for i := 0; i < 32; i++ {
		key2[i] = byte(32 - i)
	}

	plaintext := []byte("test-api-key")
	encrypted, err := Encrypt(key1, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(key2, encrypted)
	if err == nil {
		t.Fatal("Decrypt with wrong key should have failed")
	}
}

func TestDecryptWithInvalidHex(t *testing.T) {
	key := EncryptionKey{}
	_, err := Decrypt(key, "not-hex-string")
	if err == nil {
		t.Fatal("Decrypt with invalid hex should have failed")
	}
}

func TestDecryptWithShortCiphertext(t *testing.T) {
	key := EncryptionKey{}
	_, err := Decrypt(key, "abcd")
	if err == nil {
		t.Fatal("Decrypt with short ciphertext should have failed")
	}
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"sk-abc123456789def", "sk-a****9def"},
		{"abcdefghijklmnop", "abcd****mnop"},
		{"shortone", "sh****ne"},
		{"short", "sh****rt"},
		{"abcd", "a****d"},
		{"ab", "a****b"},
		{"a", "a****a"},
		{"", "****"},
	}

	for _, tt := range tests {
		result := MaskAPIKey(tt.input)
		if result != tt.expected {
			t.Errorf("MaskAPIKey(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestLoadEncryptionKey(t *testing.T) {
	keyBytes := make([]byte, 32)
	for i := range keyBytes {
		keyBytes[i] = byte(i)
	}
	t.Setenv("ENCRYPTION_KEY", hex.EncodeToString(keyBytes))
	key, err := LoadEncryptionKey()
	if err != nil {
		t.Fatalf("LoadEncryptionKey failed with valid key: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("Expected 32-byte key, got %d", len(key))
	}
}

func TestLoadEncryptionKeyInvalidHex(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz")
	_, err := LoadEncryptionKey()
	if err == nil {
		t.Fatal("LoadEncryptionKey should have failed with invalid hex")
	}
}

func TestLoadEncryptionKeyWrongLength(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "abc123")
	_, err := LoadEncryptionKey()
	if err == nil {
		t.Fatal("LoadEncryptionKey should have failed with wrong length")
	}
}

func TestLoadEncryptionKeyEmpty(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "")
	_, err := LoadEncryptionKey()
	if err == nil {
		t.Fatal("LoadEncryptionKey should have failed with empty key")
	}
}

func TestMustLoadEncryptionKeyPanic(t *testing.T) {
	// Only test panic when ENCRYPTION_KEY is not set
	if os.Getenv("ENCRYPTION_KEY") != "" {
		t.Skip("ENCRYPTION_KEY is set, skipping panic test")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustLoadEncryptionKey should have panicked")
		}
	}()

	MustLoadEncryptionKey()
}
