package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func newTestVault(t *testing.T) *Vault {
	t.Helper()
	v := &Vault{Dir: t.TempDir()}
	if err := v.Create("correct horse"); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, keySize)
	ct, err := encrypt("s3cret pässword", key)
	if err != nil {
		t.Fatal(err)
	}
	ct2, _ := encrypt("s3cret pässword", key)
	if ct == ct2 {
		t.Error("same plaintext encrypted twice gave identical output; nonce not random")
	}
	pt, err := decrypt(ct, key)
	if err != nil || pt != "s3cret pässword" {
		t.Fatalf("got %q, %v", pt, err)
	}
	wrong := make([]byte, keySize)
	wrong[0] = 1
	if _, err := decrypt(ct, wrong); err == nil {
		t.Error("decrypt with the wrong key should fail")
	}
}

func TestUnlock(t *testing.T) {
	v := newTestVault(t)
	if err := (&Vault{Dir: v.Dir}).Unlock("wrong"); !errors.Is(err, ErrWrongPassword) {
		t.Errorf("wrong password: got %v", err)
	}
	if err := (&Vault{Dir: v.Dir}).Unlock("correct horse"); err != nil {
		t.Errorf("right password: got %v", err)
	}
	if err := v.Create("again"); err == nil {
		t.Error("Create over an existing vault should fail")
	}
}

func TestPutGetDelete(t *testing.T) {
	v := newTestVault(t)
	if _, err := v.Put("GitHub", "alice", "pw1"); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Put("github", "bob", "pw2"); err != nil {
		t.Fatal(err)
	}
	updated, err := v.Put("github", "alice", "pw3")
	if err != nil || !updated {
		t.Fatalf("expected update, got %v %v", updated, err)
	}

	got, err := v.Get("GITHUB")
	if err != nil {
		t.Fatal(err)
	}
	want := []Entry{{"GitHub", "alice", "pw3"}, {"github", "bob", "pw2"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	// Secrets must not be on disk in plaintext.
	raw, _ := os.ReadFile(filepath.Join(v.Dir, entriesFile))
	if bytes.Contains(raw, []byte("alice")) || bytes.Contains(raw, []byte("pw3")) {
		t.Error("plaintext found in vault file")
	}

	if err := v.Delete("github", "alice"); err != nil {
		t.Fatal(err)
	}
	if err := v.Delete("github", "alice"); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: got %v", err)
	}
	if err := v.Delete("github", "bob"); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Get("github"); !errors.Is(err, ErrNotFound) {
		t.Errorf("get after delete: got %v", err)
	}
}

func TestFilePermissions(t *testing.T) {
	v := newTestVault(t)
	v.Put("x", "y", "z")
	for _, f := range []string{saltFile, verifierFile, entriesFile} {
		info, err := os.Stat(filepath.Join(v.Dir, f))
		if err != nil {
			t.Fatal(err)
		}
		if os.PathSeparator == '/' && info.Mode().Perm() != 0o600 {
			t.Errorf("%s has mode %v, want 0600", f, info.Mode().Perm())
		}
	}
}

// A vault written by the original version (v1, 2024) must still open.
func TestLegacyVaultCompatible(t *testing.T) {
	dir := t.TempDir()
	salt := []byte("0123456789abcdef")
	key, _ := deriveKey("legacy-pass", salt)
	os.WriteFile(filepath.Join(dir, saltFile), []byte("30313233343536373839616263646566"), 0o644)
	os.WriteFile(filepath.Join(dir, verifierFile), []byte(keyVerifier(key)), 0o644)
	u, _ := encrypt("user", key)
	p, _ := encrypt("pass", key)
	os.WriteFile(filepath.Join(dir, entriesFile),
		[]byte(`[{"service":"site","username":"`+u+`","password":"`+p+`"}]`), 0o644)

	v := &Vault{Dir: dir}
	if err := v.Unlock("legacy-pass"); err != nil {
		t.Fatal(err)
	}
	got, err := v.Get("site")
	if err != nil || got[0].Username != "user" || got[0].Password != "pass" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestSplitArgs(t *testing.T) {
	cases := map[string][]string{
		`add -s github -u me`:       {"add", "-s", "github", "-u", "me"},
		`add -s "My Bank" -u 'a b'`: {"add", "-s", "My Bank", "-u", "a b"},
		`add -p pa\ ss`:             {"add", "-p", "pa ss"},
		`add -s="quoted value"`:     {"add", "-s=quoted value"},
		`  spaced    out  `:         {"spaced", "out"},
		`empty ""`:                  {"empty", ""},
	}
	for in, want := range cases {
		got, err := splitArgs(in)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("splitArgs(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := splitArgs(`add "unterminated`); err == nil {
		t.Error("expected error for unterminated quote")
	}
}

func TestGeneratePassword(t *testing.T) {
	pw, err := generatePassword(32)
	if err != nil || len(pw) != 32 {
		t.Fatalf("got %q, %v", pw, err)
	}
	if _, err := generatePassword(2); err == nil {
		t.Error("expected error for too-short length")
	}
}

func TestREPL(t *testing.T) {
	v := newTestVault(t)
	var out bytes.Buffer
	a := &app{vault: v, out: &out, secret: func(string) (string, error) { return "hidden-pw", nil }}
	a.repl(strings.NewReader(strings.Join([]string{
		`add -s "My Bank" -u alice`,
		`add -s mail -u bob -p "has space"`,
		`add -s bad`,
		`add -x`,
		`list`,
		`get my bank`,
		`get -s mail`,
		`delete -s mail -u bob`,
		`bogus`,
		`exit`,
		`list`, // never reached
	}, "\n")))

	s := out.String()
	for _, want := range []string{
		"Saved alice for My Bank.",
		"Saved bob for mail.",
		"Usage: add",
		"flag provided but not defined: -x",
		"  mail\n  My Bank\n",
		"username: alice  password: hidden-pw",
		"username: bob  password: has space",
		"Deleted bob from mail.",
		`Unknown command "bogus"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q\n--- output ---\n%s", want, s)
		}
	}
	if strings.Count(s, "  My Bank\n") != 1 {
		t.Error("commands after exit were executed")
	}
}
