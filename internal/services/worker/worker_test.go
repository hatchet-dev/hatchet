package worker

import (
	"reflect"
	"testing"
)

func TestGetSearchPaths(t *testing.T) {
	tests := []struct {
		name       string
		targetPath string
		want       []string
	}{
		{
			name:       "Basic Path",
			targetPath: "/usr/local/hatchet/src/worker.py",
			want: []string{
				"usr/local/hatchet/src/worker.py",
				"local/hatchet/src/worker.py",
				"hatchet/src/worker.py",
				"src/worker.py",
				"worker.py",
			},
		},
		{
			name:       "Root Path",
			targetPath: "/worker.py",
			want:       []string{"worker.py"},
		},
		{
			name:       "Empty Path",
			targetPath: "",
			want:       []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getSearchPaths(tt.targetPath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("[%s] getSearchPaths() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestFindAndReplace(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		value       string
		override    string
		expected    string
		expectError bool
	}{
		{
			name:        "Basic Replacement",
			input:       `context.override("test", "Original override")`,
			value:       "test",
			override:    `"New override"`,
			expected:    `context.override("test", "New override")`,
			expectError: false,
		},
		{
			name:        "Multiple Replacement",
			input:       `context.override("test", "Original override"); context.override("test", "Another original")`,
			value:       "test",
			override:    `"New override"`,
			expected:    `context.override("test", "New override"); context.override("test", "New override")`,
			expectError: false,
		},
		{
			name:        "Replace number",
			input:       `context.override("test", 1234)`,
			value:       "test",
			override:    "5678",
			expected:    `context.override("test", 5678)`,
			expectError: false,
		},
		{
			name:        "No Match",
			input:       `context.override("no_match", "Original override")`,
			value:       "test",
			override:    `"New override"`,
			expected:    `context.override("no_match", "Original override")`,
			expectError: false,
		},
		{
			name:        "Error Handling - Invalid Regex",
			input:       `context.override("test", "Original override")`,
			value:       "(", // This makes the regex invalid
			override:    `"New override"`,
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := findAndReplace([]byte(tt.input), tt.value, tt.override)
			if (err != nil) != tt.expectError {
				t.Errorf("[%s] findAndReplace() error = %v, expectError %v", tt.name, err, tt.expectError)
				return
			}
			if string(result) != tt.expected {
				t.Errorf("[%s] findAndReplace() got = %v, want %v", tt.name, string(result), tt.expected)
			}
		})
	}
}
