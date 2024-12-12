package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	sourceDir := "./sql/migrations"
	targetDir := "./sql/goose"

	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		fmt.Printf("Error creating target directory: %v\n", err)
		return
	}

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".sql") {
			newFilePath := filepath.Join(targetDir, info.Name())
			if _, err := os.Stat(newFilePath); err == nil {
				fmt.Printf("Skipping existing file: %s\n", newFilePath)
				return nil
			}

			if err := convertMigrationFile(path, newFilePath); err != nil {
				fmt.Printf("Error converting file %s: %v\n", info.Name(), err)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking through source directory: %v\n", err)
	}
}

func convertMigrationFile(sourcePath, targetPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer targetFile.Close()

	scanner := bufio.NewScanner(sourceFile)
	writer := bufio.NewWriter(targetFile)
	defer writer.Flush()

	write(writer, "-- +goose Up\n")

	insideFunction := false
	insideDoBlock := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "-- atlas:txmode none") {
			write(writer, "-- +goose NO TRANSACTION\n") // golint: errcheck
			continue
		}

		if strings.HasPrefix(line, "CREATE OR REPLACE FUNCTION") {
			write(writer, "-- +goose StatementBegin\n") // golint: errcheck
			insideFunction = true
		}

		if strings.HasPrefix(line, "DO $$") {
			write(writer, "-- +goose StatementBegin\n") // golint: errcheck
			insideDoBlock = true
		}

		write(writer, line+"\n") // golint: errcheck

		if insideFunction && strings.HasSuffix(line, "$$ LANGUAGE plpgsql;") {
			write(writer, "-- +goose StatementEnd\n") // golint: errcheck
			insideFunction = false
		}

		if insideDoBlock && strings.HasPrefix(line, "END $$;") {
			write(writer, "-- +goose StatementEnd\n") // golint: errcheck
			insideDoBlock = false
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading source file: %w", err)
	}

	return nil
}

func write(writer *bufio.Writer, line string) {
	_, err := writer.WriteString(line + "\n")
	if err != nil {
		panic(err)
	}
}
