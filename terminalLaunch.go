package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"golang.org/x/term"
)

func isRunningInTerminal() bool {
	// check if program is running in terminal already
	return term.IsTerminal(int(os.Stdin.Fd())) &&
		term.IsTerminal(int(os.Stdout.Fd())) &&
		term.IsTerminal(int(os.Stderr.Fd()))
}

func launchInNewTerminal() error {
	// determine the OS and launch a terminal based on it
	switch runtime.GOOS {
	case "windows":
		return launchInWindowsTerminal()
	case "linux":
		return launchInLinuxTerminal()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func launchInWindowsTerminal() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	cmd := exec.Command("cmd", "/C", "start", "cmd", "/K", exePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	return cmd.Start()
}

func launchInLinuxTerminal() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	cmd := exec.Command("gnome-terminal", "--", exePath)
	return cmd.Start()
}
