package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"

	"golang.org/x/term"

	"golang.org/x/crypto/pbkdf2"
)

const (
	saltSize   = 16
	keySize    = 32
	iterations = 100000
)

func promptMasterPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("failed to read master password: %v", err)
	}
	return string(bytePassword), nil
}

func deriveKeyFromPassword(masterPassword string, salt []byte) []byte {
	return pbkdf2.Key([]byte(masterPassword), salt, iterations, keySize, sha256.New)
}

func generateRandomSalt() ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %v", err)
	}
	return salt, nil
}

func hashKey(key []byte) string {
	hash := sha256.Sum256(key)
	return hex.EncodeToString(hash[:])
}

func setupMasterPassword() (string, error) {
	salt, err := generateRandomSalt()
	if err != nil {
		return "", err
	}

	masterPassword, err := promptMasterPassword("Set up master password: ")
	if err != nil {
		return "", err
	}

	derivedKey := deriveKeyFromPassword(masterPassword, salt)

	saltHex := hex.EncodeToString(salt)
	err = os.WriteFile("salt.txt", []byte(saltHex), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to save salt: %v", err)
	}
	hashedKey := hashKey(derivedKey)

	if err := os.WriteFile("key_hash.txt", []byte(hashedKey), 0644); err != nil {
		return "", fmt.Errorf("failed to save hashed key: %v", err)
	}

	fmt.Println("Salt saved successfully. Master password not stored.")
	return string(derivedKey), nil
}

func verifyMasterPasswordAndGetKey() ([]byte, error) {
	saltHex, err := os.ReadFile("salt.txt")
	var ret []byte
	if err != nil {
		return ret, fmt.Errorf("failed to read salt: %v", err)
	}

	salt, err := hex.DecodeString(string(saltHex))
	if err != nil {
		return ret, fmt.Errorf("failed to decode salt: %v", err)
	}

	storedKeyHash, err := os.ReadFile("key_hash.txt")
	if err != nil {
		return ret, fmt.Errorf("failed to read key hash: %v", err)
	}

	masterPassword, err := promptMasterPassword("Enter your master password: ")
	if err != nil {
		return ret, err
	}

	derivedKey := deriveKeyFromPassword(masterPassword, salt)

	enteredKeyHash := hashKey(derivedKey)

	if subtle.ConstantTimeCompare([]byte(enteredKeyHash), storedKeyHash) == 1 {
		fmt.Println("Login successful.")
		return derivedKey, nil
	}

	fmt.Println("Incorrect master password.")
	return ret, nil
}

func encryptData(plaintext, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher block: %v", err)
	}

	// GCM mode requires a nonce (IV)
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %v", err)
	}

	// Generate a random nonce of appropriate length
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %v", err)
	}

	// Encrypt and authenticate the plaintext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return hex.EncodeToString(ciphertext), nil
}

func decryptData(ciphertextHex string, key []byte) (string, error) {
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %v", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher block: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %v", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// Extract the nonce from the beginning of the ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt and verify the ciphertext
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt data: %v", err)
	}

	return string(plaintext), nil
}
