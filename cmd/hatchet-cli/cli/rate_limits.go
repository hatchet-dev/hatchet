package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/tui"
	"github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

var rateLimitDurations = []types.RateLimitDuration{
	types.Second,
	types.Minute,
	types.Hour,
	types.Day,
	types.Week,
	types.Month,
	types.Year,
}

var rateLimitsCmd = &cobra.Command{
	Use:     "rate-limits",
	Aliases: []string{"rate-limit", "rl", "rls"},
	Short:   "Manage rate limits",
	Long:    `Commands for listing, viewing, creating, updating, and deleting rate limits.`,
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

var rateLimitsGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a rate limit",
	Long:  `Get details about a single rate limit by key. Outputs raw JSON.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		_, hatchetClient := clientFromCmd(cmd)

		rl := findRateLimitByKey(cmd.Context(), hatchetClient, key)
		if rl == nil {
			cli.Logger.Fatalf("rate limit with key %q not found", key)
		}

		printJSON(rl)
	},
}

var rateLimitsCreateCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"update", "put", "set"},
	Short:   "Create or update a rate limit",
	Long:    `Create or update (upsert) a rate limit. In --output json mode, all required flags must be set. Otherwise, launches an interactive form.`,
	Example: `  # Interactive mode
  hatchet rate-limits create --profile local

  # JSON mode (required flags)
  hatchet rate-limits create --key my-key --limit 100 --duration minute -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)

		key, _ := cmd.Flags().GetString("key")
		limit, _ := cmd.Flags().GetInt("limit")
		durationStr, _ := cmd.Flags().GetString("duration")

		if !isJSON {
			limitStr := ""
			if cmd.Flags().Changed("limit") {
				limitStr = strconv.Itoa(limit)
			}
			if durationStr == "" {
				durationStr = string(types.Minute)
			}

			durationOptions := make([]huh.Option[string], 0, len(rateLimitDurations))
			for _, d := range rateLimitDurations {
				durationOptions = append(durationOptions, huh.NewOption(string(d), string(d)))
			}

			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Rate limit key").
						Value(&key).
						Placeholder("my-key"),
				),
				huh.NewGroup(
					huh.NewInput().
						Title("Limit (max requests per window)").
						Value(&limitStr).
						Placeholder("100").
						Validate(func(s string) error {
							n, err := strconv.Atoi(strings.TrimSpace(s))
							if err != nil {
								return fmt.Errorf("must be a number")
							}
							if n <= 0 {
								return fmt.Errorf("must be greater than 0")
							}
							return nil
						}),
				),
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("Window duration").
						Options(durationOptions...).
						Value(&durationStr),
				),
			).WithTheme(styles.HatchetTheme())
			if err := form.Run(); err != nil {
				cli.Logger.Fatalf("form cancelled: %v", err)
			}

			parsed, err := strconv.Atoi(strings.TrimSpace(limitStr))
			if err != nil {
				cli.Logger.Fatalf("invalid limit: %v", err)
			}
			limit = parsed
		}

		if key == "" {
			cli.Logger.Fatal("--key is required")
		}
		if limit <= 0 {
			cli.Logger.Fatal("--limit is required and must be greater than 0")
		}

		duration, err := parseRateLimitDuration(durationStr)
		if err != nil {
			cli.Logger.Fatalf("%v", err)
		}

		if err := hatchetClient.Admin().PutRateLimit(key, &types.RateLimitOpts{
			Max:      limit,
			Duration: duration,
		}); err != nil {
			cli.Logger.Fatalf("failed to upsert rate limit: %v", err)
		}

		if isJSON {
			printJSON(map[string]interface{}{
				"key":      key,
				"limit":    limit,
				"duration": string(duration),
			})
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Saved rate limit: %s (%d per %s)", key, limit, duration)))
		}
	},
}

var rateLimitsDeleteCmd = &cobra.Command{
	Use:   "delete [key]",
	Short: "Delete a rate limit",
	Long:  `Delete a rate limit by key. Omit the key to pick from a list interactively. Use --yes to skip confirmation.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		yes, _ := cmd.Flags().GetBool("yes")
		_, hatchetClient := clientFromCmd(cmd)
		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)

		var key string
		if len(args) == 1 {
			key = args[0]
		} else if !isJSON {
			limit := int64(100)
			listResp, listErr := hatchetClient.API().RateLimitListWithResponse(ctx, tenantUUID, &rest.RateLimitListParams{
				Limit: &limit,
			})
			if listErr != nil {
				cli.Logger.Fatalf("failed to list rate limits: %v", listErr)
			}
			if listResp.JSON200 == nil || listResp.JSON200.Rows == nil || len(*listResp.JSON200.Rows) == 0 {
				cli.Logger.Fatal("no rate limits found")
			}

			var options []huh.Option[string]
			for _, rl := range *listResp.JSON200.Rows {
				label := fmt.Sprintf("%s  (%d per %s)", rl.Key, rl.LimitValue, rl.Window)
				options = append(options, huh.NewOption(label, rl.Key))
			}

			height := len(options)
			if height > 10 {
				height = 10
			}
			form := huh.NewForm(huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select a rate limit to delete").
					Options(options...).
					Height(height).
					Value(&key),
			)).WithTheme(styles.HatchetTheme())
			if formErr := form.Run(); formErr != nil {
				cli.Logger.Fatalf("selection cancelled: %v", formErr)
			}
		} else {
			cli.Logger.Fatal("rate limit key is required in JSON mode")
		}

		if key == "" {
			cli.Logger.Fatal("rate limit key is required")
		}

		if !isJSON && !yes {
			if !confirmAction(fmt.Sprintf("Delete rate limit '%s'?", key)) {
				fmt.Println("Aborted.")
				return
			}
		}

		resp, err := hatchetClient.API().RateLimitDeleteWithResponse(ctx, tenantUUID, &rest.RateLimitDeleteParams{
			Key: key,
		})
		if err != nil {
			cli.Logger.Fatalf("failed to delete rate limit: %v", err)
		}
		if resp.StatusCode() >= 400 {
			cli.Logger.Fatalf("failed to delete rate limit (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(map[string]interface{}{"deleted": true, "key": key})
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Deleted rate limit: %s", key)))
		}
	},
}

// findRateLimitByKey looks up a single rate limit by exact key using the list
// endpoint's search filter (there is no dedicated get-by-key endpoint).
func findRateLimitByKey(ctx context.Context, hatchetClient client.Client, key string) *rest.RateLimit { //nolint:staticcheck
	tenantUUID := clientTenantUUID(hatchetClient)
	limit := int64(100)
	resp, err := hatchetClient.API().RateLimitListWithResponse(ctx, tenantUUID, &rest.RateLimitListParams{
		Limit:  &limit,
		Search: &key,
	})
	if err != nil {
		cli.Logger.Fatalf("failed to get rate limit: %v", err)
	}
	if resp.JSON200 == nil || resp.JSON200.Rows == nil {
		return nil
	}
	for i := range *resp.JSON200.Rows {
		if (*resp.JSON200.Rows)[i].Key == key {
			return &(*resp.JSON200.Rows)[i]
		}
	}
	return nil
}

// parseRateLimitDuration validates a duration string against the supported set.
func parseRateLimitDuration(s string) (types.RateLimitDuration, error) {
	d := types.RateLimitDuration(strings.ToLower(strings.TrimSpace(s)))
	for _, valid := range rateLimitDurations {
		if d == valid {
			return d, nil
		}
	}
	names := make([]string, len(rateLimitDurations))
	for i, valid := range rateLimitDurations {
		names[i] = string(valid)
	}
	return "", fmt.Errorf("invalid duration %q (must be one of: %s)", s, strings.Join(names, ", "))
}

func init() {
	rootCmd.AddCommand(rateLimitsCmd)
	rateLimitsCmd.AddCommand(rateLimitsListCmd, rateLimitsGetCmd, rateLimitsCreateCmd, rateLimitsDeleteCmd)

	rateLimitsCmd.PersistentFlags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	rateLimitsCmd.PersistentFlags().StringP("output", "o", "", "Output format: json (skips interactive TUI)")

	rateLimitsListCmd.Flags().StringP("search", "s", "", "Search rate limits by key")
	rateLimitsListCmd.Flags().Int64("limit", 50, "Number of results to return")
	rateLimitsListCmd.Flags().Int64("offset", 0, "Offset for pagination")

	rateLimitsCreateCmd.Flags().StringP("key", "k", "", "Rate limit key")
	rateLimitsCreateCmd.Flags().IntP("limit", "l", 0, "Maximum number of requests allowed within the window")
	rateLimitsCreateCmd.Flags().StringP("duration", "d", "", "Window duration: second, minute, hour, day, week, month, year")

	rateLimitsDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}
