package main

import (
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	saltFile     = "salt.txt"
	verifierFile = "key_hash.txt"
	entriesFile  = "passwords.json"
)

var (
	ErrWrongPassword = errors.New("incorrect master password")
	ErrNotFound      = errors.New("entry not found")
)

// Entry is one stored credential. On disk, Username and Password are encrypted
// (hex-encoded AES-GCM); the service name is kept in plaintext so it can be looked up.
type Entry struct {
	Service  string `json:"service"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Vault is the encrypted store in a single directory.
type Vault struct {
	Dir string
	key []byte
}

func (v *Vault) path(name string) string { return filepath.Join(v.Dir, name) }

// Exists reports whether a master password has been set up in this directory.
func (v *Vault) Exists() bool {
	_, err := os.Stat(v.path(saltFile))
	return err == nil
}

// Create initialises a new vault with the given master password.
func (v *Vault) Create(masterPassword string) error {
	if v.Exists() {
		return errors.New("a vault already exists in " + v.Dir)
	}
	if err := os.MkdirAll(v.Dir, 0o700); err != nil {
		return fmt.Errorf("create vault directory: %w", err)
	}
	salt, err := newSalt()
	if err != nil {
		return err
	}
	key, err := deriveKey(masterPassword, salt)
	if err != nil {
		return err
	}
	if err := writeFileAtomic(v.path(saltFile), []byte(hex.EncodeToString(salt))); err != nil {
		return fmt.Errorf("save salt: %w", err)
	}
	if err := writeFileAtomic(v.path(verifierFile), []byte(keyVerifier(key))); err != nil {
		return fmt.Errorf("save key verifier: %w", err)
	}
	v.key = key
	return nil
}

// Unlock checks the master password and, if correct, keeps the derived key in memory.
func (v *Vault) Unlock(masterPassword string) error {
	saltHex, err := os.ReadFile(v.path(saltFile))
	if err != nil {
		return fmt.Errorf("read salt: %w", err)
	}
	salt, err := hex.DecodeString(strings.TrimSpace(string(saltHex)))
	if err != nil {
		return fmt.Errorf("decode salt: %w", err)
	}
	stored, err := os.ReadFile(v.path(verifierFile))
	if err != nil {
		return fmt.Errorf("read key verifier: %w", err)
	}
	key, err := deriveKey(masterPassword, salt)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(keyVerifier(key)), []byte(strings.TrimSpace(string(stored)))) != 1 {
		return ErrWrongPassword
	}
	v.key = key
	return nil
}

func (v *Vault) load() ([]Entry, error) {
	data, err := os.ReadFile(v.path(entriesFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read entries: %w", err)
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse entries: %w", err)
	}
	return entries, nil
}

func (v *Vault) save(entries []Entry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("encode entries: %w", err)
	}
	return writeFileAtomic(v.path(entriesFile), data)
}

// Put stores a credential. If the service already has an entry for that
// username, its password is replaced and updated is true.
func (v *Vault) Put(service, username, password string) (updated bool, err error) {
	entries, err := v.load()
	if err != nil {
		return false, err
	}
	encUser, err := encrypt(username, v.key)
	if err != nil {
		return false, err
	}
	encPass, err := encrypt(password, v.key)
	if err != nil {
		return false, err
	}
	newEntry := Entry{Service: service, Username: encUser, Password: encPass}

	for i, e := range entries {
		if strings.EqualFold(e.Service, service) {
			if u, err := decrypt(e.Username, v.key); err == nil && u == username {
				newEntry.Service = e.Service // keep the original spelling
				entries[i] = newEntry
				return true, v.save(entries)
			}
		}
	}
	return false, v.save(append(entries, newEntry))
}

// Get returns all decrypted credentials for a service (case-insensitive).
func (v *Vault) Get(service string) ([]Entry, error) {
	entries, err := v.load()
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, e := range entries {
		if !strings.EqualFold(e.Service, service) {
			continue
		}
		u, err := decrypt(e.Username, v.key)
		if err != nil {
			return nil, err
		}
		p, err := decrypt(e.Password, v.key)
		if err != nil {
			return nil, err
		}
		out = append(out, Entry{Service: e.Service, Username: u, Password: p})
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	return out, nil
}

// Delete removes the credential for the given service and username.
func (v *Vault) Delete(service, username string) error {
	entries, err := v.load()
	if err != nil {
		return err
	}
	for i, e := range entries {
		if !strings.EqualFold(e.Service, service) {
			continue
		}
		if u, err := decrypt(e.Username, v.key); err == nil && u == username {
			return v.save(append(entries[:i], entries[i+1:]...))
		}
	}
	return ErrNotFound
}

// Services lists stored service names with the number of accounts for each.
func (v *Vault) Services() ([]string, map[string]int, error) {
	entries, err := v.load()
	if err != nil {
		return nil, nil, err
	}
	counts := map[string]int{}
	for _, e := range entries {
		counts[e.Service]++
	}
	names := make([]string, 0, len(counts))
	for n := range counts {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool { return strings.ToLower(names[i]) < strings.ToLower(names[j]) })
	return names, counts, nil
}

// writeFileAtomic writes to a temp file and renames it into place so a crash
// mid-write can never leave a truncated vault. os.CreateTemp creates the file
// with mode 0600, so vault files are readable by the owner only.
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
