package templater

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

// Data holds the template data passed to each file.
type Data struct {
	Name           string
	PackageManager string
}

// Process reads all files from the specified directory within the provided filesystem,
// executes them as text/templates with the provided data, and writes the results
// to the destination directory, preserving the directory structure.
// Files named POST_QUICKSTART.md are skipped and not copied to the destination.
func Process(fsys fs.FS, srcDir, dstDir string, data Data) error {
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

// ProcessMultiSource processes templates from multiple source directories (shared + package-manager-specific).
// It first processes files from the shared directory, then overlays package-manager-specific files.
// The selection's use case picks the template root. The default use case reads
// from templates/; any other use case reads from templates/use-cases/USE_CASE/.
// Under that root, languages that support multiple package managers (python, typescript) expect:
//   - shared directory: ROOT/LANG/shared/
//   - package manager directory: ROOT/LANG/PACKAGE_MANAGER/
//
// For languages with a single package manager (go), it falls back to the language root directory.
func ProcessMultiSource(fsys fs.FS, sel Selection, dstDir string, data Data) error {
	root := sel.templateRoot()

	// For Go, use the old behavior (no shared directory)
	if sel.Language == "go" {
		return Process(fsys, path.Join(root, "go"), dstDir, data)
	}

	// For Python and TypeScript, process shared + package-manager-specific
	sharedDir := path.Join(root, sel.Language, "shared")
	pkgMgrDir := path.Join(root, sel.Language, sel.PackageManager)

	// Process shared files first
	if err := Process(fsys, sharedDir, dstDir, data); err != nil {
		return err
	}

	// Process package-manager-specific files (may overwrite shared files)
	if err := Process(fsys, pkgMgrDir, dstDir, data); err != nil {
		return err
	}

	return nil
}

// ProcessPostQuickstart reads and processes the POST_QUICKSTART.md file from the template directory.
// Returns the processed content as a string, or empty string if the file doesn't exist.
func ProcessPostQuickstart(fsys fs.FS, srcDir string, data Data) (string, error) {
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

// ProcessPostQuickstartMultiSource reads and processes the POST_QUICKSTART.md file from the package-manager-specific directory.
// The selection's use case picks the template root, as in ProcessMultiSource.
// For languages with multiple package managers, it looks in ROOT/LANG/PACKAGE_MANAGER/.
// For Go, it looks in the ROOT/go/ directory.
// Returns the processed content as a string, or empty string if the file doesn't exist.
func ProcessPostQuickstartMultiSource(fsys fs.FS, sel Selection, data Data) (string, error) {
	root := sel.templateRoot()

	var srcDir string
	if sel.Language == "go" {
		srcDir = path.Join(root, "go")
	} else {
		srcDir = path.Join(root, sel.Language, sel.PackageManager)
	}

	return ProcessPostQuickstart(fsys, srcDir, data)
}
