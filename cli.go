package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const helpText = `Commands:
  add -s <service> -u <username> [-p <password> | -g [-l N]]
        Store a credential. Omit -p to type the password hidden,
        or use -g to generate a random one (length -l, default 20).
        Adding the same service + username again updates the password.
  get -s <service>          Show all credentials for a service
  list                      List stored services
  delete -s <service> -u <username>
                            Remove a credential
  gen [-l N]                Print a random password (not saved)
  help                      Show this help
  exit                      Quit (Ctrl+D also works)

Wrap values containing spaces in quotes:  add -s "My Bank" -u me -g`

// readSecret reads a line from the terminal without echoing it.
func readSecret(prompt string) (string, error) {
	fmt.Print(prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	return string(b), err
}

// splitArgs splits a command line into fields, honouring single and double
// quotes and backslash escapes, so values may contain spaces.
func splitArgs(line string) ([]string, error) {
	var (
		args    []string
		cur     strings.Builder
		inArg   bool
		quote   rune
		escaped bool
	)
	for _, r := range line {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case r == '\\' && quote != '\'':
			escaped, inArg = true, true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote, inArg = r, true
		case r == ' ' || r == '\t':
			if inArg {
				args = append(args, cur.String())
				cur.Reset()
				inArg = false
			}
		default:
			cur.WriteRune(r)
			inArg = true
		}
	}
	if quote != 0 {
		return nil, errors.New("unterminated quote")
	}
	if escaped {
		cur.WriteRune('\\')
	}
	if inArg {
		args = append(args, cur.String())
	}
	return args, nil
}

// newFlags builds a FlagSet that reports errors instead of exiting the program.
func newFlags(name string, out io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(out)
	return fs
}

type app struct {
	vault *Vault
	out   io.Writer
	// secret prompts for hidden input; replaceable in tests.
	secret func(prompt string) (string, error)
}

// run executes one command line. It returns false when the user wants to quit.
func (a *app) run(line string) bool {
	args, err := splitArgs(strings.TrimSpace(line))
	if err != nil {
		fmt.Fprintln(a.out, "Error:", err)
		return true
	}
	if len(args) == 0 {
		return true
	}
	cmd, rest := strings.ToLower(args[0]), args[1:]
	switch cmd {
	case "add":
		a.add(rest)
	case "get":
		a.get(rest)
	case "list", "ls":
		a.list()
	case "delete", "del", "rm":
		a.delete(rest)
	case "gen", "generate":
		a.gen(rest)
	case "help", "?":
		fmt.Fprintln(a.out, helpText)
	case "exit", "quit", "q":
		return false
	default:
		fmt.Fprintf(a.out, "Unknown command %q. Type 'help' for a list of commands.\n", cmd)
	}
	return true
}

func (a *app) add(args []string) {
	fs := newFlags("add", a.out)
	service := fs.String("s", "", "service name")
	username := fs.String("u", "", "username")
	password := fs.String("p", "", "password (omit to be prompted)")
	generate := fs.Bool("g", false, "generate a random password")
	length := fs.Int("l", 20, "generated password length")
	if fs.Parse(args) != nil {
		return
	}
	if *service == "" || *username == "" {
		fmt.Fprintln(a.out, "Usage: add -s <service> -u <username> [-p <password> | -g [-l N]]")
		return
	}
	pw := *password
	switch {
	case *generate && pw != "":
		fmt.Fprintln(a.out, "Use either -p or -g, not both.")
		return
	case *generate:
		var err error
		if pw, err = generatePassword(*length); err != nil {
			fmt.Fprintln(a.out, "Error:", err)
			return
		}
	case pw == "":
		var err error
		if pw, err = a.secret("Password: "); err != nil {
			fmt.Fprintln(a.out, "Error:", err)
			return
		}
		if pw == "" {
			fmt.Fprintln(a.out, "Password cannot be empty.")
			return
		}
	}
	updated, err := a.vault.Put(*service, *username, pw)
	if err != nil {
		fmt.Fprintln(a.out, "Error saving entry:", err)
		return
	}
	verb := "Saved"
	if updated {
		verb = "Updated"
	}
	fmt.Fprintf(a.out, "%s %s for %s.\n", verb, *username, *service)
	if *generate {
		fmt.Fprintln(a.out, "Generated password:", pw)
	}
}

func (a *app) get(args []string) {
	fs := newFlags("get", a.out)
	service := fs.String("s", "", "service name")
	if fs.Parse(args) != nil {
		return
	}
	if *service == "" && fs.NArg() > 0 {
		*service = strings.Join(fs.Args(), " ") // allow "get my bank" as a shortcut
	}
	if *service == "" {
		fmt.Fprintln(a.out, "Usage: get -s <service>")
		return
	}
	entries, err := a.vault.Get(*service)
	if errors.Is(err, ErrNotFound) {
		fmt.Fprintf(a.out, "No entries for %q. Use 'list' to see stored services.\n", *service)
		return
	}
	if err != nil {
		fmt.Fprintln(a.out, "Error:", err)
		return
	}
	for _, e := range entries {
		fmt.Fprintf(a.out, "%s  username: %s  password: %s\n", e.Service, e.Username, e.Password)
	}
}

func (a *app) list() {
	names, counts, err := a.vault.Services()
	if err != nil {
		fmt.Fprintln(a.out, "Error:", err)
		return
	}
	if len(names) == 0 {
		fmt.Fprintln(a.out, "The vault is empty. Add something with 'add'.")
		return
	}
	for _, n := range names {
		suffix := ""
		if counts[n] > 1 {
			suffix = fmt.Sprintf(" (%d accounts)", counts[n])
		}
		fmt.Fprintf(a.out, "  %s%s\n", n, suffix)
	}
}

func (a *app) delete(args []string) {
	fs := newFlags("delete", a.out)
	service := fs.String("s", "", "service name")
	username := fs.String("u", "", "username")
	if fs.Parse(args) != nil {
		return
	}
	if *service == "" || *username == "" {
		fmt.Fprintln(a.out, "Usage: delete -s <service> -u <username>")
		return
	}
	err := a.vault.Delete(*service, *username)
	if errors.Is(err, ErrNotFound) {
		fmt.Fprintf(a.out, "No entry for %s on %s.\n", *username, *service)
		return
	}
	if err != nil {
		fmt.Fprintln(a.out, "Error:", err)
		return
	}
	fmt.Fprintf(a.out, "Deleted %s from %s.\n", *username, *service)
}

func (a *app) gen(args []string) {
	fs := newFlags("gen", a.out)
	length := fs.Int("l", 20, "password length")
	if fs.Parse(args) != nil {
		return
	}
	pw, err := generatePassword(*length)
	if err != nil {
		fmt.Fprintln(a.out, "Error:", err)
		return
	}
	fmt.Fprintln(a.out, pw)
}

// repl reads commands until exit or end of input.
func (a *app) repl(in io.Reader) {
	fmt.Fprintln(a.out, "Type 'help' to see available commands.")
	sc := bufio.NewScanner(in)
	for {
		fmt.Fprint(a.out, "> ")
		if !sc.Scan() {
			fmt.Fprintln(a.out)
			return
		}
		if !a.run(sc.Text()) {
			return
		}
	}
}
