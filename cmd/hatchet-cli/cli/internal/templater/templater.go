package templater

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Data holds the template data passed to each file.
type Data struct {
	Name string
}

// Process reads all files from the specified directory within the embedded filesystem,
// executes them as text/templates with the provided data, and writes the results
// to the destination directory, preserving the directory structure.
// Files named POST_QUICKSTART.md are skipped and not copied to the destination.
func Process(fsys embed.FS, srcDir, dstDir string, data Data) error {
	// Get a sub-filesystem rooted at srcDir
	subFS, err := fs.Sub(fsys, srcDir)
	if err != nil {
		return err
	}

	// Walk through all files in the embedded filesystem
	return fs.WalkDir(subFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip POST_QUICKSTART.md files - they're for display only, not copying
		if !d.IsDir() && filepath.Base(path) == "POST_QUICKSTART.md" {
			return nil
		}

		// Remove .embed suffix if present for the destination path
		dstPath := filepath.Join(dstDir, path)
		dstPath = strings.TrimSuffix(dstPath, ".embed")

		if d.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}

		content, err := fs.ReadFile(subFS, path)
		if err != nil {
			return err
		}

		tmpl, err := template.New(path).Parse(string(content))
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		outFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer outFile.Close()

		return tmpl.Execute(outFile, data)
	})
}

// ProcessPostQuickstart reads and processes the POST_QUICKSTART.md file from the template directory.
// Returns the processed content as a string, or empty string if the file doesn't exist.
func ProcessPostQuickstart(fsys embed.FS, srcDir string, data Data) (string, error) {
	// Get a sub-filesystem rooted at srcDir
	subFS, err := fs.Sub(fsys, srcDir)
	if err != nil {
		return "", err
	}

	// Try to read POST_QUICKSTART.md
	content, err := fs.ReadFile(subFS, "POST_QUICKSTART.md")
	if err != nil {
		// File doesn't exist, return empty string (not an error)
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	// Process as template
	tmpl, err := template.New("POST_QUICKSTART.md").Parse(string(content))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
