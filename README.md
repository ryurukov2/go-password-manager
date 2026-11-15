# go-password-manager

[![CI](https://github.com/ryurukov2/go-password-manager/actions/workflows/ci.yml/badge.svg)](https://github.com/ryurukov2/go-password-manager/actions/workflows/ci.yml)

A small, offline password manager for the terminal, written in Go. Credentials are encrypted with AES-256-GCM using a key derived from a master password. The master password is never stored.

```
$ password-manager
Password Manager v2.0.0
Vault: /home/me/.config/go-password-manager
Master password:
Unlocked.
Type 'help' to see available commands.
> add -s github -u octocat -g
Saved octocat for github.
Generated password: 9vX!q2mR#Lw7zT@k4pHs
> get github
github  username: octocat  password: 9vX!q2mR#Lw7zT@k4pHs
> exit
```

## Features

- **Master password, never stored.** A 256-bit key is derived with PBKDF2-HMAC-SHA256 (100k iterations, random 16-byte salt). Only a SHA-256 hash of the derived key is saved, so the app can check the password at login. The check uses a constant-time comparison.
- **Authenticated encryption.** Each username and password is sealed separately with AES-256-GCM and a fresh random nonce, so any tampering is detected.
- **Hidden input.** The master password and entered passwords are not echoed to the screen. Passwords never need to go on the command line.
- **Password generator.** Uses `crypto/rand` with unbiased sampling.
- **Crash-safe storage.** Writes go to a temp file that is then renamed into place, and files are owner-only (`0600`).
- **Cross-platform.** Runs on Linux, macOS and Windows. If you double-click it, it opens a terminal window for itself.
- No dependencies beyond the Go standard library and `golang.org/x/term`.

## Install

Download a binary for your OS from the [releases page](https://github.com/ryurukov2/go-password-manager/releases/latest).
On Linux or macOS, make it executable first: `chmod +x password-manager-*`.

Or build it from source (requires [Go](https://go.dev/doc/install) 1.24+):

```sh
git clone https://github.com/ryurukov2/go-password-manager
cd go-password-manager
make build         # or simply: go build -o password-manager .
./password-manager
```

## Usage

On the first run you choose a master password, which must be at least 8 characters. **It cannot be recovered**: if you forget it, the stored data is lost.

| Command | Description |
|---|---|
| `add -s <service> -u <user>` | Add a credential; you are prompted for the password (hidden) |
| `add -s <service> -u <user> -g [-l 24]` | Add with a generated password |
| `add -s <service> -u <user> -p <password>` | Add with the password inline (it stays in your scrollback) |
| `get <service>` / `get -s <service>` | Show credentials for a service (case-insensitive) |
| `list` | List stored services |
| `delete -s <service> -u <user>` | Delete a credential |
| `gen [-l 20]` | Print a random password without saving it |
| `help`, `exit` | Show help or quit (Ctrl+D also quits) |

Use quotes for values that contain spaces: `add -s "My Bank" -u "John Doe"`.
If you add the same service and username again, the password is updated.

### Where data is stored

The vault is three files: `salt.txt`, `key_hash.txt` and `passwords.json`. The app looks for them in this order:

1. the directory given with `-dir <path>`
2. `$PASSWORD_MANAGER_DIR`
3. the current directory or the executable's directory, if a vault already exists there (this keeps vaults from v1 working)
4. the OS config directory (`~/.config/go-password-manager`, `~/Library/Application Support/go-password-manager`, or `%AppData%\go-password-manager`)

To back up the vault, copy those three files. Service names are stored in plaintext; usernames and passwords are encrypted.

## Development

```sh
make test      # go vet + tests with the race detector
make release   # cross-compile into dist/
```

To publish a release, push a tag (`git tag v2.0.0 && git push --tags`). GitHub Actions runs the tests, builds binaries for every platform and attaches them with checksums.

### Project layout

| File | Responsibility |
|---|---|
| `crypto.go` | Key derivation, AES-GCM encrypt/decrypt, password generation |
| `vault.go` | Vault setup/unlock and CRUD on the encrypted store |
| `cli.go` | Interactive shell, quote-aware argument parsing, commands |
| `terminal.go` | Detecting a terminal and relaunching inside one |
| `main.go` | Flags, choosing the vault location, login flow |

## Limitations

This is a learning project, not a replacement for an audited password manager. Known trade-offs:
- Service names are not encrypted.
- PBKDF2 was kept for compatibility with existing vaults; Argon2id would resist GPU attacks better.
- Decrypted secrets are held as Go strings, which cannot be reliably wiped from memory.
