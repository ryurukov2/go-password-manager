// Command password-manager is a small, local, encrypted password manager for the terminal.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const maxAttempts = 3

var version = "dev" // overridden at build time with -ldflags "-X main.version=..."

func main() {
	dirFlag := flag.String("dir", "", "vault directory (overrides $PASSWORD_MANAGER_DIR)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("password-manager", version)
		return
	}

	// When started by double-clicking, there is no terminal to type into.
	if !isRunningInTerminal() {
		if err := launchInNewTerminal(); err != nil {
			fmt.Fprintln(os.Stderr, "This program must be run from a terminal:", err)
			os.Exit(1)
		}
		return
	}

	dir, err := resolveVaultDir(*dirFlag)
	if err != nil {
		fatal(err)
	}
	vault := &Vault{Dir: dir}

	fmt.Println("Password Manager", version)
	fmt.Println("Vault:", dir)

	if vault.Exists() {
		if err := unlock(vault); err != nil {
			fatal(err)
		}
	} else if err := setup(vault); err != nil {
		fatal(err)
	}

	a := &app{vault: vault, out: os.Stdout, secret: readSecret}
	a.repl(os.Stdin)
	fmt.Println("Goodbye!")
}

func setup(v *Vault) error {
	fmt.Println("No vault found. Let's create one.")
	fmt.Println("Choose a master password. It is never stored, and it cannot be recovered if forgotten.")
	for {
		pw, err := readSecret("New master password: ")
		if err != nil {
			return err
		}
		if len(pw) < 8 {
			fmt.Println("Please use at least 8 characters.")
			continue
		}
		confirm, err := readSecret("Repeat master password: ")
		if err != nil {
			return err
		}
		if pw != confirm {
			fmt.Println("Passwords do not match, try again.")
			continue
		}
		if err := v.Create(pw); err != nil {
			return err
		}
		fmt.Println("Vault created.")
		return nil
	}
}

func unlock(v *Vault) error {
	for i := 1; i <= maxAttempts; i++ {
		pw, err := readSecret("Master password: ")
		if err != nil {
			return err
		}
		err = v.Unlock(pw)
		if err == nil {
			fmt.Println("Unlocked.")
			return nil
		}
		if !errors.Is(err, ErrWrongPassword) {
			return err
		}
		fmt.Printf("Incorrect password (%d/%d).\n", i, maxAttempts)
	}
	return errors.New("too many failed attempts")
}

// resolveVaultDir picks where the vault files live, in priority order:
//  1. the -dir flag
//  2. $PASSWORD_MANAGER_DIR
//  3. the current directory or the executable's directory, if a vault from an
//     older version already exists there (older versions used the working dir)
//  4. the OS config directory, e.g. ~/.config/go-password-manager
func resolveVaultDir(flagDir string) (string, error) {
	if flagDir != "" {
		return filepath.Abs(flagDir)
	}
	if env := os.Getenv("PASSWORD_MANAGER_DIR"); env != "" {
		return filepath.Abs(env)
	}
	var legacy []string
	if wd, err := os.Getwd(); err == nil {
		legacy = append(legacy, wd)
	}
	if exe, err := os.Executable(); err == nil {
		legacy = append(legacy, filepath.Dir(exe))
	}
	for _, d := range legacy {
		if (&Vault{Dir: d}).Exists() {
			return d, nil
		}
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine a vault directory, use -dir: %w", err)
	}
	return filepath.Join(cfg, "go-password-manager"), nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}
