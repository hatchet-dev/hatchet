package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	doc := map[string]any{}
	require.NoError(t, json.Unmarshal(data, &doc))

	return doc
}

func TestNormalizeMCPInstallTargets(t *testing.T) {
	targets, err := normalizeMCPInstallTargets([]string{" Claude-Code ", "cursor", "claude-code", ""})
	require.NoError(t, err)
	assert.Equal(t, []string{"claude-code", "cursor"}, targets)

	_, err = normalizeMCPInstallTargets([]string{"zed"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown target 'zed'")
	assert.Contains(t, err.Error(), "claude-code, cursor, vscode, codex")
}

func TestMCPInstallConfigPathScopes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := mcpInstallConfigPath("claude-code", false)
	require.NoError(t, err)
	assert.Equal(t, ".mcp.json", path)

	path, err = mcpInstallConfigPath("cursor", false)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(".cursor", "mcp.json"), path)

	path, err = mcpInstallConfigPath("cursor", true)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".cursor", "mcp.json"), path)

	path, err = mcpInstallConfigPath("vscode", false)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(".vscode", "mcp.json"), path)

	// codex is user scope regardless of the flag
	for _, userScope := range []bool{false, true} {
		path, err = mcpInstallConfigPath("codex", userScope)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(home, ".codex", "config.toml"), path)
	}

	for _, target := range []string{"claude-code", "vscode"} {
		_, err = mcpInstallConfigPath(target, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project-scope")
		assert.Contains(t, err.Error(), "--user")
	}

	_, err = mcpInstallConfigPath("zed", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown target 'zed'")
}

func TestInstallMCPServerConfigFreshFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(t.TempDir())

	tests := []struct {
		target string
		topKey string
	}{
		{target: "claude-code", topKey: "mcpServers"},
		{target: "cursor", topKey: "mcpServers"},
		{target: "vscode", topKey: "servers"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			path, err := mcpInstallConfigPath(tt.target, false)
			require.NoError(t, err)

			created, err := installMCPServerConfig(tt.target, path, "hatchet")
			require.NoError(t, err)
			assert.True(t, created)

			doc := readJSONFile(t, path)
			servers, ok := doc[tt.topKey].(map[string]any)
			require.True(t, ok, "top-level key %q should be an object", tt.topKey)

			entry, ok := servers["hatchet"].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, "hatchet", entry["command"])
			assert.Equal(t, []any{"mcp", "serve"}, entry["args"])

			if tt.target == "vscode" {
				assert.Equal(t, "stdio", entry["type"])
			} else {
				assert.NotContains(t, entry, "type")
			}

			data, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.True(t, strings.HasSuffix(string(data), "}\n"), "file should end with a trailing newline")

			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
		})
	}
}

func TestWriteMCPServerJSONMergePreservesExistingContent(t *testing.T) {
	t.Chdir(t.TempDir())

	existing := `{
  "mcpServers": {
    "other": {
      "command": "other-tool",
      "args": ["run"]
    }
  },
  "unknownTopLevel": {"keep": true}
}
`
	require.NoError(t, os.WriteFile(".mcp.json", []byte(existing), 0o644))

	created, err := installMCPServerConfig("claude-code", ".mcp.json", "hatchet")
	require.NoError(t, err)
	assert.False(t, created)

	doc := readJSONFile(t, ".mcp.json")

	servers := doc["mcpServers"].(map[string]any)
	assert.Contains(t, servers, "hatchet")

	other := servers["other"].(map[string]any)
	assert.Equal(t, "other-tool", other["command"])
	assert.Equal(t, []any{"run"}, other["args"])

	unknown := doc["unknownTopLevel"].(map[string]any)
	assert.Equal(t, true, unknown["keep"])
}

func TestWriteMCPServerJSONIdempotent(t *testing.T) {
	t.Chdir(t.TempDir())

	_, err := installMCPServerConfig("claude-code", ".mcp.json", "hatchet")
	require.NoError(t, err)

	first, err := os.ReadFile(".mcp.json")
	require.NoError(t, err)

	created, err := installMCPServerConfig("claude-code", ".mcp.json", "hatchet")
	require.NoError(t, err)
	assert.False(t, created)

	second, err := os.ReadFile(".mcp.json")
	require.NoError(t, err)
	assert.Equal(t, string(first), string(second))

	// A changed command updates the entry in place.
	_, err = installMCPServerConfig("claude-code", ".mcp.json", "/opt/hatchet")
	require.NoError(t, err)

	doc := readJSONFile(t, ".mcp.json")
	entry := doc["mcpServers"].(map[string]any)["hatchet"].(map[string]any)
	assert.Equal(t, "/opt/hatchet", entry["command"])
}

func TestWriteMCPServerJSONRefusesMalformedFile(t *testing.T) {
	t.Chdir(t.TempDir())

	malformed := "{ not json"
	require.NoError(t, os.WriteFile(".mcp.json", []byte(malformed), 0o644))

	_, err := installMCPServerConfig("claude-code", ".mcp.json", "hatchet")
	require.Error(t, err)
	assert.Contains(t, err.Error(), ".mcp.json")
	assert.Contains(t, err.Error(), "invalid JSON")

	data, readErr := os.ReadFile(".mcp.json")
	require.NoError(t, readErr)
	assert.Equal(t, malformed, string(data), "the malformed file must not be overwritten")
}

func TestWriteMCPServerJSONRefusesUnexpectedTopKey(t *testing.T) {
	t.Chdir(t.TempDir())

	require.NoError(t, os.WriteFile(".mcp.json", []byte(`{"mcpServers": "nope"}`), 0o644))

	_, err := installMCPServerConfig("claude-code", ".mcp.json", "hatchet")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mcpServers")
}

func TestWriteMCPServerJSONPreservesFileMode(t *testing.T) {
	t.Chdir(t.TempDir())

	require.NoError(t, os.WriteFile(".mcp.json", []byte(`{}`), 0o600))

	_, err := installMCPServerConfig("claude-code", ".mcp.json", "hatchet")
	require.NoError(t, err)

	info, err := os.Stat(".mcp.json")
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestWriteCodexConfigFreshFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := mcpInstallConfigPath("codex", true)
	require.NoError(t, err)

	created, err := installMCPServerConfig("codex", path, "hatchet")
	require.NoError(t, err)
	assert.True(t, created)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "[mcp_servers.hatchet]")
	assert.Contains(t, content, "command = 'hatchet'")
	assert.Contains(t, content, "args = ['mcp', 'serve']")
	assert.True(t, strings.HasSuffix(content, "\n"))
}

func TestWriteCodexConfigPreservesOtherContent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".codex", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))

	existing := `# my codex settings
model = "gpt-5"

[mcp_servers.other]
command = "other-tool"
args = ["run"]

[mcp_servers.hatchet]
command = "stale-path"
args = ["mcp", "serve"]

[history]
persistence = "save-all"
`
	require.NoError(t, os.WriteFile(path, []byte(existing), 0o644))

	created, err := installMCPServerConfig("codex", path, "hatchet")
	require.NoError(t, err)
	assert.False(t, created)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "# my codex settings", "comments must be preserved")
	assert.Contains(t, content, `model = "gpt-5"`)
	assert.Contains(t, content, "[mcp_servers.other]")
	assert.Contains(t, content, `command = "other-tool"`)
	assert.Contains(t, content, "[history]")
	assert.Contains(t, content, "command = 'hatchet'")
	assert.NotContains(t, content, "stale-path")
	assert.Equal(t, 1, strings.Count(content, "[mcp_servers.hatchet]"))

	// Re-running is idempotent.
	_, err = installMCPServerConfig("codex", path, "hatchet")
	require.NoError(t, err)

	again, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(again))
}

func TestWriteCodexConfigRefusesMalformedFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".codex", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))

	malformed := "[unclosed\n"
	require.NoError(t, os.WriteFile(path, []byte(malformed), 0o644))

	_, err := installMCPServerConfig("codex", path, "hatchet")
	require.Error(t, err)
	assert.Contains(t, err.Error(), path)
	assert.Contains(t, err.Error(), "invalid TOML")

	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	assert.Equal(t, malformed, string(data), "the malformed file must not be overwritten")
}

func TestResolveMCPCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake PATH executables are not portable to windows")
	}

	binDir := t.TempDir()
	fake := filepath.Join(binDir, "hatchet")
	require.NoError(t, os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755)) // #nosec G306 -- test fixture executable

	t.Setenv("PATH", binDir)
	assert.Equal(t, "hatchet", resolveMCPCommand())

	// With no hatchet on PATH, the current executable is used.
	t.Setenv("PATH", t.TempDir())
	exe, err := os.Executable()
	require.NoError(t, err)
	assert.Equal(t, exe, resolveMCPCommand())
}

func TestMCPInstallPrintWritesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cwd := t.TempDir()
	t.Chdir(cwd)

	out, err := mcpInstallPrint(mcpInstallTargetNames(), false, "hatchet")
	require.NoError(t, err)

	assert.Contains(t, out, "# claude-code: .mcp.json")
	assert.Contains(t, out, `"mcpServers"`)
	assert.Contains(t, out, `"servers"`)
	assert.Contains(t, out, `"type": "stdio"`)
	assert.Contains(t, out, "[mcp_servers.hatchet]")

	for _, dir := range []string{cwd, home} {
		entries, readErr := os.ReadDir(dir)
		require.NoError(t, readErr)
		assert.Empty(t, entries, "print mode must not create files in %s", dir)
	}
}

func TestInstallMCPServerConfigRefusesSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on windows")
	}

	outside := t.TempDir()
	t.Chdir(t.TempDir())

	// A checkout can commit the config file itself as a symlink.
	victim := filepath.Join(outside, "victim.json")
	require.NoError(t, os.WriteFile(victim, []byte(`{"mcpServers":{}}`), 0o644))
	require.NoError(t, os.Symlink(victim, ".mcp.json"))

	_, err := installMCPServerConfig("claude-code", ".mcp.json", "hatchet")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symbolic link")

	data, readErr := os.ReadFile(victim)
	require.NoError(t, readErr)
	assert.Equal(t, `{"mcpServers":{}}`, string(data), "the symlink destination must not be modified")

	// Or a parent directory as a symlink.
	outsideDir := filepath.Join(outside, "cursor-elsewhere")
	require.NoError(t, os.MkdirAll(outsideDir, 0o755))
	require.NoError(t, os.Symlink(outsideDir, ".cursor"))

	_, err = installMCPServerConfig("cursor", filepath.Join(".cursor", "mcp.json"), "hatchet")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symbolic link")

	entries, readDirErr := os.ReadDir(outsideDir)
	require.NoError(t, readDirErr)
	assert.Empty(t, entries, "nothing may be written through the symlinked directory")
}

func TestWriteCodexConfigMatchesHeaderVariants(t *testing.T) {
	// TOML allows several spellings of the same table header; each must be
	// recognized and replaced rather than duplicated. The escaped spelling
	// pins part of F03: decoded keys, not source text, name the table.
	variants := []string{
		`[mcp_servers."hatchet"]`,
		`[mcp_servers.hatchet] # managed by hatchet`,
		`[ mcp_servers . hatchet ]`,
		`[mcp_servers."hat\u0063het"]`,
	}

	for _, header := range variants {
		t.Run(header, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			path := filepath.Join(home, ".codex", "config.toml")
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))

			existing := "model = \"gpt-5\"\n\n" + header + "\ncommand = \"stale-path\"\nargs = [\"mcp\", \"serve\"]\n\n[history]\npersistence = \"save-all\"\n"
			require.NoError(t, os.WriteFile(path, []byte(existing), 0o644))

			_, err := installMCPServerConfig("codex", path, "hatchet")
			require.NoError(t, err)

			data, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			content := string(data)

			assert.NotContains(t, content, "stale-path")
			assert.Contains(t, content, "command = 'hatchet'")
			assert.Contains(t, content, `model = "gpt-5"`)
			assert.Contains(t, content, "[history]")
			assert.Equal(t, 1, strings.Count(content, "mcp_servers"), "the variant header must be replaced, not duplicated")
		})
	}
}

func writeCodexFixture(t *testing.T, content string) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".codex", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	return path
}

func readTOMLFile(t *testing.T, path string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	doc := map[string]any{}
	require.NoError(t, toml.Unmarshal(data, &doc))

	return doc
}

// F01: quoted keys with significant whitespace name different tables; they
// must survive an install untouched instead of being treated as the entry.
func TestWriteCodexConfigPreservesQuotedKeyWhitespace(t *testing.T) {
	for _, header := range []string{`[mcp_servers." hatchet "]`, `[" mcp_servers ".hatchet]`} {
		t.Run(header, func(t *testing.T) {
			path := writeCodexFixture(t, header+"\ncommand = 'other-tool'\n")

			_, err := installMCPServerConfig("codex", path, "hatchet")
			require.NoError(t, err)

			data, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			content := string(data)

			assert.Contains(t, content, "other-tool", "the whitespace-keyed table must be preserved")
			assert.Contains(t, content, "[mcp_servers.hatchet]")
			assert.Contains(t, content, "command = 'hatchet'")
		})
	}
}

// F02: a multi-line string containing header-looking lines must never be
// rewritten in place; either the real table installs around it or the file
// is refused unchanged.
func TestWriteCodexConfigMultilineFalseHeader(t *testing.T) {
	for _, quote := range []string{`"""`, `'''`} {
		t.Run(quote, func(t *testing.T) {
			original := "developer_instructions = " + quote + "\nConfiguration example:\n[mcp_servers.hatchet]\ncommand = 'example'\n[example_boundary]\nKeep this text.\n" + quote + "\n"
			path := writeCodexFixture(t, original)

			var before map[string]any
			require.NoError(t, toml.Unmarshal([]byte(original), &before))

			_, err := installMCPServerConfig("codex", path, "hatchet")
			data, readErr := os.ReadFile(path)
			require.NoError(t, readErr)

			if err != nil {
				assert.Equal(t, original, string(data), "a refusal must leave the file unchanged")
				return
			}

			after := readTOMLFile(t, path)
			assert.Equal(t, before["developer_instructions"], after["developer_instructions"], "the unrelated string must be untouched")

			entry := after["mcp_servers"].(map[string]any)["hatchet"].(map[string]any)
			assert.Equal(t, "hatchet", entry["command"])
			assert.Equal(t, []any{"mcp", "serve"}, entry["args"])
		})
	}
}

// F03: the dotted-assignment form of the entry is replaced, not refused and
// not duplicated.
func TestWriteCodexConfigReplacesDottedAssignment(t *testing.T) {
	path := writeCodexFixture(t, "mcp_servers.hatchet.command = 'stale-path'\nother = 1\n")

	_, err := installMCPServerConfig("codex", path, "hatchet")
	require.NoError(t, err)

	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	content := string(data)

	assert.NotContains(t, content, "stale-path")
	assert.Contains(t, content, "other = 1")
	assert.Equal(t, 1, strings.Count(content, "mcp_servers"))

	entry := readTOMLFile(t, path)["mcp_servers"].(map[string]any)["hatchet"].(map[string]any)
	assert.Equal(t, "hatchet", entry["command"])
	assert.Equal(t, []any{"mcp", "serve"}, entry["args"])
}

// F09: hatchet subtables separated from the parent by unrelated tables must
// be collected and replaced too, not left behind.
func TestWriteCodexConfigRemovesSeparatedSubtable(t *testing.T) {
	path := writeCodexFixture(t, "[mcp_servers.hatchet]\ncommand = 'old'\n[mcp_servers.other]\ncommand = 'other'\n[mcp_servers.hatchet.env]\nKEEP = 'old-value'\n")

	_, err := installMCPServerConfig("codex", path, "hatchet")
	require.NoError(t, err)

	doc := readTOMLFile(t, path)
	servers := doc["mcp_servers"].(map[string]any)

	entry := servers["hatchet"].(map[string]any)
	assert.Equal(t, map[string]any{"command": "hatchet", "args": []any{"mcp", "serve"}}, entry, "the stale env subtable must be gone")

	other := servers["other"].(map[string]any)
	assert.Equal(t, "other", other["command"])
}

// F03: inline-table and array-of-tables forms cannot be spliced; they are
// refused explicitly with the file left unchanged.
func TestWriteCodexConfigRefusesInlineAndArrayForms(t *testing.T) {
	cases := map[string]struct {
		content string
		wantErr string
	}{
		"inline_parent":   {"mcp_servers = { hatchet = { command = 'old' }, other = { command = 'keep' } }\n", "inline table"},
		"inline_child":    {"[mcp_servers]\nhatchet = { command = 'old' }\n", "inline table"},
		"inline_dotted":   {"mcp_servers.hatchet = { command = 'old' }\n", "inline table"},
		"array_of_tables": {"[[mcp_servers.hatchet]]\ncommand = 'old'\n", "array of tables"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeCodexFixture(t, tc.content)

			_, err := installMCPServerConfig("codex", path, "hatchet")
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
			assert.Contains(t, err.Error(), "manually")

			data, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			assert.Equal(t, tc.content, string(data), "a refused file must not be modified")
		})
	}
}

// F04: unrelated numeric values must survive the JSON round-trip exactly.
func TestWriteMCPServerJSONPreservesNumberPrecision(t *testing.T) {
	t.Chdir(t.TempDir())

	original := `{"large":9007199254740993,"decimal":0.1234567890123456789,"mcpServers":{"other":{"value":9223372036854775807}}}`
	require.NoError(t, os.WriteFile(".mcp.json", []byte(original), 0o644))

	_, err := installMCPServerConfig("claude-code", ".mcp.json", "hatchet")
	require.NoError(t, err)

	data, readErr := os.ReadFile(".mcp.json")
	require.NoError(t, readErr)
	content := string(data)

	assert.Contains(t, content, "9007199254740993")
	assert.Contains(t, content, "0.1234567890123456789")
	assert.Contains(t, content, "9223372036854775807")
}

// F05: a null JSON root (and any other non-object root) is refused without a
// panic and without touching the file.
func TestWriteMCPServerJSONRefusesNonObjectRoots(t *testing.T) {
	cases := map[string]string{
		"null":          "null\n",
		"array":         "[]\n",
		"string":        `"text"`,
		"two_documents": "{} {}",
	}

	for name, original := range cases {
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			require.NoError(t, os.WriteFile(".mcp.json", []byte(original), 0o644))

			_, err := installMCPServerConfig("claude-code", ".mcp.json", "hatchet")
			require.Error(t, err)
			assert.Contains(t, err.Error(), ".mcp.json")

			data, readErr := os.ReadFile(".mcp.json")
			require.NoError(t, readErr)
			assert.Equal(t, original, string(data), "a refused file must not be modified")
		})
	}
}

// F07: a refusal on a later target must surface during the prepare phase,
// before the earlier target's file is written.
func TestPrepareMCPInstallsRefusesBeforeAnyWrite(t *testing.T) {
	t.Chdir(t.TempDir())

	require.NoError(t, os.MkdirAll(".cursor", 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(".cursor", "mcp.json"), []byte("not json"), 0o644))

	paths := map[string]string{
		"claude-code": ".mcp.json",
		"cursor":      filepath.Join(".cursor", "mcp.json"),
	}

	_, err := prepareMCPInstalls([]string{"claude-code", "cursor"}, paths, "hatchet")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")

	_, statErr := os.Stat(".mcp.json")
	assert.True(t, os.IsNotExist(statErr), "the first target must not be written when a later target is refused")
}

// F11: an edit landing between prepare and commit is detected and refused
// instead of being silently overwritten.
func TestMCPInstallCommitRefusesConcurrentEdit(t *testing.T) {
	t.Chdir(t.TempDir())

	require.NoError(t, os.WriteFile(".mcp.json", []byte(`{"mcpServers":{}}`), 0o644))

	p, err := prepareMCPInstall("claude-code", ".mcp.json", "hatchet")
	require.NoError(t, err)
	defer p.close()

	concurrent := `{"concurrent":true}`
	require.NoError(t, os.WriteFile(".mcp.json", []byte(concurrent), 0o644))

	err = p.commit()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "changed while installing")

	data, readErr := os.ReadFile(".mcp.json")
	require.NoError(t, readErr)
	assert.Equal(t, concurrent, string(data), "the concurrent edit must not be overwritten")
}

// F10: a hatchet found on PATH inside the current directory (or via a
// relative PATH entry) is never trusted as the configured command.
func TestResolveMCPCommandRefusesCwdPATHHit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake PATH executables are not portable to windows")
	}

	t.Chdir(t.TempDir())
	cwd, err := os.Getwd()
	require.NoError(t, err)

	binDir := filepath.Join(cwd, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "hatchet"), []byte("#!/bin/sh\n"), 0o755)) // #nosec G306 -- test fixture executable

	exe, err := os.Executable()
	require.NoError(t, err)

	// An absolute PATH entry inside the checkout must fall back.
	t.Setenv("PATH", binDir)
	assert.Equal(t, exe, resolveMCPCommand())

	// A relative PATH entry resolves with exec.ErrDot and must fall back too.
	t.Setenv("PATH", "bin")
	assert.Equal(t, exe, resolveMCPCommand())
}

func TestWriteCodexConfigPreservesCommentBeforeNextTable(t *testing.T) {
	// Greptile P2 on the PR: a comment block directly above the table that
	// follows the hatchet entry documents that table and must survive the
	// replacement. A comment inside the hatchet span separated from the next
	// table by a blank line still goes with the removed entry.
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".codex", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))

	existing := `[mcp_servers.hatchet]
command = "stale-path"
# stale note about the hatchet entry

# keep me: documents the history table
[history]
persistence = "save-all"
`
	require.NoError(t, os.WriteFile(path, []byte(existing), 0o644))

	_, err := installMCPServerConfig("codex", path, "hatchet")
	require.NoError(t, err)

	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	content := string(data)

	assert.Contains(t, content, "# keep me: documents the history table")
	assert.Contains(t, content, "[history]")
	assert.NotContains(t, content, "stale-path")
	assert.NotContains(t, content, "# stale note")
	assert.Contains(t, content, "command = 'hatchet'")
	assert.Equal(t, 1, strings.Count(content, "mcp_servers"))

	// The preserved comment must still sit directly above its table.
	assert.Contains(t, content, "# keep me: documents the history table\n[history]")
}
