package cli

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/unstable"
	"github.com/spf13/cobra"

	configcli "github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/mcp"
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

  # Print the config snippets without modifying any agent configuration or grants
  hatchet mcp install --print`,
	Run: func(cmd *cobra.Command, args []string) {
		runMCPInstall(cmd)
	},
}

const mcpServerEntryName = "hatchet"

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

	// Two-phase install: prepare (read, merge, refuse) every target and
	// preflight the grant request before the first write, so a predictable
	// validation failure cannot leave a partial install behind.
	prepared, err := prepareMCPInstalls(targets, paths, command)
	if err != nil {
		configcli.Logger.Fatalf("%v", err)
	}
	defer func() {
		for _, p := range prepared {
			p.close()
		}
	}()

	store := mcpGrantStore()
	var grants *mcp.Grants
	if len(grantFlags) > 0 {
		var loadErr error
		grants, loadErr = store.Load()
		if loadErr != nil {
			configcli.Logger.Fatalf("could not load MCP grants: %v", loadErr)
		}
		if grantErr := validateMCPGrantProfiles(grantFlags); grantErr != nil {
			configcli.Logger.Fatalf("%v", grantErr)
		}
	}

	written := make([]string, 0, len(prepared))
	for _, p := range prepared {
		if commitErr := p.commit(); commitErr != nil {
			if len(written) > 0 {
				configcli.Logger.Fatalf("%v (already written: %s)", commitErr, strings.Join(written, ", "))
			}
			configcli.Logger.Fatalf("%v", commitErr)
		}
		written = append(written, p.path)

		verb := "Updated"
		if p.created {
			verb = "Created"
		}
		fmt.Println(styles.SuccessMessage(fmt.Sprintf("%s %s (%s)", verb, p.path, p.target)))
	}

	if len(grantFlags) > 0 {
		fmt.Println()
		runMCPAuthFlags(store, grants, grantFlags, nil)

		return
	}

	grants, loadErr := store.Load()

	// No grant operation was requested, so the grant state is informational
	// only: a problem with the grants file must not fail an install whose
	// configs are already written.
	if loadErr != nil {
		fmt.Println()
		fmt.Println(styles.InfoMessage(fmt.Sprintf("Could not read the MCP grants file (%v). Run 'hatchet mcp auth' to manage grants.", loadErr)))
		return
	}

	if interactive && len(grants.Names()) == 0 {
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
			if reloaded, reloadErr := store.Load(); reloadErr == nil {
				grants = reloaded
			}
		}
	}

	if len(grants.Names()) == 0 {
		fmt.Println()
		fmt.Println(styles.InfoMessage("No profiles are granted for MCP use yet. Run 'hatchet mcp auth' to grant some. A running embedded Hatchet instance is usable without a grant."))
	}
}

// validateMCPGrantProfiles mirrors the profile check runMCPAuthFlags applies,
// so an unknown --grant name fails before any config file is written.
func validateMCPGrantProfiles(grantFlags []string) error {
	profiles := configcli.Profiles.GetProfiles()

	for _, name := range grantFlags {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || name == mcp.GrantWildcard {
			continue
		}
		if _, ok := profiles[name]; !ok {
			return fmt.Errorf("profile '%s' not found; available profiles: %s", name, strings.Join(configcli.Profiles.ListProfiles(), ", "))
		}
	}

	return nil
}

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
//
// A PATH hit inside the current directory is never trusted: agents resolve
// the configured command from the same directory later, and a checkout-local
// binary must never be what they launch.
func resolveMCPCommand() string {
	if found, err := exec.LookPath("hatchet"); err == nil && mcpCommandOutsideCwd(found) {
		return "hatchet"
	}

	if exe, err := os.Executable(); err == nil {
		return exe
	}

	return "hatchet"
}

// mcpCommandOutsideCwd reports whether a LookPath result is an absolute path
// outside the current working directory. Relative results (including the
// exec.ErrDot cases) and anything under the checkout are rejected.
func mcpCommandOutsideCwd(found string) bool {
	if !filepath.IsAbs(found) {
		return false
	}

	cwd, err := os.Getwd()
	if err != nil {
		return false
	}

	if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
		cwd = resolved
	}
	if resolved, err := filepath.EvalSymlinks(found); err == nil {
		found = resolved
	}

	rel, err := filepath.Rel(cwd, found)
	if err != nil {
		// No relative path exists (e.g. different volumes), so the hit
		// cannot be inside the current directory.
		return true
	}

	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
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

// mcpInstallPrepared is one target's install split into two phases: prepare
// does all reads, merging, and refusals; commit does the only writes. The
// open root pins the directory tree so every filesystem operation stays
// inside it.
type mcpInstallPrepared struct {
	target  string
	path    string // as shown to the user
	root    *os.Root
	rel     string // path relative to root
	prior   []byte // bytes read at prepare time; nil when the file was absent
	merged  []byte
	mode    os.FileMode
	created bool
}

// prepareMCPInstalls runs the prepare phase for every target. Nothing is
// written; on any error every already-open root is closed and no file has
// been touched.
func prepareMCPInstalls(targets []string, paths map[string]string, command string) ([]*mcpInstallPrepared, error) {
	prepared := make([]*mcpInstallPrepared, 0, len(targets))

	for _, target := range targets {
		p, err := prepareMCPInstall(target, paths[target], command)
		if err != nil {
			for _, q := range prepared {
				q.close()
			}
			return nil, err
		}
		prepared = append(prepared, p)
	}

	return prepared, nil
}

// prepareMCPInstall reads the target's config through a pinned root and
// computes the merged content. All refusal errors surface here, before any
// write.
func prepareMCPInstall(target, path, command string) (*mcpInstallPrepared, error) {
	root, rel, err := openMCPConfigRoot(path)
	if err != nil {
		return nil, err
	}

	p := &mcpInstallPrepared{target: target, path: path, root: root, rel: rel, mode: 0o644}

	data, readErr := root.ReadFile(rel)
	switch {
	case readErr == nil:
		p.prior = data
		if info, statErr := root.Stat(rel); statErr == nil {
			p.mode = info.Mode().Perm()
		}
	case errors.Is(readErr, fs.ErrNotExist):
		p.created = true
	default:
		p.close()
		return nil, fmt.Errorf("could not read %s: %w", path, readErr)
	}

	var merged []byte
	var mergeErr error
	if target == "codex" {
		merged, mergeErr = mergeCodexConfig(p.prior, command, path)
	} else {
		topKey, entry := mcpServerJSONEntry(target, command)
		merged, mergeErr = mergeMCPServerJSON(p.prior, topKey, entry, path)
	}
	if mergeErr != nil {
		p.close()
		return nil, mergeErr
	}
	p.merged = merged

	return p, nil
}

// commit writes the prepared content through the pinned root.
func (p *mcpInstallPrepared) commit() error {
	if dir := filepath.Dir(p.rel); dir != "." {
		if err := p.root.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("could not create %s: %w", dir, err)
		}
	}

	// Best-effort lost-update check, not a transaction: a write landing
	// between this read and the rename below can still be discarded.
	current, err := p.root.ReadFile(p.rel)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("could not read %s: %w", p.path, err)
	}
	if !bytes.Equal(current, p.prior) {
		return fmt.Errorf("%s changed while installing; rerun the command", p.path)
	}

	if writeErr := writeFileAtomicInRoot(p.root, p.rel, p.merged, p.mode); writeErr != nil {
		return fmt.Errorf("could not write %s: %w", p.path, writeErr)
	}

	return nil
}

func (p *mcpInstallPrepared) close() {
	_ = p.root.Close()
}

// installMCPServerConfig writes the hatchet server entry into the target's
// config file, reporting whether the file was created.
func installMCPServerConfig(target, path, command string) (bool, error) {
	p, err := prepareMCPInstall(target, path, command)
	if err != nil {
		return false, err
	}
	defer p.close()

	if err := p.commit(); err != nil {
		return false, err
	}

	return p.created, nil
}

// openMCPConfigRoot opens the directory tree a config path may be written
// under: the current directory for project-scope paths, the home directory
// for user-scope paths. os.Root enforcement makes the confinement hold even
// against concurrent symlink swaps.
func openMCPConfigRoot(path string) (*os.Root, string, error) {
	if !filepath.IsAbs(path) {
		// A repository checkout can commit a project config path (.mcp.json,
		// .cursor, .vscode) as a symlink pointing outside the project. The
		// root confines the write either way; this pre-check exists to give
		// the static case a clear error instead of a generic escape failure.
		if err := rejectSymlinkComponents(path); err != nil {
			return nil, "", err
		}

		root, err := os.OpenRoot(".")
		if err != nil {
			return nil, "", fmt.Errorf("could not open the current directory: %w", err)
		}
		return root, path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, "", fmt.Errorf("could not resolve the home directory: %w", err)
	}

	rel, err := filepath.Rel(home, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, "", fmt.Errorf("%s is outside the home directory %s", path, home)
	}

	root, err := os.OpenRoot(home)
	if err != nil {
		return nil, "", fmt.Errorf("could not open %s: %w", home, err)
	}

	return root, rel, nil
}

// rejectSymlinkComponents fails when any existing component of the relative
// path is a symbolic link.
func rejectSymlinkComponents(path string) error {
	components := strings.Split(filepath.ToSlash(path), "/")
	for i := range components {
		prefix := filepath.Join(components[:i+1]...)

		info, err := os.Lstat(prefix)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("could not inspect %s: %w", prefix, err)
		}

		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symbolic link; refusing to write through it", prefix)
		}
	}

	return nil
}

func mcpServerJSONEntry(target, command string) (string, map[string]any) {
	// VS Code uses a different schema than the mcpServers convention: a
	// top-level "servers" key with an explicit transport type.
	if target == "vscode" {
		return "servers", map[string]any{"type": "stdio", "command": command, "args": mcpServeArgs}
	}

	return "mcpServers", map[string]any{"command": command, "args": mcpServeArgs}
}

// mergeMCPServerJSON adds or replaces only the hatchet entry under topKey;
// everything else in the file survives so an install can never clobber other
// servers or unknown keys. Numbers are decoded as json.Number so unrelated
// values round-trip exactly.
func mergeMCPServerJSON(prior []byte, topKey string, entry map[string]any, path string) ([]byte, error) {
	doc := map[string]any{}

	if len(bytes.TrimSpace(prior)) > 0 {
		dec := json.NewDecoder(bytes.NewReader(prior))
		dec.UseNumber()

		var parsed any
		if err := dec.Decode(&parsed); err != nil {
			return nil, fmt.Errorf("%s contains invalid JSON; fix or remove it and rerun: %v", path, err)
		}
		if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%s contains more than one JSON document; fix or remove it and rerun", path)
		}

		obj, ok := parsed.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s does not contain a JSON object; fix or remove it and rerun", path)
		}
		doc = obj
	}

	servers, ok := doc[topKey].(map[string]any)
	if !ok {
		if _, exists := doc[topKey]; exists {
			return nil, fmt.Errorf("%s has an unexpected '%s' value; fix or remove it and rerun", path, topKey)
		}
		servers = map[string]any{}
	}
	servers[mcpServerEntryName] = entry
	doc[topKey] = servers

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("could not encode %s: %w", path, err)
	}

	return append(out, '\n'), nil
}

// writeFileAtomicInRoot writes via a same-directory temp file and rename, so
// an interruption or write error cannot leave the user's existing config
// truncated or half-written. Every operation goes through the root so a
// concurrently introduced symlink cannot redirect it.
func writeFileAtomicInRoot(root *os.Root, path string, data []byte, mode os.FileMode) error {
	var tmp *os.File
	var tmpName string

	for range 10 {
		var suffix [4]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return err
		}

		name := path + ".tmp-" + hex.EncodeToString(suffix[:])
		f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			if errors.Is(err, fs.ErrExist) {
				continue
			}
			return err
		}

		tmp, tmpName = f, name
		break
	}
	if tmp == nil {
		return fmt.Errorf("could not create a temporary file next to %s", path)
	}
	defer func() { _ = root.Remove(tmpName) }()

	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return root.Rename(tmpName, path)
}

const codexSectionHeader = "[mcp_servers." + mcpServerEntryName + "]"

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

// mergeCodexConfig adds or replaces the [mcp_servers.hatchet] table in the
// codex config bytes. Source spans owned by the hatchet subtree are located
// with the TOML parser (decoded keys, byte offsets) and removed, and the
// fresh table is appended; every untouched span, comments included, is
// preserved byte for byte.
func mergeCodexConfig(prior []byte, command, path string) ([]byte, error) {
	section, err := codexServerSection(command)
	if err != nil {
		return nil, err
	}

	if len(prior) > 0 {
		var doc map[string]any
		if tomlErr := toml.Unmarshal(prior, &doc); tomlErr != nil {
			return nil, fmt.Errorf("%s contains invalid TOML; fix or remove it and rerun: %v", path, tomlErr)
		}
	}

	merged, err := spliceCodexConfig(prior, section, path)
	if err != nil {
		return nil, err
	}

	if err := verifyCodexMerge(prior, merged, command, path); err != nil {
		return nil, err
	}

	return merged, nil
}

// codexTopExpr is one top-level TOML expression with its decoded key path and
// the byte offset of the line it starts on. Its span runs to the start of the
// next expression, which keeps trailing comments and blank lines attached.
type codexTopExpr struct {
	kind      unstable.Kind
	keys      []string
	valueKind unstable.Kind
	start     int
}

func parseCodexTopExprs(data []byte) ([]codexTopExpr, error) {
	parser := &unstable.Parser{}
	parser.Reset(data)

	var exprs []codexTopExpr
	for parser.NextExpression() {
		node := parser.Expression()

		switch node.Kind {
		case unstable.Table, unstable.ArrayTable, unstable.KeyValue:
		default:
			continue
		}

		expr := codexTopExpr{kind: node.Kind, start: -1}
		it := node.Key()
		for it.Next() {
			key := it.Node()
			if expr.start < 0 {
				expr.start = lineStart(data, int(key.Raw.Offset))
			}
			expr.keys = append(expr.keys, string(key.Data))
		}
		if node.Kind == unstable.KeyValue {
			expr.valueKind = node.Value().Kind
		}
		if expr.start < 0 {
			return nil, fmt.Errorf("expression without a key")
		}

		exprs = append(exprs, expr)
	}
	if err := parser.Error(); err != nil {
		return nil, err
	}

	return exprs, nil
}

// lineStart backs an in-expression byte offset up to the start of its line,
// so a table header span includes its opening bracket.
func lineStart(data []byte, offset int) int {
	if i := bytes.LastIndexByte(data[:offset], '\n'); i >= 0 {
		return i + 1
	}
	return 0
}

// attachedCommentStart backs an expression's line start up over the comment
// lines directly above it (no blank line in between), which document that
// expression rather than whatever precedes it. A misjudgment here cannot
// corrupt the file: the merged result still has to pass verifyCodexMerge.
func attachedCommentStart(data []byte, exprStart int) int {
	start := exprStart
	for start > 0 {
		prev := lineStart(data, start-1)
		line := strings.TrimSpace(string(data[prev:start]))
		if !strings.HasPrefix(line, "#") {
			break
		}
		start = prev
	}
	return start
}

func isHatchetKeyPath(keys []string) bool {
	return len(keys) >= 2 && keys[0] == "mcp_servers" && keys[1] == mcpServerEntryName
}

// spliceCodexConfig removes every source span belonging to the exact
// mcp_servers.hatchet subtree (matching decoded keys, wherever the spans
// appear in the file) and appends section. Representations that cannot be
// spliced (inline tables, array of tables) are refused explicitly.
func spliceCodexConfig(data []byte, section, path string) ([]byte, error) {
	section = strings.TrimRight(section, "\n") + "\n"

	if len(bytes.TrimSpace(data)) == 0 {
		return []byte(section), nil
	}

	exprs, err := parseCodexTopExprs(data)
	if err != nil {
		return nil, fmt.Errorf("%s contains invalid TOML; fix or remove it and rerun: %v", path, err)
	}

	owned := make([]bool, len(exprs))
	inHatchet := false
	var context []string

	for i, expr := range exprs {
		switch expr.kind {
		case unstable.Table:
			context = expr.keys
			inHatchet = isHatchetKeyPath(expr.keys)
			owned[i] = inHatchet
		case unstable.ArrayTable:
			if isHatchetKeyPath(expr.keys) {
				return nil, fmt.Errorf("%s defines mcp_servers.hatchet as an array of tables; edit the file manually to update it", path)
			}
			context = expr.keys
			inHatchet = false
		case unstable.KeyValue:
			if inHatchet {
				owned[i] = true
				continue
			}

			effective := make([]string, 0, len(context)+len(expr.keys))
			effective = append(effective, context...)
			effective = append(effective, expr.keys...)

			if expr.valueKind == unstable.InlineTable {
				if len(effective) == 1 && effective[0] == "mcp_servers" {
					return nil, fmt.Errorf("%s defines mcp_servers as an inline table; edit the file manually to add the hatchet entry", path)
				}
				if len(effective) == 2 && isHatchetKeyPath(effective) {
					return nil, fmt.Errorf("%s defines mcp_servers.hatchet as an inline table; edit the file manually to update it", path)
				}
			}

			if isHatchetKeyPath(effective) {
				owned[i] = true
			}
		}
	}

	var rest bytes.Buffer
	prev := 0
	for i, expr := range exprs {
		if !owned[i] {
			continue
		}

		end := len(data)
		if i+1 < len(exprs) {
			end = exprs[i+1].start
			// A comment block contiguous with the next unrelated expression
			// documents that expression, not the hatchet entry: leave it in
			// place. Comments above a removed hatchet subtable go with it.
			if !owned[i+1] {
				end = attachedCommentStart(data, end)
			}
		}
		if expr.start > prev {
			rest.Write(data[prev:expr.start])
		}
		if end > prev {
			prev = end
		}
	}
	rest.Write(data[prev:])

	remainder := rest.String()
	if strings.TrimSpace(remainder) == "" {
		return []byte(section), nil
	}

	return []byte(strings.TrimRight(remainder, "\n") + "\n\n" + section), nil
}

// verifyCodexMerge is the safety net that turns any remaining splice bug into
// a refusal: apart from the mcp_servers.hatchet subtree the merged document
// must parse to exactly the original, and the subtree must be exactly the
// configured entry. Nothing is written when this fails.
func verifyCodexMerge(original, merged []byte, command, path string) error {
	var before, after map[string]any
	if err := toml.Unmarshal(original, &before); err != nil {
		return fmt.Errorf("%s contains invalid TOML; fix or remove it and rerun: %v", path, err)
	}
	if err := toml.Unmarshal(merged, &after); err != nil {
		return fmt.Errorf("could not merge the server entry into %s: %v", path, err)
	}
	if before == nil {
		before = map[string]any{}
	}
	if after == nil {
		after = map[string]any{}
	}

	servers, _ := after["mcp_servers"].(map[string]any)
	entry, _ := servers[mcpServerEntryName].(map[string]any)
	wantArgs := make([]any, len(mcpServeArgs))
	for i, arg := range mcpServeArgs {
		wantArgs[i] = arg
	}
	if !reflect.DeepEqual(entry, map[string]any{"command": command, "args": wantArgs}) {
		return fmt.Errorf("could not safely update %s; edit the file manually to add the hatchet entry", path)
	}

	stripCodexHatchet(before)
	stripCodexHatchet(after)
	if !reflect.DeepEqual(before, after) {
		return fmt.Errorf("could not safely update %s; edit the file manually to add the hatchet entry", path)
	}

	return nil
}

func stripCodexHatchet(doc map[string]any) {
	servers, ok := doc["mcp_servers"].(map[string]any)
	if !ok {
		return
	}
	delete(servers, mcpServerEntryName)
	if len(servers) == 0 {
		delete(doc, "mcp_servers")
	}
}

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
