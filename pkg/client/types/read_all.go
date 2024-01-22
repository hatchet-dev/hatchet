package types

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func DefaultLoader() []*Workflow {
	workflowFiles, err := ReadAllValidFilesInDir("./.hatchet")

	if err != nil {
		panic(err)
	}

	return workflowFiles
}

func ReadAllValidFilesInDir(filedir string) ([]*Workflow, error) {
	files, err := readYAMLFiles(filedir)

	if err != nil {
		return nil, err
	}

	var workflowFiles []*Workflow

	for _, file := range files {
		workflowFile, err := ParseYAML(context.Background(), file)

		if err != nil {
			continue
		}

		workflowFiles = append(workflowFiles, &workflowFile)
	}

	return workflowFiles, nil
}

// readYAMLFiles reads all .yaml files in a given directory, including subdirectories.
func readYAMLFiles(rootDir string) ([][]byte, error) {
	yamlFiles := make([][]byte, 0)

	// Walk the directory tree
	err := filepath.WalkDir(rootDir, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check if the file is a YAML file
		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml")) {
			// Read the file
			data, err := os.ReadFile(path) // #nosec G304 -- files are meant to be read from user-supplied directory
			if err != nil {
				return fmt.Errorf("error reading file %s: %v", path, err)
			}

			yamlFiles = append(yamlFiles, data)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking the path %s: %v", rootDir, err)
	}

	return yamlFiles, nil
}
