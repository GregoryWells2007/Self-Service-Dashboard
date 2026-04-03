package helpers

import (
	"net/http"
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

func Mkdir(path string, perm os.FileMode) error {
	logging.Infof("Making directory %s", path)
	err := os.Mkdir(path, perm)
	if err != nil {
		logging.Errorf("Failed to make %s directory", path)
		logging.Error(err.Error())
		return err
	}
	return nil
}

func CreateFile(path string) (*os.File, error) {
	logging.Infof("Creating %s", path)
	file, err := os.Create(path)
	if err != nil {
		logging.Errorf("Faile to create %s", path)
		logging.Error(err.Error())
	}
	return file, nil
}

func HandleFunc(path string, handler func(http.ResponseWriter, *http.Request)) {
	logging.Infof("Handling %s", path)
	http.HandleFunc(path, handler)
}
