package hatchet

import "testing"

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input               string
		major, minor, patch int
	}{
		{"v0.78.23", 0, 78, 23},
		{"v1.2.3", 1, 2, 3},
		{"0.78.23", 0, 78, 23},
		{"v0.1.0-alpha.0", 0, 1, 0},
		{"v10.20.30-rc.1", 10, 20, 30},
		{"", 0, 0, 0},
		{"v1.2", 0, 0, 0},
		{"not-a-version", 0, 0, 0},
	}

	for _, tt := range tests {
		major, minor, patch := ParseSemver(tt.input)
		if major != tt.major || minor != tt.minor || patch != tt.patch {
			t.Errorf("ParseSemver(%q) = (%d, %d, %d), want (%d, %d, %d)",
				tt.input, major, minor, patch, tt.major, tt.minor, tt.patch)
		}
	}
}

func TestSemverLessThan(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"v0.78.22", "v0.78.23", true},
		{"v0.78.23", "v0.78.23", false},
		{"v0.78.24", "v0.78.23", false},
		{"v0.77.99", "v0.78.0", true},
		{"v0.79.0", "v0.78.99", false},
		{"v0.78.23", "v1.0.0", true},
		{"v1.0.0", "v0.99.99", false},
		{"v0.1.0-alpha.0", "v0.78.23", true},
		{"", "v0.78.23", true},
		{"v0.78.23", "", false},
	}

	for _, tt := range tests {
		got := SemverLessThan(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("SemverLessThan(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
