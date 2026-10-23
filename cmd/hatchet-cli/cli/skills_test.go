package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func managedSection(body string) string {
	return skillsMarkerStart + "\n" + body + "\n" + skillsMarkerEnd
}

func TestReplaceManagedSection(t *testing.T) {
	entry := managedSection("new content")

	tests := []struct {
		name     string
		existing string
		expected string
		ok       bool
	}{
		{
			name:     "replaces the only section",
			existing: "# Project\n\n" + managedSection("old content") + "\n\ntrailing notes\n",
			expected: "# Project\n\n" + managedSection("new content") + "\n\ntrailing notes\n",
			ok:       true,
		},
		{
			name: "collapses duplicate sections from older installers",
			existing: "# Project\n\n" + managedSection("old one") + "\n\n" +
				managedSection("old two") + "\n\nbetween\n\n" + managedSection("old three") + "\n",
			ok: true,
		},
		{
			name:     "no markers in existing",
			existing: "# Project\n\nno managed section here\n",
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := replaceManagedSection(tt.existing, entry)

			assert.Equal(t, tt.ok, ok)

			if !tt.ok {
				return
			}

			assert.Equal(t, 1, strings.Count(got, skillsMarkerStart), "exactly one start marker should remain")
			assert.Equal(t, 1, strings.Count(got, skillsMarkerEnd), "exactly one end marker should remain")
			assert.Contains(t, got, "new content")
			assert.NotContains(t, got, "old")

			if tt.expected != "" {
				assert.Equal(t, tt.expected, got)
			}
		})
	}

	t.Run("duplicate collapse preserves surrounding content", func(t *testing.T) {
		existing := "# Project\n\n" + managedSection("old one") + "\n\nbetween\n\n" + managedSection("old two") + "\n\nafter\n"
		got, ok := replaceManagedSection(existing, entry)

		assert.True(t, ok)
		assert.Contains(t, got, "# Project")
		assert.Contains(t, got, "between")
		assert.Contains(t, got, "after")
	})

	t.Run("entry without markers fails", func(t *testing.T) {
		_, ok := replaceManagedSection(managedSection("old"), "no markers")
		assert.False(t, ok)
	})
}
