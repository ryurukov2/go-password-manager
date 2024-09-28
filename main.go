package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type PasswordEntry struct {
	Service  string `json:"service"`
	Username string `json:"username"`
	Password string `json:"password"`
}

const dataFile = "passwords.json"

func readEntries() ([]PasswordEntry, error) {
	var entries []PasswordEntry

	_, err := os.Stat(dataFile)
	if os.IsNotExist(err) {
		return entries, nil
	}

	data, err := os.ReadFile(dataFile)
	if err != nil {
		return nil, fmt.Errorf("could not read file: %v", err)
	}
	err = json.Unmarshal(data, &entries)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal data: %v", err)
	}
	return entries, nil
}

func saveEntry(entry PasswordEntry) error {
	entries, err := readEntries()
	if err != nil {
		return fmt.Errorf("could not read entries: %v", err)
	}

	entries = append(entries, entry)

	data, err := json.MarshalIndent(entries, "", " ")
	if err != nil {
		return fmt.Errorf("could not marshal entries: %v", err)
	}

	err = os.WriteFile(dataFile, data, 0644)
	if err != nil {
		return fmt.Errorf("could not write to file: %v", err)
	}

	fmt.Println(("Entry saved successfully!"))
	return nil
}

func addCommand() {
	addCmd := flag.NewFlagSet("add", flag.ExitOnError)
	service := addCmd.String("s", "", "Service name")
	username := addCmd.String("u", "", "Username")
	password := addCmd.String("p", "", "Password")
	addCmd.Parse(os.Args[2:])
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

func getCommand() {
	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	service := getCmd.String("s", "", "Service name")
	getCmd.Parse(os.Args[2:])

	if *service == "" {
		fmt.Println("Usage: get -s=<service>")
		return
	}

	entries, err := readEntries()
	if err != nil {
		fmt.Println("Error reading the file:", err)
		return
	}

	for _, entry := range entries {
		if entry.Service == *service {
			fmt.Printf("Username: %v, Password: %v\n", entry.Username, entry.Password)
			return
		}
	}
	fmt.Println("No entry found for that service.")
}

func saveUpdatedEntries(entries []PasswordEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal updated entries: %v", err)
	}

	err = os.WriteFile(dataFile, data, 0644)
	if err != nil {
		return fmt.Errorf("could not write updated entries to file: %v", err)
	}

	return nil
}
func deleteCommand() {
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	service := deleteCmd.String("s", "", "Service name")
	username := deleteCmd.String("u", "", "Service name")
	deleteCmd.Parse(os.Args[2:])
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
	if len(os.Args) < 2 {
		fmt.Println("Expected 'add', 'get' or 'delete' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "add":
		addCommand()
	case "get":
		getCommand()
	case "delete":
		deleteCommand()
	default:
		fmt.Println("Unknown command:", os.Args[1])
		os.Exit(1)
	}
}
