package templater

import (
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
