package patternmatcher

import (
	"os"
	"path/filepath"
	"strings"
)

// DirMatches returns true if "dir" should be excluded based on the patterns.
// It returns false if the directory is explicitly re-included by an exclusion pattern
// (patterns starting with '!').
//
// The "dir" argument should be a slash-delimited path.
//
// DirMatches is not safe to call concurrently.
func (pm *PatternMatcher) DirMatches(dir string) (bool, error) {
	dir = filepath.FromSlash(dir)

	// Ensure dir doesn't have trailing separator for consistent matching
	dir = strings.TrimSuffix(dir, string(os.PathSeparator))

	if dir == "." {
		return true, nil
	}

	// Only check exclusion patterns (those starting with '!')
	for _, pattern := range pm.patterns {
		if !pattern.exclusion {
			continue
		}

		// Check if the directory itself matches the exclusion pattern
		match, err := pattern.match(dir)
		if err != nil {
			return false, err
		}

		if match {
			return false, nil
		}

		// Also check if the pattern would match contents of this directory
		// by appending a dummy path segment. This handles patterns like "!**/node_modules/**"
		testPath := dir + string(os.PathSeparator) + "x"
		match, _ = pattern.match(testPath)
		if match {
			return false, nil
		}
	}

	return true, nil
}
