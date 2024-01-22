package types

import (
	"context"
	"fmt"
	"os"
)

// ReadHatchetYAMLFileBytes reads a given YAML file from a filepath and return the parsed workflow file
func ReadHatchetYAMLFileBytes(filepath string) (*Workflow, error) {
	yamlFileBytes, err := readHatchetYAMLFileBytes(filepath)

	if err != nil {
		return nil, err
	}

	workflowFile, err := ParseYAML(context.Background(), yamlFileBytes)

	if err != nil {
		return nil, err
	}

	return &workflowFile, nil
}

func readHatchetYAMLFileBytes(filepath string) ([]byte, error) {
	if !fileExists(filepath) {
		return nil, fmt.Errorf("file does not exist: %s", filepath)
	}

	yamlFileBytes, err := os.ReadFile(filepath) // #nosec G304 -- files are meant to be read from user-supplied directory

	if err != nil {
		panic(err)
	}

	return yamlFileBytes, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
