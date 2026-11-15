package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/term"
)

func isRunningInTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// launchInNewTerminal re-runs this executable inside a terminal window.
// This lets the program work when it is double-clicked from a file manager.
func launchInNewTerminal() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable path: %w", err)
	}
	args := os.Args[1:]

	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", append([]string{"/C", "start", "cmd", "/K", exe}, args...)...).Start()
	case "darwin":
		return exec.Command("open", "-a", "Terminal", exe).Start()
	case "linux", "freebsd", "openbsd", "netbsd":
		// Try common terminal emulators in order.
		candidates := [][]string{
			{"x-terminal-emulator", "-e"},
			{"gnome-terminal", "--"},
			{"konsole", "-e"},
			{"xfce4-terminal", "-x"},
			{"kitty"},
			{"alacritty", "-e"},
			{"xterm", "-e"},
		}
		for _, c := range candidates {
			if _, err := exec.LookPath(c[0]); err != nil {
				continue
			}
			cmdArgs := append(append(c[1:], exe), args...)
			return exec.Command(c[0], cmdArgs...).Start()
		}
		return errors.New("no supported terminal emulator found")
	default:
		return fmt.Errorf("unsupported operating system %s", runtime.GOOS)
	}
}
