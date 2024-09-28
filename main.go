package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

type PasswordEntry struct {
	Service  string `json:"service"`
	Username string `json:"username"`
	Password string `json:"password"`
}

const dataFile = "passwords.json"

func addCommand(args []string) {
	addCmd := flag.NewFlagSet("add", flag.ExitOnError)
	service := addCmd.String("s", "", "Service name")
	username := addCmd.String("u", "", "Username")
	password := addCmd.String("p", "", "Password")
	addCmd.Parse(args)
	if *service == "" || *username == "" || *password == "" {
		fmt.Println("Usage: add -s=<service> -u=<username> -p=<password>")
		return
	}
	entry := PasswordEntry{
		Service:  *service,
		Username: *username,
		Password: *password,
	}
	if err := saveEntry(entry); err != nil {
		fmt.Println("Error saving entry:", err)
		return
	}
	fmt.Printf("Added: %s, %s, %s\n", *service, *username, *password)

}

func getCommand(args []string) {
	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	service := getCmd.String("s", "", "Service name")
	getCmd.Parse(args)

	if *service == "" {
		fmt.Println("Usage: get -s=<service>")
		return
	}

	entries, err := readEntries()
	if err != nil {
		fmt.Println("Error reading the file:", err)
		return
	}
	found := false
	for _, entry := range entries {
		if entry.Service == *service {
			fmt.Printf("Username: %v, Password: %v\n", entry.Username, entry.Password)
			found = true
		}
	}
	if !found {
		fmt.Println("No entry found for that service.")
	}
}

func deleteCommand(args []string) {
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	service := deleteCmd.String("s", "", "Service name")
	username := deleteCmd.String("u", "", "Service name")
	deleteCmd.Parse(args)
	if *service == "" || *username == "" {
		fmt.Println("Usage: delete -s=<service> -u=<username>")
		return
	}
	entries, err := readEntries()
	if err != nil {
		fmt.Println("Error reading the file:", err)
	}
	updatedEntries := []PasswordEntry{}
	found := false
	for _, entry := range entries {
		if entry.Service == *service && entry.Username == *username {
			found = true
			fmt.Printf("Deleting entry for username %s for service %s\n", *username, *service)
			continue
		}
		updatedEntries = append(updatedEntries, entry)
	}

	if !found {
		fmt.Printf("No entry found for username %s for service %s\n", *username, *service)
		return
	}

	err = saveUpdatedEntries(updatedEntries)
	if err != nil {
		fmt.Println("Error saving updated entries:", err)
		return
	}
	fmt.Println("Entry deleted successfully.")

}

func main() {
	if !isRunningInTerminal() {
		fmt.Println("Launching in a new terminal...")
		if err := launchInNewTerminal(); err != nil {
			fmt.Printf("Error launching in new terminal: %v\n", err)
		}
		return
	}
	fmt.Println("Welcome to your Password Manager!")

	if _, err := os.Stat("./salt.txt"); errors.Is(err, os.ErrNotExist) {
		setupMasterPassword()
	}

	isAuthd, err := verifyMasterPassword()
	if err != nil {
		fmt.Println("Error verifying master password -", err)
	}

	if !isAuthd {
		fmt.Println("Login unsuccessful. Exiting.")
		os.Exit(1)
	}

	fmt.Println("Available commands: add, get, delete, exit")
	for {

		fmt.Print("> ")

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		input = strings.TrimSpace(input)

		if input == "" {
			fmt.Println("Please enter a command.")
			continue
		}

		args := strings.Fields(input)
		command := args[0]

		switch command {
		case "add":
			addCommand(args[1:])
		case "get":
			getCommand(args[1:])
		case "delete":
			deleteCommand(args[1:])
		case "exit":
			fmt.Println("Exiting the Password Manager. Goodbye!")
			os.Exit(0)
		default:
			fmt.Println("Unknown command:", command)
			fmt.Println("Available commands: add, get, delete, exit")
		}
	}
}
