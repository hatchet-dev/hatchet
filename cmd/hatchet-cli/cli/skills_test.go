package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestLinkSkillDir(t *testing.T) {
	newSkillDir := func(t *testing.T) (baseDir, skillDir string) {
		t.Helper()
		baseDir = t.TempDir()
		skillDir = filepath.Join(baseDir, ".agents", "skills", "hatchet-cli")
		require.NoError(t, os.MkdirAll(skillDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("skill"), 0o600))
		return baseDir, skillDir
	}

	t.Run("creates a relative symlink with parent dirs", func(t *testing.T) {
		baseDir, skillDir := newSkillDir(t)
		link := filepath.Join(baseDir, ".claude", "skills", "hatchet-cli")

		skipped, err := linkSkillDir(link, skillDir)

		require.NoError(t, err)
		assert.False(t, skipped)

		target, err := os.Readlink(link)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join("..", "..", ".agents", "skills", "hatchet-cli"), target)

		data, err := os.ReadFile(filepath.Join(link, "SKILL.md"))
		require.NoError(t, err)
		assert.Equal(t, "skill", string(data))
	})

	t.Run("replaces an existing symlink", func(t *testing.T) {
		baseDir, skillDir := newSkillDir(t)
		link := filepath.Join(baseDir, ".claude", "skills", "hatchet-cli")
		require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
		require.NoError(t, os.Symlink(filepath.Join(baseDir, "elsewhere"), link))

		skipped, err := linkSkillDir(link, skillDir)

		require.NoError(t, err)
		assert.False(t, skipped)

		_, err = os.Stat(filepath.Join(link, "SKILL.md"))
		assert.NoError(t, err)
	})

	t.Run("skips a real directory without destroying it", func(t *testing.T) {
		baseDir, skillDir := newSkillDir(t)
		link := filepath.Join(baseDir, ".claude", "skills", "hatchet-cli")
		require.NoError(t, os.MkdirAll(link, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(link, "custom.md"), []byte("user data"), 0o600))

		skipped, err := linkSkillDir(link, skillDir)

		require.NoError(t, err)
		assert.True(t, skipped)

		data, err := os.ReadFile(filepath.Join(link, "custom.md"))
		require.NoError(t, err)
		assert.Equal(t, "user data", string(data))
	})
}

func TestRemoveLegacySkillDir(t *testing.T) {
	t.Run("no legacy dir is a no-op", func(t *testing.T) {
		removed, err := removeLegacySkillDir(t.TempDir())
		require.NoError(t, err)
		assert.False(t, removed)
	})

	t.Run("removes legacy dir and empty skills parent", func(t *testing.T) {
		baseDir := t.TempDir()
		legacy := filepath.Join(baseDir, "skills", "hatchet-cli")
		require.NoError(t, os.MkdirAll(filepath.Join(legacy, "references"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(legacy, "SKILL.md"), []byte("skill"), 0o600))

		removed, err := removeLegacySkillDir(baseDir)

		require.NoError(t, err)
		assert.True(t, removed)

		_, err = os.Lstat(filepath.Join(baseDir, "skills"))
		assert.True(t, os.IsNotExist(err), "empty skills/ parent should be pruned")
	})

	t.Run("keeps skills parent when it holds other content", func(t *testing.T) {
		baseDir := t.TempDir()
		legacy := filepath.Join(baseDir, "skills", "hatchet-cli")
		require.NoError(t, os.MkdirAll(legacy, 0o755))
		other := filepath.Join(baseDir, "skills", "other-skill")
		require.NoError(t, os.MkdirAll(other, 0o755))

		removed, err := removeLegacySkillDir(baseDir)

		require.NoError(t, err)
		assert.True(t, removed)

		_, err = os.Lstat(legacy)
		assert.True(t, os.IsNotExist(err))

		_, err = os.Stat(other)
		assert.NoError(t, err)
	})
}
