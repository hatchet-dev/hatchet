package profilestore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	cliconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
)

func TestKeyringTokens_EndToEnd(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".hatchet")
	require.NoError(t, os.MkdirAll(dir, 0700))

	legacy, err := NewStore(dir, "profiles.yaml")
	require.NoError(t, err)
	require.NoError(t, legacy.AddProfile("legacy", makeTestProfile("legacy", "legacy-token")))
	require.Equal(t, []string{"legacy"}, legacy.PlaintextProfiles())

	secrets := MemorySecrets{}
	s, err := NewStore(dir, "profiles.yaml")
	require.NoError(t, err)
	s.secrets = secrets

	require.NoError(t, s.AddProfile("new", makeTestProfile("new", "new-token")))
	raw, err := os.ReadFile(s.Path())
	require.NoError(t, err)
	require.NotContains(t, string(raw), "new-token")
	require.Contains(t, string(raw), "legacy-token")

	p, err := s.GetProfile("new")
	require.NoError(t, err)
	require.Equal(t, "new-token", p.Token)
	require.Equal(t, "new-token", s.GetProfiles()["new"].Token)

	moved, err := s.MigrateTokens()
	require.NoError(t, err)
	require.Equal(t, []string{"legacy"}, moved)
	require.Empty(t, s.PlaintextProfiles())
	raw, _ = os.ReadFile(s.Path())
	require.NotContains(t, string(raw), "legacy-token")
	p, err = s.GetProfile("legacy")
	require.NoError(t, err)
	require.Equal(t, "legacy-token", p.Token)

	require.NoError(t, s.UpdateProfile("legacy", &cliconfig.Profile{Token: "rotated"}))
	p, _ = s.GetProfile("legacy")
	require.Equal(t, "rotated", p.Token)
	raw, _ = os.ReadFile(s.Path())
	require.NotContains(t, string(raw), "rotated")

	removed, err := s.RemoveProfileIfTokenMatches("new", "wrong")
	require.NoError(t, err)
	require.False(t, removed)
	removed, err = s.RemoveProfileIfTokenMatches("new", "new-token")
	require.NoError(t, err)
	require.True(t, removed)
	_, err = secrets.Get(s.secretKey("new"))
	require.Error(t, err)

	moved, err = s.MigrateTokens()
	require.NoError(t, err)
	require.Empty(t, moved)

	require.NoError(t, s.RemoveProfile("legacy"))
	require.Empty(t, s.ListProfiles())
	_, err = secrets.Get(s.secretKey("legacy"))
	require.Error(t, err)

	fileOnly, err := NewStore(dir, "profiles.yaml")
	require.NoError(t, err)
	require.NoError(t, fileOnly.AddProfile("headless", makeTestProfile("headless", "headless-token")))
	moved, err = fileOnly.MigrateTokens()
	require.NoError(t, err)
	require.Empty(t, moved)
	require.Equal(t, []string{"headless"}, fileOnly.PlaintextProfiles())
}
