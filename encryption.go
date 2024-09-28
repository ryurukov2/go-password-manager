package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
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

func verifyMasterPassword() (bool, error) {
	saltHex, err := os.ReadFile("salt.txt")
	if err != nil {
		return false, fmt.Errorf("failed to read salt: %v", err)
	}

	salt, err := hex.DecodeString(string(saltHex))
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %v", err)
	}

	storedKeyHash, err := os.ReadFile("key_hash.txt")
	if err != nil {
		return false, fmt.Errorf("failed to read key hash: %v", err)
	}

	masterPassword, err := promptMasterPassword("Enter your master password: ")
	if err != nil {
		return false, err
	}

	derivedKey := deriveKeyFromPassword(masterPassword, salt)

	enteredKeyHash := hashKey(derivedKey)

	if subtle.ConstantTimeCompare([]byte(enteredKeyHash), storedKeyHash) == 1 {
		fmt.Println("Login successful.")
		return true, nil
	}

	fmt.Println("Incorrect master password.")
	return false, nil
}
