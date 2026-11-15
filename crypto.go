package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

const (
	saltSize   = 16
	keySize    = 32 // AES-256
	iterations = 100_000
)

// deriveKey turns the master password into a 256-bit AES key using PBKDF2-HMAC-SHA256.
func deriveKey(masterPassword string, salt []byte) ([]byte, error) {
	return pbkdf2.Key(sha256.New, masterPassword, salt, iterations, keySize)
}

// keyVerifier is what gets stored on disk to check the master password.
// It is a hash of the derived key, so neither the password nor the key is ever stored.
func keyVerifier(key []byte) string {
	sum := sha256.Sum256(key)
	return hex.EncodeToString(sum[:])
}

func newSalt() ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	return salt, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	return cipher.NewGCM(block)
}

// encrypt seals plaintext with AES-GCM and returns hex(nonce || ciphertext).
func encrypt(plaintext string, key []byte) (string, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	return hex.EncodeToString(gcm.Seal(nonce, nonce, []byte(plaintext), nil)), nil
}

// decrypt reverses encrypt.
func decrypt(ciphertextHex string, key []byte) (string, error) {
	data, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}

// generatePassword returns a random password of the given length drawn from a
// mixed alphabet, using crypto/rand with rejection sampling to avoid bias.
func generatePassword(length int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*-_=+?"
	if length < 4 || length > 256 {
		return "", errors.New("length must be between 4 and 256")
	}
	out := make([]byte, 0, length)
	buf := make([]byte, 1)
	limit := byte(256 - 256%len(alphabet))
	for len(out) < length {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		if buf[0] < limit {
			out = append(out, alphabet[int(buf[0])%len(alphabet)])
		}
	}
	return string(out), nil
}
