package main

import (
	"encoding/json"
	"fmt"
	"os"
)

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
