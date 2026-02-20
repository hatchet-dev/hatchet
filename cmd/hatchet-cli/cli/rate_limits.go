package cli

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/tui"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

var rateLimitsCmd = &cobra.Command{
	Use:     "rate-limits",
	Aliases: []string{"rate-limit", "rl", "rls"},
	Short:   "Manage rate limits",
	Long:    `Commands for listing and viewing rate limit usage.`,
	Run:     func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var rateLimitsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List rate limits",
	Long:  `List rate limits. Without --output json, launches the interactive TUI. With --output json, outputs raw JSON.`,
	Example: `  # Launch interactive TUI (default)
  hatchet rate-limits list --profile local

  # JSON output
  hatchet rate-limits list -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		selectedProfile, hatchetClient := clientFromCmd(cmd)

		if !isJSON {
			tuiM := newTUIModel(selectedProfile, hatchetClient)
			tuiM.currentViewType = ViewTypeRateLimits
			tuiM.currentView = tui.NewRateLimitsView(tuiM.ctx)
			p := tea.NewProgram(tuiM, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				cli.Logger.Fatalf("error running TUI: %v", err)
			}
			return
		}

		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)
		searchStr, _ := cmd.Flags().GetString("search")
		limit, _ := cmd.Flags().GetInt64("limit")
		offset, _ := cmd.Flags().GetInt64("offset")

		params := &rest.RateLimitListParams{
			Limit:  &limit,
			Offset: &offset,
		}
		if searchStr != "" {
			params.Search = &searchStr
		}

		resp, err := hatchetClient.API().RateLimitListWithResponse(ctx, tenantUUID, params)
		if err != nil {
			cli.Logger.Fatalf("failed to list rate limits: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

func init() {
	rootCmd.AddCommand(rateLimitsCmd)
	rateLimitsCmd.AddCommand(rateLimitsListCmd)

	rateLimitsCmd.PersistentFlags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	rateLimitsCmd.PersistentFlags().StringP("output", "o", "", "Output format: json (skips interactive TUI)")

	rateLimitsListCmd.Flags().StringP("search", "s", "", "Search rate limits by key")
	rateLimitsListCmd.Flags().Int64("limit", 50, "Number of results to return")
	rateLimitsListCmd.Flags().Int64("offset", 0, "Offset for pagination")
}
