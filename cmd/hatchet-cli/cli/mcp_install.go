package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"

	configcli "github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

// mcpInstallCmd represents the mcp install subcommand
var mcpInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Add the Hatchet MCP server to AI coding agents",
	Long: `Write the local Hatchet MCP server ('hatchet mcp serve' over stdio) into the
MCP configuration files of AI coding agents, so setup is one command instead
of hand-editing JSON.

Supported targets:
  claude-code   project .mcp.json
  cursor        project .cursor/mcp.json, or user ~/.cursor/mcp.json with --user
  vscode        project .vscode/mcp.json
  codex         user ~/.codex/config.toml

Existing configuration is preserved: only the 'hatchet' server entry is added
or updated. For agents not listed above, use --print to get the config
snippets and add them by hand.`,
	Example: `  # Interactive multi-select (detected agents are pre-checked)
  hatchet mcp install

  # Non-interactive
  hatchet mcp install --target claude-code,cursor

  # User-scope Cursor config
  hatchet mcp install --target cursor --user

  # Also grant profiles, exactly like 'hatchet mcp auth --grant'
  hatchet mcp install --target claude-code --grant local

  # Print the config snippets without writing anything
  hatchet mcp install --print`,
	Run: func(cmd *cobra.Command, args []string) {
		runMCPInstall(cmd)
	},
}

// mcpServerEntryName is the server name written into agent MCP configs.
const mcpServerEntryName = "hatchet"

// mcpServeArgs is the argv tail configured for every agent.
var mcpServeArgs = []string{"mcp", "serve"}

// mcpInstallTargetList defines the supported agents in display order.
var mcpInstallTargetList = []struct {
	name  string
	label string
}{
	{"claude-code", "Claude Code (project .mcp.json)"},
	{"cursor", "Cursor (project .cursor/mcp.json)"},
	{"vscode", "VS Code (project .vscode/mcp.json)"},
	{"codex", "Codex (user ~/.codex/config.toml)"},
}

func mcpInstallTargetNames() []string {
	names := make([]string, 0, len(mcpInstallTargetList))
	for _, target := range mcpInstallTargetList {
		names = append(names, target.name)
	}

	return names
}

func runMCPInstall(cmd *cobra.Command) {
	targetFlags, _ := cmd.Flags().GetStringSlice("target")
	userScope, _ := cmd.Flags().GetBool("user")
	grantFlags, _ := cmd.Flags().GetStringSlice("grant")
	printOnly, _ := cmd.Flags().GetBool("print")

	command := resolveMCPCommand()

	targets, err := normalizeMCPInstallTargets(targetFlags)
	if err != nil {
		configcli.Logger.Fatalf("%v", err)
	}

	if printOnly {
		if len(targets) == 0 {
			targets = mcpInstallTargetNames()
		}

		out, printErr := mcpInstallPrint(targets, userScope, command)
		if printErr != nil {
			configcli.Logger.Fatalf("%v", printErr)
		}

		fmt.Print(out)

		return
	}

	interactive := len(targets) == 0
	if interactive {
		targets = selectMCPInstallTargets()
		if len(targets) == 0 {
			fmt.Println(styles.InfoMessage("Nothing selected."))
			return
		}
	}

	// Resolve every path before writing anything, so a scope error cannot
	// leave a partial install behind.
	paths := make(map[string]string, len(targets))
	for _, target := range targets {
		path, pathErr := mcpInstallConfigPath(target, userScope)
		if pathErr != nil {
			configcli.Logger.Fatalf("%v", pathErr)
		}
		paths[target] = path
	}

	for _, target := range targets {
		created, installErr := installMCPServerConfig(target, paths[target], command)
		if installErr != nil {
			configcli.Logger.Fatalf("%v", installErr)
		}

		verb := "Updated"
		if created {
			verb = "Created"
		}
		fmt.Println(styles.SuccessMessage(fmt.Sprintf("%s %s (%s)", verb, paths[target], target)))
	}

	store := mcpGrantStore()
	grants, err := store.Load()
	if err != nil {
		configcli.Logger.Fatalf("could not load MCP grants: %v", err)
	}

	switch {
	case len(grantFlags) > 0:
		fmt.Println()
		runMCPAuthFlags(store, grants, grantFlags, nil)
	case interactive && len(grants.Names()) == 0:
		fmt.Println()
		var doGrant bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("No profiles are granted for MCP use. Grant some now?").
					Description("A running embedded Hatchet instance is usable without a grant.").
					Value(&doGrant),
			),
		).WithTheme(styles.HatchetTheme())
		if formErr := form.Run(); formErr == nil && doGrant {
			runMCPAuthInteractive(store, grants)
		}
	}

	if len(grants.Names()) == 0 {
		fmt.Println()
		fmt.Println(styles.InfoMessage("No profiles are granted for MCP use yet. Run 'hatchet mcp auth' to grant some. A running embedded Hatchet instance is usable without a grant."))
	}
}

// normalizeMCPInstallTargets validates and dedupes the --target values.
func normalizeMCPInstallTargets(targetFlags []string) ([]string, error) {
	known := make(map[string]bool, len(mcpInstallTargetList))
	for _, target := range mcpInstallTargetList {
		known[target.name] = true
	}

	seen := make(map[string]bool, len(targetFlags))
	targets := make([]string, 0, len(targetFlags))

	for _, name := range targetFlags {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || seen[name] {
			continue
		}
		if !known[name] {
			return nil, fmt.Errorf("unknown target '%s'; supported targets: %s", name, strings.Join(mcpInstallTargetNames(), ", "))
		}
		seen[name] = true
		targets = append(targets, name)
	}

	return targets, nil
}

// selectMCPInstallTargets opens a multi-select of targets, pre-checking the
// ones whose config directory or file is detected.
func selectMCPInstallTargets() []string {
	detected := detectMCPInstallTargets()

	options := make([]huh.Option[string], 0, len(mcpInstallTargetList))
	selected := make([]string, 0, len(mcpInstallTargetList))
	for _, target := range mcpInstallTargetList {
		options = append(options, huh.NewOption(target.label, target.name))
		if detected[target.name] {
			selected = append(selected, target.name)
		}
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select the agents to configure:").
				Description("Detected agents are pre-checked.").
				Options(options...).
				Value(&selected),
		),
	).WithTheme(styles.HatchetTheme())

	if err := form.Run(); err != nil {
		configcli.Logger.Fatalf("could not run the target selection form: %v (use --target for non-interactive use)", err)
	}

	return selected
}

// detectMCPInstallTargets reports which targets look present, based on the
// current directory (or the home directory for codex).
func detectMCPInstallTargets() map[string]bool {
	detected := map[string]bool{}

	if pathExists("CLAUDE.md") || pathExists(".claude") || pathExists(".mcp.json") {
		detected["claude-code"] = true
	}
	if pathExists(".cursor") {
		detected["cursor"] = true
	}
	if pathExists(".vscode") {
		detected["vscode"] = true
	}
	if home, err := os.UserHomeDir(); err == nil && pathExists(filepath.Join(home, ".codex")) {
		detected["codex"] = true
	}

	return detected
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// resolveMCPCommand picks the command agents launch. A 'hatchet' found on
// PATH keeps the config portable so it can be committed; otherwise the
// absolute path of the current binary is used.
func resolveMCPCommand() string {
	if _, err := exec.LookPath("hatchet"); err == nil {
		return "hatchet"
	}

	if exe, err := os.Executable(); err == nil {
		return exe
	}

	return "hatchet"
}

// mcpInstallConfigPath resolves the config file a target uses at the given
// scope. Project-scope paths are relative to the current directory.
func mcpInstallConfigPath(target string, userScope bool) (string, error) {
	switch target {
	case "claude-code":
		if userScope {
			return "", fmt.Errorf("target claude-code only writes project-scope configuration; rerun without --user")
		}
		return ".mcp.json", nil
	case "cursor":
		if userScope {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("could not resolve the home directory: %w", err)
			}
			return filepath.Join(home, ".cursor", "mcp.json"), nil
		}
		return filepath.Join(".cursor", "mcp.json"), nil
	case "vscode":
		if userScope {
			return "", fmt.Errorf("target vscode only writes project-scope configuration; rerun without --user")
		}
		return filepath.Join(".vscode", "mcp.json"), nil
	case "codex":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not resolve the home directory: %w", err)
		}
		return filepath.Join(home, ".codex", "config.toml"), nil
	default:
		return "", fmt.Errorf("unknown target '%s'; supported targets: %s", target, strings.Join(mcpInstallTargetNames(), ", "))
	}
}

// installMCPServerConfig writes the hatchet server entry into the target's
// config file, reporting whether the file was created.
func installMCPServerConfig(target, path, command string) (bool, error) {
	if target == "codex" {
		return writeCodexConfig(path, command)
	}

	topKey, entry := mcpServerJSONEntry(target, command)

	return writeMCPServerJSON(path, topKey, entry)
}

// mcpServerJSONEntry builds the per-target server entry and the top-level key
// it lives under.
func mcpServerJSONEntry(target, command string) (string, map[string]any) {
	if target == "vscode" {
		return "servers", map[string]any{"type": "stdio", "command": command, "args": mcpServeArgs}
	}

	return "mcpServers", map[string]any{"command": command, "args": mcpServeArgs}
}

// writeMCPServerJSON merges the hatchet server entry into the JSON config at
// path, preserving every other key and server. It reports whether the file
// was created.
func writeMCPServerJSON(path, topKey string, entry map[string]any) (bool, error) {
	doc := map[string]any{}
	mode := os.FileMode(0o644)
	created := false

	data, readErr := os.ReadFile(path) // #nosec G304 -- path is one of a fixed set of well-known local agent config locations
	switch {
	case readErr == nil:
		if info, statErr := os.Stat(path); statErr == nil {
			mode = info.Mode().Perm()
		}
		if len(strings.TrimSpace(string(data))) > 0 {
			if jsonErr := json.Unmarshal(data, &doc); jsonErr != nil {
				return false, fmt.Errorf("%s contains invalid JSON; fix or remove it and rerun: %v", path, jsonErr)
			}
		}
	case os.IsNotExist(readErr):
		created = true
	default:
		return false, fmt.Errorf("could not read %s: %w", path, readErr)
	}

	servers, ok := doc[topKey].(map[string]any)
	if !ok {
		if _, exists := doc[topKey]; exists {
			return false, fmt.Errorf("%s has an unexpected '%s' value; fix or remove it and rerun", path, topKey)
		}
		servers = map[string]any{}
	}
	servers[mcpServerEntryName] = entry
	doc[topKey] = servers

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return false, fmt.Errorf("could not encode %s: %w", path, err)
	}
	out = append(out, '\n')

	if dir := filepath.Dir(path); dir != "." {
		if mkdirErr := os.MkdirAll(dir, 0o755); mkdirErr != nil {
			return false, fmt.Errorf("could not create %s: %w", dir, mkdirErr)
		}
	}

	if writeErr := os.WriteFile(path, out, mode); writeErr != nil { // #nosec G306 -- non-sensitive editor config, meant to be committed
		return false, fmt.Errorf("could not write %s: %w", path, writeErr)
	}

	return created, nil
}

// codexSectionHeader is the TOML table header for the hatchet server entry.
const codexSectionHeader = "[mcp_servers." + mcpServerEntryName + "]"

// codexServerSection renders the [mcp_servers.hatchet] table.
func codexServerSection(command string) (string, error) {
	body, err := toml.Marshal(struct {
		Command string   `toml:"command"`
		Args    []string `toml:"args"`
	}{Command: command, Args: mcpServeArgs})
	if err != nil {
		return "", fmt.Errorf("could not encode the codex server entry: %w", err)
	}

	return codexSectionHeader + "\n" + string(body), nil
}

// writeCodexConfig adds or replaces the [mcp_servers.hatchet] table in the
// codex config at path. The rest of the file, including comments, is
// preserved byte for byte: only the hatchet table itself is rewritten.
func writeCodexConfig(path, command string) (bool, error) {
	section, err := codexServerSection(command)
	if err != nil {
		return false, err
	}

	existing := ""
	mode := os.FileMode(0o644)
	created := false

	data, readErr := os.ReadFile(path) // #nosec G304 -- path is the well-known codex config location under the home directory
	switch {
	case readErr == nil:
		if info, statErr := os.Stat(path); statErr == nil {
			mode = info.Mode().Perm()
		}
		existing = string(data)
		var doc map[string]any
		if tomlErr := toml.Unmarshal(data, &doc); tomlErr != nil {
			return false, fmt.Errorf("%s contains invalid TOML; fix or remove it and rerun: %v", path, tomlErr)
		}
	case os.IsNotExist(readErr):
		created = true
	default:
		return false, fmt.Errorf("could not read %s: %w", path, readErr)
	}

	merged := replaceCodexSection(existing, section)

	// Never write a config codex cannot parse back.
	var check map[string]any
	if err := toml.Unmarshal([]byte(merged), &check); err != nil {
		return false, fmt.Errorf("could not merge the server entry into %s: %v", path, err)
	}

	if mkdirErr := os.MkdirAll(filepath.Dir(path), 0o755); mkdirErr != nil {
		return false, fmt.Errorf("could not create %s: %w", filepath.Dir(path), mkdirErr)
	}

	if writeErr := os.WriteFile(path, []byte(merged), mode); writeErr != nil { // #nosec G306 G703 -- non-sensitive editor config at a fixed well-known location under the home directory
		return false, fmt.Errorf("could not write %s: %w", path, writeErr)
	}

	return created, nil
}

// replaceCodexSection splices section into existing: an existing
// [mcp_servers.hatchet] table (and its subtables) is replaced in place,
// otherwise the section is appended. Everything else is left untouched.
func replaceCodexSection(existing, section string) string {
	section = strings.TrimRight(section, "\n") + "\n"

	if strings.TrimSpace(existing) == "" {
		return section
	}

	lines := strings.Split(existing, "\n")
	start := -1
	end := len(lines)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if start == -1 {
			if trimmed == codexSectionHeader {
				start = i
			}
			continue
		}
		// The table runs until the next header that is not one of its own
		// subtables.
		if strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "[mcp_servers."+mcpServerEntryName+".") {
			end = i
			break
		}
	}

	if start == -1 {
		return strings.TrimRight(existing, "\n") + "\n\n" + section
	}

	var b strings.Builder
	if start > 0 {
		b.WriteString(strings.Join(lines[:start], "\n"))
		b.WriteString("\n")
	}
	b.WriteString(section)

	rest := strings.Join(lines[end:], "\n")
	if strings.TrimSpace(rest) != "" {
		b.WriteString("\n")
		b.WriteString(strings.TrimLeft(rest, "\n"))
	}

	return b.String()
}

// mcpInstallPrint renders the config snippets for targets without touching
// any files.
func mcpInstallPrint(targets []string, userScope bool, command string) (string, error) {
	var b strings.Builder

	for _, target := range targets {
		path, err := mcpInstallConfigPath(target, userScope)
		if err != nil {
			return "", err
		}

		snippet, err := mcpInstallSnippet(target, command)
		if err != nil {
			return "", err
		}

		fmt.Fprintf(&b, "# %s: %s\n%s\n", target, path, snippet)
	}

	return b.String(), nil
}

// mcpInstallSnippet renders the standalone config snippet for one target.
func mcpInstallSnippet(target, command string) (string, error) {
	if target == "codex" {
		return codexServerSection(command)
	}

	topKey, entry := mcpServerJSONEntry(target, command)

	out, err := json.MarshalIndent(map[string]any{topKey: map[string]any{mcpServerEntryName: entry}}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("could not encode the %s snippet: %w", target, err)
	}

	return string(out) + "\n", nil
}
