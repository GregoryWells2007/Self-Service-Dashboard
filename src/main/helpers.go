package main

import (
	"os"

	"astraltech.xyz/accountmanager/src/logging"
)

// Reads a file, if fails just returns an error
func ReadFile(path string) ([]byte, error) {
	logging.Event(logging.ReadFile, "static/blank_profile.jpg")
	data, err := os.ReadFile(path)
	if err != nil {
		logging.Infof("Could not read file at %s", path)
		logging.Infof("Error code: %e", err)
		return nil, err
	}
	logging.Infof("Successfully read file at %s", path)
	return data, err
}

func ReadRequiredFile(path string) []byte {
	logging.Event(logging.ReadFile, "static/blank_profile.jpg")
	data, err := os.ReadFile(path)
	if err != nil {
		logging.Fatalf("Could not read file at %s", path)
		return nil
	}
	logging.Infof("Successfully read file at %s", path)
	return data
}
