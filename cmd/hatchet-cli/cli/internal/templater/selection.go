package templater

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// DefaultUseCase maps to the template tree root, which released CLI versions
// already generate from.
const DefaultUseCase = "simple"

// The layout under useCasesDir is a contract with the hatchet-quickstarts module.
const useCasesDir = "templates/use-cases"

// Selection identifies one template to generate.
type Selection struct {
	UseCase        string
	Language       string
	PackageManager string
}

// languageOrder is the order languages appear in prompts and error messages.
var languageOrder = []string{"python", "typescript", "go"}

// Each language has one set of package managers, the same for every use case.
var packageManagersByLanguage = map[string][]string{
	"python":     {"poetry", "uv", "pip"},
	"typescript": {"npm", "pnpm", "yarn", "bun"},
	"go":         {"go"},
}

func (s Selection) normalizedUseCase() string {
	if s.UseCase == "" {
		return DefaultUseCase
	}

	return s.UseCase
}

func (s Selection) templateRoot() string {
	useCase := s.normalizedUseCase()

	if useCase == DefaultUseCase {
		return "templates"
	}

	return path.Join(useCasesDir, useCase)
}

// UseCases returns the selectable use cases. The default comes first, then
// the use cases found in the filesystem, sorted.
func UseCases(fsys fs.FS) ([]string, error) {
	useCases := []string{DefaultUseCase}

	entries, err := fs.ReadDir(fsys, useCasesDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return useCases, nil
		}

		return nil, err
	}

	var discovered []string
	for _, entry := range entries {
		if entry.IsDir() {
			discovered = append(discovered, entry.Name())
		}
	}

	sort.Strings(discovered)

	return append(useCases, discovered...), nil
}

// LanguagesFor returns the languages the use case supports, in the order of
// languageOrder. A language is supported when its template tree exists under
// the use case's root, meaning the language directory itself for go and the
// shared subdirectory for python and typescript.
func LanguagesFor(fsys fs.FS, useCase string) ([]string, error) {
	root := Selection{UseCase: useCase}.templateRoot()

	var languages []string

	for _, language := range languageOrder {
		dir := path.Join(root, language)
		if language != "go" {
			dir = path.Join(dir, "shared")
		}

		if _, err := fs.Stat(fsys, dir); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}

			return nil, err
		}

		languages = append(languages, language)
	}

	return languages, nil
}

// PackageManagersFor returns the package managers the language supports.
func PackageManagersFor(language string) []string {
	return append([]string(nil), packageManagersByLanguage[language]...)
}

// Validate checks that the selection names an existing use case, a language
// that use case supports, and a package manager that language supports.
func Validate(fsys fs.FS, sel Selection) error {
	useCase := sel.normalizedUseCase()

	useCases, err := UseCases(fsys)
	if err != nil {
		return err
	}

	if !contains(useCases, useCase) {
		return fmt.Errorf("unknown use case %q (available: %s)", useCase, strings.Join(useCases, ", "))
	}

	if !contains(languageOrder, sel.Language) {
		return fmt.Errorf("invalid language: %s (must be one of: %s)", sel.Language, strings.Join(languageOrder, ", "))
	}

	languages, err := LanguagesFor(fsys, useCase)
	if err != nil {
		return err
	}

	if !contains(languages, sel.Language) {
		return fmt.Errorf("use case %q does not support language %q (supported: %s)", useCase, sel.Language, strings.Join(languages, ", "))
	}

	if !contains(packageManagersByLanguage[sel.Language], sel.PackageManager) {
		return fmt.Errorf("invalid package manager '%s' for language '%s' (must be one of: %s)", sel.PackageManager, sel.Language, strings.Join(packageManagersByLanguage[sel.Language], ", "))
	}

	return nil
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}

	return false
}
