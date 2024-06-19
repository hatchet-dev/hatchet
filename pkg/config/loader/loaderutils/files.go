package loaderutils

import (
	"fmt"
	"os"
)

func GetConfigBytes(configFilePath string) ([][]byte, error) {
	configFileBytes := make([][]byte, 0)

	if fileExists(configFilePath) {
		fileBytes, err := os.ReadFile(configFilePath) // #nosec G304 -- config files are meant to be read from user-supplied directory

		if err != nil {
			return nil, fmt.Errorf("could not read config file at path %s: %w", configFilePath, err)
		}

		configFileBytes = append(configFileBytes, fileBytes)
	}

	return configFileBytes, nil
}

func GetFileBytes(filename string) ([]byte, error) {
	if fileExists(filename) {
		fileBytes, err := os.ReadFile(filename) // #nosec G304 -- config files are meant to be read from user-supplied directory

		if err != nil {
			return nil, fmt.Errorf("could not read config file at path %s: %w", filename, err)
		}

		return fileBytes, nil
	}

	return nil, fmt.Errorf("could not read config file at path %s", filename)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil && os.IsNotExist(err) {
		return false
	} else if err != nil {
		return false
	}

	return !info.IsDir()
}
