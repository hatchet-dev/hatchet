package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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
