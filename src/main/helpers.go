package main

import (
	"os"

	"astraltech.xyz/accountmanager/src/logging"
)

func ReadFile(path string) ([]byte, error) {
	logging.Event(logging.ReadFile, "static/blank_profile.jpg")
	data, err := os.ReadFile(path)
	if err != nil {
		logging.Infof("Could not read file at %s", path)
	}
	logging.Infof("Successfully read file at %s", path)
	return data, err
}
