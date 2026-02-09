package patternmatcher

import (
	"testing"
)

func TestDirMatches(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		dir      string
		expected bool
	}{
		// Basic exclusion patterns
		{
			name:     "no patterns returns true",
			patterns: []string{},
			dir:      "somedir",
			expected: true,
		},
		{
			name:     "dot directory returns true",
			patterns: []string{"!**/foo"},
			dir:      ".",
			expected: true,
		},
		// Node modules style exclusions
		{
			name:     "node_modules exclusion matches node_modules dir",
			patterns: []string{"!**/node_modules/**"},
			dir:      "node_modules",
			expected: false,
		},
		{
			name:     "node_modules exclusion matches nested node_modules",
			patterns: []string{"!**/node_modules/**"},
			dir:      "project/node_modules",
			expected: false,
		},
		{
			name:     "node_modules exclusion matches dir inside node_modules",
			patterns: []string{"!**/node_modules/**"},
			dir:      "node_modules/lodash",
			expected: false,
		},
		{
			name:     "node_modules exclusion matches deeply nested dir",
			patterns: []string{"!**/node_modules/**"},
			dir:      "node_modules/lodash/lib",
			expected: false,
		},
		{
			name:     "node_modules exclusion matches nested project node_modules content",
			patterns: []string{"!**/node_modules/**"},
			dir:      "project/node_modules/package/src",
			expected: false,
		},
		{
			name:     "non-matching dir returns true",
			patterns: []string{"!**/node_modules/**"},
			dir:      "src",
			expected: true,
		},
		{
			name:     "non-matching nested dir returns true",
			patterns: []string{"!**/node_modules/**"},
			dir:      "src/components",
			expected: true,
		},
		// Specific directory exclusions
		{
			name:     "specific dir exclusion matches exact dir",
			patterns: []string{"!docs"},
			dir:      "docs",
			expected: false,
		},
		{
			name:     "specific dir exclusion does not match other dirs",
			patterns: []string{"!docs"},
			dir:      "src",
			expected: true,
		},
		{
			name:     "specific dir exclusion does not match nested same-name dir",
			patterns: []string{"!docs"},
			dir:      "project/docs",
			expected: true,
		},
		// Wildcard exclusions
		{
			name:     "wildcard exclusion matches dir",
			patterns: []string{"!**/vendor/**"},
			dir:      "vendor",
			expected: false,
		},
		{
			name:     "wildcard exclusion matches nested vendor",
			patterns: []string{"!**/vendor/**"},
			dir:      "pkg/vendor/github.com",
			expected: false,
		},
		// Multiple exclusion patterns
		{
			name:     "multiple exclusions first matches",
			patterns: []string{"!**/node_modules/**", "!**/vendor/**"},
			dir:      "node_modules/foo",
			expected: false,
		},
		{
			name:     "multiple exclusions second matches",
			patterns: []string{"!**/node_modules/**", "!**/vendor/**"},
			dir:      "vendor/foo",
			expected: false,
		},
		{
			name:     "multiple exclusions none match",
			patterns: []string{"!**/node_modules/**", "!**/vendor/**"},
			dir:      "src/lib",
			expected: true,
		},
		// Mixed inclusion and exclusion patterns (only exclusions should be checked)
		{
			name:     "inclusion pattern ignored",
			patterns: []string{"*.go", "!**/testdata/**"},
			dir:      "testdata",
			expected: false,
		},
		{
			name:     "inclusion pattern ignored non-matching exclusion",
			patterns: []string{"*.go", "!**/testdata/**"},
			dir:      "src",
			expected: true,
		},
		// Path with trailing slash handling
		{
			name:     "dir with trailing slash",
			patterns: []string{"!**/build/**"},
			dir:      "build/",
			expected: false,
		},
		// Exclusion with specific path
		{
			name:     "exclusion with path matches",
			patterns: []string{"!util/docker/web"},
			dir:      "util/docker/web",
			expected: false,
		},
		{
			name:     "exclusion with path does not match partial",
			patterns: []string{"!util/docker/web"},
			dir:      "util/docker",
			expected: true,
		},
		// Single star exclusions
		{
			name:     "single star exclusion in subdir",
			patterns: []string{"!docs/*"},
			dir:      "docs/api",
			expected: false,
		},
		{
			name:     "single star exclusion does not match deeper",
			patterns: []string{"!docs/*"},
			dir:      "docs/api/v1",
			expected: true,
		},
		// Special characters in patterns
		{
			name:     "pattern with special chars",
			patterns: []string{"!**/a(b)c/**"},
			dir:      "a(b)c",
			expected: false,
		},
		{
			name:     "pattern with special chars nested",
			patterns: []string{"!**/a.|)$(}+{bc/**"},
			dir:      "foo/a.|)$(}+{bc/bar",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm, err := New(tt.patterns)
			if err != nil {
				t.Fatalf("failed to create pattern matcher: %v", err)
			}

			result, err := pm.DirMatches(tt.dir)
			if err != nil {
				t.Fatalf("DirMatches returned error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("DirMatches(%q) with patterns %v = %v, want %v",
					tt.dir, tt.patterns, result, tt.expected)
			}
		})
	}
}

func TestDirMatchesError(t *testing.T) {
	// Test with malformed pattern
	_, err := New([]string{"!["})
	if err == nil {
		t.Error("expected error for malformed pattern, got nil")
	}
}
