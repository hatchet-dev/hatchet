package profilestore

// Tests for the yaml.Node writer's interop guarantees: unquoted timestamps
// (the CLI cannot decode quoted timestamps into its time.Time fields), and
// preservation of comments, key ordering, and content the store does not
// know about (the compatibility promise external writers such as the
// embedded engine's profile registration rely on).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func readProfilesFile(t *testing.T, hatchetDir string) (string, map[string]any) {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(hatchetDir, "profiles.yaml"))
	require.NoError(t, err, "could not read profiles file")

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(data, &parsed), "written profiles file does not parse")

	return string(data), parsed
}

func writeProfilesFile(t *testing.T, hatchetDir, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(hatchetDir, "profiles.yaml"), []byte(content), 0600))
}

func profileEntry(t *testing.T, parsed map[string]any, name string) map[string]any {
	t.Helper()

	profiles, ok := parsed["profiles"].(map[string]any)
	require.True(t, ok, "profiles key missing or not a mapping: %#v", parsed["profiles"])

	entry, ok := profiles[name].(map[string]any)
	require.True(t, ok, "profile %q missing or not a mapping: %#v", name, profiles[name])

	return entry
}

// TestAddProfileWritesUnquotedTimestamp is the regression test for the
// quoted-timestamp trap: the CLI loads the profiles file into a struct with
// time.Time fields via viper, and a quoted expiresAt fails that unmarshal,
// which makes every CLI command fatal on startup.
func TestAddProfileWritesUnquotedTimestamp(t *testing.T) {
	s, hatchetDir := setupTestStore(t)

	require.NoError(t, s.AddProfile("prod", makeTestProfile("prod", "token-1")))

	raw, parsed := readProfilesFile(t, hatchetDir)

	assert.NotContains(t, raw, `expiresat: "`, "expiresat was written double-quoted:\n%s", raw)
	assert.NotContains(t, raw, "expiresat: '", "expiresat was written single-quoted:\n%s", raw)

	entry := profileEntry(t, parsed, "prod")
	_, isTime := entry["expiresat"].(time.Time)
	assert.True(t, isTime, "expiresat did not parse as a yaml timestamp: %#v", entry["expiresat"])

	// The load path used by every CLI command (NewStore unmarshals into
	// cli.ProfileFile the same way the CLI init does) must accept the file.
	reloaded, err := NewStore(hatchetDir, "profiles.yaml")
	require.NoError(t, err)

	profile, err := reloaded.GetProfile("prod")
	require.NoError(t, err)
	assert.False(t, profile.ExpiresAt.IsZero())
}

// TestQuotedTimestampFailsLoad documents why the unquoted form is
// load-bearing: a quoted expiresAt cannot be decoded into the CLI's time.Time
// field, and the CLI's load path (mirrored by NewStore) rejects the file.
func TestQuotedTimestampFailsLoad(t *testing.T) {
	_, hatchetDir := setupTestStore(t)

	writeProfilesFile(t, hatchetDir, `profiles:
    prod:
        apiserverurl: https://prod.example.com
        expiresat: "2027-01-02T03:04:05Z"
        grpchostport: prod.example.com:443
        name: prod
        tenantid: 11111111-2222-3333-4444-555555555555
        tlsstrategy: tls
        token: prod-token
`)

	_, err := NewStore(hatchetDir, "profiles.yaml")
	require.Error(t, err, "a quoted timestamp must be rejected by the CLI load path")
}

// TestAddProfilePreservesUnknownContent is the compatibility promise external
// writers rely on: comments, key ordering, unknown top-level keys, and extra
// metadata fields under a profile entry all survive a CLI AddProfile.
func TestAddProfilePreservesUnknownContent(t *testing.T) {
	s, hatchetDir := setupTestStore(t)

	writeProfilesFile(t, hatchetDir, `# keep this comment
defaultprofile: prod
unknowntoplevel: keep-me
profiles:
    embedded:
        tenantid: 707d0855-80ab-4e1f-a156-f1c4546cbf52
        name: embedded
        token: embedded-token
        expiresat: 2027-01-02T03:04:05Z
        apiserverurl: http://localhost:28243
        grpchostport: 127.0.0.1:50051
        tlsstrategy: none
        embedded: true
        pid: 4242
        cwd: /somewhere
        startedat: 2026-01-02T03:04:05Z
    prod:
        apiserverurl: https://prod.example.com
        expiresat: 2027-01-02T03:04:05Z
        grpchostport: prod.example.com:443
        name: prod
        tenantid: 11111111-2222-3333-4444-555555555555
        tlsstrategy: tls
        token: prod-token
`)

	// reopen so the store sees the seeded file, then add a new profile
	s, err := NewStore(hatchetDir, "profiles.yaml")
	require.NoError(t, err)
	require.NoError(t, s.AddProfile("staging", makeTestProfile("staging", "staging-token")))

	raw, parsed := readProfilesFile(t, hatchetDir)

	// comment and unknown top-level key survive
	assert.Contains(t, raw, "# keep this comment")
	assert.Equal(t, "keep-me", parsed["unknowntoplevel"])
	assert.Equal(t, "prod", parsed["defaultprofile"])

	// ordering of existing entries is preserved (embedded before prod, and
	// embedded's hand-written field order untouched)
	assert.Less(t, strings.Index(raw, "embedded:"), strings.Index(raw, "prod:"))
	assert.Less(t, strings.Index(raw, "tenantid: 707d0855"), strings.Index(raw, "name: embedded"))

	// the embedded entry's extra metadata fields survive
	embedded := profileEntry(t, parsed, "embedded")
	assert.Equal(t, true, embedded["embedded"])
	assert.Equal(t, 4242, embedded["pid"])
	assert.Equal(t, "/somewhere", embedded["cwd"])
	_, isTime := embedded["startedat"].(time.Time)
	assert.True(t, isTime, "startedat did not stay a yaml timestamp: %#v", embedded["startedat"])

	// the untouched prod entry is intact
	prod := profileEntry(t, parsed, "prod")
	assert.Equal(t, "prod-token", prod["token"])

	// the new profile exists and the file is still fully CLI-readable
	reloaded, err := NewStore(hatchetDir, "profiles.yaml")
	require.NoError(t, err)
	assert.Equal(t, []string{"embedded", "prod", "staging"}, reloaded.ListProfiles())

	stagingProfile, err := reloaded.GetProfile("staging")
	require.NoError(t, err)
	assert.Equal(t, "staging-token", stagingProfile.Token)
}

// TestAddProfileKeepsExtraFieldsOnSameEntry: overwriting a profile via
// AddProfile only replaces the fields the CLI knows about; extra metadata
// under the same entry is kept (matching the old viper writer, which carried
// unknown nested keys through every write).
func TestAddProfileKeepsExtraFieldsOnSameEntry(t *testing.T) {
	s, hatchetDir := setupTestStore(t)

	require.NoError(t, s.UpsertProfile("embedded", makeTestProfile("embedded", "token-old"), map[string]any{
		"embedded":  true,
		"pid":       4242,
		"startedat": time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}))

	require.NoError(t, s.AddProfile("embedded", makeTestProfile("embedded", "token-new")))

	_, parsed := readProfilesFile(t, hatchetDir)
	entry := profileEntry(t, parsed, "embedded")
	assert.Equal(t, "token-new", entry["token"])
	assert.Equal(t, true, entry["embedded"])
	assert.Equal(t, 4242, entry["pid"])
}

func TestUpsertProfileWithExtraFields(t *testing.T) {
	s, hatchetDir := setupTestStore(t)

	startedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	profile := makeTestProfile("embedded", "embedded-token")
	profile.TLSStrategy = "none"

	require.NoError(t, s.UpsertProfile("embedded", profile, map[string]any{
		"embedded":  true,
		"PID":       4242,
		"cwd":       "/somewhere",
		"startedat": startedAt,
	}))

	raw, parsed := readProfilesFile(t, hatchetDir)
	entry := profileEntry(t, parsed, "embedded")

	// extra fields are written lowercased alongside the known fields
	assert.Equal(t, true, entry["embedded"])
	assert.Equal(t, 4242, entry["pid"])
	assert.Equal(t, "/somewhere", entry["cwd"])

	// time-valued extra fields get the same unquoted-timestamp treatment
	assert.NotContains(t, raw, `startedat: "`)
	assert.NotContains(t, raw, "startedat: '")
	parsedStartedAt, isTime := entry["startedat"].(time.Time)
	require.True(t, isTime, "startedat did not parse as a yaml timestamp: %#v", entry["startedat"])
	assert.True(t, startedAt.Equal(parsedStartedAt))

	// the registration is a normal profile from the CLI's point of view
	got, err := s.GetProfile("embedded")
	require.NoError(t, err)
	assert.Equal(t, "embedded-token", got.Token)
	assert.Equal(t, "none", got.TLSStrategy)
}

func TestRemoveProfileIfTokenMatches(t *testing.T) {
	s, hatchetDir := setupTestStore(t)

	require.NoError(t, s.AddProfile("embedded", makeTestProfile("embedded", "token-mine")))
	require.NoError(t, s.AddProfile("prod", makeTestProfile("prod", "prod-token")))
	require.NoError(t, s.SetDefaultProfile("embedded"))

	// non-matching token: entry is kept (a newer instance's registration must
	// not be deleted by an older instance's shutdown)
	removed, err := s.RemoveProfileIfTokenMatches("embedded", "token-other")
	require.NoError(t, err)
	assert.False(t, removed)

	profile, err := s.GetProfile("embedded")
	require.NoError(t, err)
	assert.Equal(t, "token-mine", profile.Token)

	// matching token: entry is removed, default-profile setting is untouched
	removed, err = s.RemoveProfileIfTokenMatches("embedded", "token-mine")
	require.NoError(t, err)
	assert.True(t, removed)

	_, err = s.GetProfile("embedded")
	assert.Error(t, err)

	_, parsed := readProfilesFile(t, hatchetDir)
	assert.Equal(t, "embedded", parsed["defaultprofile"], "token-matched removal must not touch the default profile")

	// missing profile: no-op, no error
	removed, err = s.RemoveProfileIfTokenMatches("embedded", "token-mine")
	require.NoError(t, err)
	assert.False(t, removed)
}

func TestRemoveProfileIfTokenMatches_MissingFile(t *testing.T) {
	hatchetDir := filepath.Join(t.TempDir(), ".hatchet")

	s, err := NewStore(hatchetDir, "profiles.yaml")
	require.NoError(t, err)

	removed, err := s.RemoveProfileIfTokenMatches("embedded", "token-any")
	require.NoError(t, err)
	assert.False(t, removed)

	_, err = os.Stat(s.Path())
	assert.True(t, os.IsNotExist(err), "a no-op removal must not create the profiles file")
}

// TestReAddAfterRemovingLastProfileStaysBlockStyle guards against the flow
// style trap: removing the last profile leaves "profiles: {}" behind, which
// parses as a flow-style mapping, and a profile merged into a flow mapping
// would be emitted with quoted timestamps.
func TestReAddAfterRemovingLastProfileStaysBlockStyle(t *testing.T) {
	s, hatchetDir := setupTestStore(t)

	require.NoError(t, s.AddProfile("only", makeTestProfile("only", "token-1")))
	require.NoError(t, s.RemoveProfile("only"))
	require.NoError(t, s.AddProfile("again", makeTestProfile("again", "token-2")))

	raw, _ := readProfilesFile(t, hatchetDir)
	assert.NotContains(t, raw, "{", "profiles file was written in flow style:\n%s", raw)

	reloaded, err := NewStore(hatchetDir, "profiles.yaml")
	require.NoError(t, err)

	profile, err := reloaded.GetProfile("again")
	require.NoError(t, err)
	assert.Equal(t, "token-2", profile.Token)
	assert.False(t, profile.ExpiresAt.IsZero())
}

// TestRemoveProfilePreservesUnknownContent: the old viper-based remove
// rewrote the file from only the keys it knew about; the yaml.Node writer
// must keep everything else.
func TestRemoveProfilePreservesUnknownContent(t *testing.T) {
	_, hatchetDir := setupTestStore(t)

	writeProfilesFile(t, hatchetDir, `# a comment to keep
unknowntoplevel: keep-me
profiles:
    one:
        tenantid: tenant-123
        name: one
        token: token-1
        expiresat: 2027-01-02T03:04:05Z
        apiserverurl: http://localhost:8080
        grpchostport: localhost:7077
        tlsstrategy: tls
    two:
        tenantid: tenant-123
        name: two
        token: token-2
        expiresat: 2027-01-02T03:04:05Z
        apiserverurl: http://localhost:8080
        grpchostport: localhost:7077
        tlsstrategy: tls
`)

	s, err := NewStore(hatchetDir, "profiles.yaml")
	require.NoError(t, err)
	require.NoError(t, s.RemoveProfile("one"))

	raw, parsed := readProfilesFile(t, hatchetDir)
	assert.Contains(t, raw, "# a comment to keep")
	assert.Equal(t, "keep-me", parsed["unknowntoplevel"])

	two := profileEntry(t, parsed, "two")
	assert.Equal(t, "token-2", two["token"])
}

// TestFilePermissions: the profiles file contains tokens and must be written
// 0600.
func TestFilePermissions(t *testing.T) {
	s, hatchetDir := setupTestStore(t)

	require.NoError(t, s.AddProfile("prod", makeTestProfile("prod", "token-1")))

	info, err := os.Stat(filepath.Join(hatchetDir, "profiles.yaml"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

// TestNewDefaultStoreResolvesFileName: the store resolves its location the
// way the CLI does: HATCHET_CLI_PROFILE_FILE_NAME beats the profileFileName
// key in ~/.hatchet/config.yaml, which beats the profiles.yaml default.
func TestNewDefaultStoreResolvesFileName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("HATCHET_CLI_PROFILE_FILE_NAME", "")

	hatchetDir := filepath.Join(home, ".hatchet")
	require.NoError(t, os.MkdirAll(hatchetDir, 0700))

	// default
	s, err := NewDefaultStore()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(hatchetDir, "profiles.yaml"), s.Path())

	// config.yaml override
	require.NoError(t, os.WriteFile(filepath.Join(hatchetDir, "config.yaml"), []byte("profileFileName: from-config.yaml\n"), 0600))

	s, err = NewDefaultStore()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(hatchetDir, "from-config.yaml"), s.Path())

	// env var beats config.yaml
	t.Setenv("HATCHET_CLI_PROFILE_FILE_NAME", "from-env.yaml")

	s, err = NewDefaultStore()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(hatchetDir, "from-env.yaml"), s.Path())
}
