package mcp

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrantsIsGranted(t *testing.T) {
	tests := []struct {
		name    string
		entries []Grant
		query   string
		want    bool
	}{
		{
			name:    "empty grants deny",
			entries: nil,
			query:   "local",
			want:    false,
		},
		{
			name:    "direct grant",
			entries: []Grant{{Profile: "local"}},
			query:   "local",
			want:    true,
		},
		{
			name:    "ungranted profile denied",
			entries: []Grant{{Profile: "local"}},
			query:   "prod",
			want:    false,
		},
		{
			name:    "wildcard grants everything including future profiles",
			entries: []Grant{{Profile: GrantWildcard}},
			query:   "profile-created-later",
			want:    true,
		},
		{
			name:    "empty name never granted even with wildcard",
			entries: []Grant{{Profile: GrantWildcard}},
			query:   "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grants := &Grants{Entries: tt.entries}
			assert.Equal(t, tt.want, grants.IsGranted(tt.query))
		})
	}
}

func TestGrantsAddRemoveClear(t *testing.T) {
	grants := &Grants{}

	assert.True(t, grants.Add("local"))
	assert.False(t, grants.Add("local"), "adding a duplicate should not change the set")
	assert.True(t, grants.Add(GrantWildcard))
	assert.True(t, grants.HasWildcard())

	// wildcard sorts first
	assert.Equal(t, []string{GrantWildcard, "local"}, grants.Names())

	assert.True(t, grants.Remove(GrantWildcard))
	assert.False(t, grants.HasWildcard())
	assert.False(t, grants.Remove("missing"))

	grants.Clear()
	assert.Empty(t, grants.Names())
	assert.False(t, grants.IsGranted("local"))
}

func TestGrantStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewGrantStore(dir)

	// missing file loads as empty
	grants, err := store.Load()
	require.NoError(t, err)
	assert.Empty(t, grants.Names())

	grants.Add("local")
	grants.Add(GrantWildcard)
	require.NoError(t, store.Save(grants))

	// entries are objects ({ profile: name }), not bare strings
	data, err := os.ReadFile(filepath.Join(dir, GrantFileName))
	require.NoError(t, err)
	assert.Contains(t, string(data), "profile: local")
	assert.Contains(t, string(data), `profile: '*'`)

	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dir, GrantFileName))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}

	reloaded, err := store.Load()
	require.NoError(t, err)
	assert.True(t, reloaded.IsGranted("local"))
	assert.True(t, reloaded.IsGranted("future-profile"), "wildcard must cover profiles that do not exist yet")
	assert.Equal(t, []string{GrantWildcard, "local"}, reloaded.Names())
}

func TestGrantStoreLoadDropsMalformedEntries(t *testing.T) {
	dir := t.TempDir()
	store := NewGrantStore(dir)

	require.NoError(t, os.WriteFile(store.Path(), []byte("grants:\n  - profile: \"\"\n  - profile: local\n"), 0600))

	grants, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, []string{"local"}, grants.Names())
	assert.False(t, grants.IsGranted(""))
}

func TestGrantStoreLoadRejectsInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	store := NewGrantStore(dir)

	require.NoError(t, os.WriteFile(store.Path(), []byte("{not yaml"), 0600))

	_, err := store.Load()
	assert.Error(t, err)
}
