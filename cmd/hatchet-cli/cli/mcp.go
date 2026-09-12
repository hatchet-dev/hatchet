package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	configcli "github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/mcp"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	profileconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
)

// mcpCmd represents the mcp parent command
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Local MCP server for AI coding agents",
	Long: `Run a local MCP (Model Context Protocol) server that lets AI coding agents
operate against a running Hatchet deployment: trigger workflow runs, inspect
run status and events, list workers, replay runs, and check engine status.

The server only uses CLI profiles you have explicitly granted with
'hatchet mcp auth'. A running embedded Hatchet instance is usable without a
grant.

Use 'hatchet mcp install' to write the server into the MCP configuration of
supported AI coding agents.`,
	Example: `  # Add the local MCP server to your AI coding agents
  hatchet mcp install

  # Grant profiles for MCP use (interactive)
  hatchet mcp auth

  # Run the MCP server over stdio (configure this command in your AI editor)
  hatchet mcp serve`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

// mcpServeCmd represents the mcp serve subcommand
var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the Hatchet MCP server over stdio",
	Long: `Run the Hatchet MCP server on stdin/stdout. This is the command to configure
in an AI editor or coding agent as a local MCP server.`,
	Example: `  # Run the server (normally launched by your AI editor, not by hand)
  hatchet mcp serve`,
	Run: func(cmd *cobra.Command, args []string) {
		runMCPServe()
	},
}

// mcpAuthCmd represents the mcp auth subcommand
var mcpAuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Grant or revoke profiles for MCP use",
	Long: `Control which CLI profiles the MCP server may use. Tool calls against
profiles that are not granted are refused.

Without flags, opens an interactive multi-select of your profiles. The '*'
entry grants every profile, including profiles added in the future.`,
	Example: `  # Interactive multi-select
  hatchet mcp auth

  # Grant specific profiles
  hatchet mcp auth --grant local,staging

  # Grant everything, including future profiles
  hatchet mcp auth --grant '*'

  # Revoke a profile / revoke everything
  hatchet mcp auth --revoke staging
  hatchet mcp auth --all

  # List current grants
  hatchet mcp auth --list`,
	Run: func(cmd *cobra.Command, args []string) {
		runMCPAuth(cmd)
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpServeCmd)
	mcpCmd.AddCommand(mcpAuthCmd)
	mcpCmd.AddCommand(mcpInstallCmd)

	mcpAuthCmd.Flags().StringSlice("grant", nil, "Profiles to grant, comma-separated ('*' grants all profiles, including future ones)")
	mcpAuthCmd.Flags().StringSlice("revoke", nil, "Profiles to revoke, comma-separated")
	mcpAuthCmd.Flags().Bool("all", false, "Revoke all grants")
	mcpAuthCmd.Flags().Bool("list", false, "List current grants")

	mcpInstallCmd.Flags().StringSlice("target", nil, "Targets to configure, comma-separated (claude-code, cursor, vscode, codex); skips the interactive prompt")
	mcpInstallCmd.Flags().Bool("user", false, "Write user-scope configuration where supported (cursor)")
	mcpInstallCmd.Flags().StringSlice("grant", nil, "Profiles to grant for MCP use, comma-separated ('*' grants all profiles, including future ones)")
	mcpInstallCmd.Flags().Bool("print", false, "Print the config snippets to stdout without writing any files")
}

// mcpEngineFactory builds the MCP Engine for a resolved profile. Client
// construction is profile-authoritative: it never reads HATCHET_CLIENT_*
// environment variables, so the profile the user granted stays the source of
// truth for the token, tenant, and endpoints. Any residual panic from the
// legacy SDK constructor is contained here so one bad profile cannot take
// down the whole stdio server.
func mcpEngineFactory(profile *profileconfig.Profile) (engine mcp.Engine, err error) {
	defer func() {
		if r := recover(); r != nil {
			engine = nil
			// Sanitized on purpose: panic values from config validation can
			// echo stored configuration.
			err = fmt.Errorf("could not initialize a client for this profile: its stored connection configuration is invalid")
		}
	}()

	nopLogger := zerolog.Nop()
	hatchetClient, err := newClientFromProfileOnly(profile, &nopLogger)
	if err != nil {
		return nil, err
	}
	return mcp.NewClientEngine(hatchetClient), nil
}

// mcpGrantStore returns the grant store next to the CLI profile store.
func mcpGrantStore() *mcp.GrantStore {
	return mcp.NewGrantStore(configcli.Profiles.Dir())
}

// mcpProfileSource adapts the CLI profile store for the MCP server.
func mcpProfileSource() mcp.ProfileSource {
	return mcp.ProfileSource{
		Profiles:       configcli.Profiles.GetProfiles,
		DefaultProfile: configcli.Profiles.GetDefaultProfile,
	}
}

func runMCPServe() {
	ctx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	server := mcp.NewServer(mcp.Deps{
		Version:     Version,
		Profiles:    mcpProfileSource(),
		Grants:      mcpGrantStore(),
		NewEngine:   mcpEngineFactory,
		Feedback:    mcp.NewHTTPFeedbackSender(),
		AnonymousID: configcli.EnsureAnonymousID(),
	})

	if err := server.Run(ctx); err != nil && ctx.Err() == nil {
		configcli.Logger.Fatalf("MCP server exited with an error: %v", err)
	}
}

func runMCPAuth(cmd *cobra.Command) {
	grantFlags, _ := cmd.Flags().GetStringSlice("grant")
	revokeFlags, _ := cmd.Flags().GetStringSlice("revoke")
	revokeAll, _ := cmd.Flags().GetBool("all")
	list, _ := cmd.Flags().GetBool("list")

	store := mcpGrantStore()

	grants, err := store.Load()
	if err != nil {
		configcli.Logger.Fatalf("could not load MCP grants: %v", err)
	}

	switch {
	case list:
		fmt.Println(mcpGrantListView(grants))
	case revokeAll:
		if len(grantFlags) > 0 || len(revokeFlags) > 0 {
			configcli.Logger.Fatal("--all cannot be combined with --grant or --revoke")
		}
		grants.Clear()
		saveMCPGrants(store, grants)
		fmt.Println(styles.SuccessMessage("All MCP grants revoked"))
	case len(grantFlags) > 0 || len(revokeFlags) > 0:
		runMCPAuthFlags(store, grants, grantFlags, revokeFlags)
	default:
		runMCPAuthInteractive(store, grants)
	}
}

// runMCPAuthFlags applies --grant / --revoke non-interactively.
func runMCPAuthFlags(store *mcp.GrantStore, grants *mcp.Grants, grantFlags, revokeFlags []string) {
	profiles := configcli.Profiles.GetProfiles()

	for _, name := range grantFlags {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}

		if name != mcp.GrantWildcard {
			if _, ok := profiles[name]; !ok {
				configcli.Logger.Fatalf("profile '%s' not found; available profiles: %s", name, strings.Join(configcli.Profiles.ListProfiles(), ", "))
			}
		}

		if grants.Add(name) {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Granted '%s' for MCP use", name)))
		} else {
			fmt.Println(styles.InfoMessage(fmt.Sprintf("'%s' is already granted", name)))
		}
	}

	for _, name := range revokeFlags {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}

		if grants.Remove(name) {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Revoked '%s'", name)))
		} else {
			fmt.Println(styles.InfoMessage(fmt.Sprintf("'%s' was not granted", name)))
		}
	}

	saveMCPGrants(store, grants)
}

// runMCPAuthInteractive opens a multi-select of profiles; the selection
// replaces the current grant set.
func runMCPAuthInteractive(store *mcp.GrantStore, grants *mcp.Grants) {
	profiles := configcli.Profiles.GetProfiles()
	defaultProfile := configcli.Profiles.GetDefaultProfile()

	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	if len(names) == 0 && !grants.HasWildcard() {
		fmt.Println(styles.InfoMessage("No profiles configured. Add one with 'hatchet profile add', or grant all future profiles with: hatchet mcp auth --grant '*'"))
		return
	}

	options := make([]huh.Option[string], 0, len(names)+1)
	options = append(options, huh.NewOption("* — all profiles (including future ones)", mcp.GrantWildcard))
	for _, name := range names {
		label := name
		if name == defaultProfile {
			label = name + " (default)"
		}
		options = append(options, huh.NewOption(label, name))
	}

	selected := make([]string, 0, len(grants.Names()))
	for _, name := range grants.Names() {
		if name == mcp.GrantWildcard {
			selected = append(selected, name)
			continue
		}
		if _, ok := profiles[name]; ok {
			selected = append(selected, name)
		}
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select profiles the MCP server may use:").
				Description("Tool calls against profiles that are not selected here are refused.").
				Options(options...).
				Value(&selected),
		),
	).WithTheme(styles.HatchetTheme())

	if err := form.Run(); err != nil {
		configcli.Logger.Fatalf("could not run the grant selection form: %v", err)
	}

	grants.Clear()
	for _, name := range selected {
		grants.Add(name)
	}

	saveMCPGrants(store, grants)

	fmt.Println(mcpGrantListView(grants))
}

func saveMCPGrants(store *mcp.GrantStore, grants *mcp.Grants) {
	if err := store.Save(grants); err != nil {
		configcli.Logger.Fatalf("could not save MCP grants: %v", err)
	}
}

func mcpGrantListView(grants *mcp.Grants) string {
	names := grants.Names()
	if len(names) == 0 {
		return styles.InfoMessage("No profiles are granted for MCP use. Run 'hatchet mcp auth' to grant some.")
	}

	var lines []string
	lines = append(lines, styles.Section("Profiles granted for MCP use"))
	for _, name := range names {
		label := name
		if name == mcp.GrantWildcard {
			label = "* (all profiles, including future ones)"
		}
		lines = append(lines, styles.ListItem.Render(styles.Accent.Render("• ")+label))
	}

	return strings.Join(lines, "\n")
}
