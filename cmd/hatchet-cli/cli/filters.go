package cli

import (
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/tui"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

var filtersCmd = &cobra.Command{
	Use:     "filters",
	Aliases: []string{"filter"},
	Short:   "Manage filters",
	Long:    `Commands for listing, viewing, creating, updating, and deleting filters.`,
	Run:     func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var filtersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List filters",
	Long:  `List filters. Without --output json, launches the interactive TUI. With --output json, outputs raw JSON.`,
	Example: `  # Launch interactive TUI (default)
  hatchet filters list --profile local

  # JSON output
  hatchet filters list -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		selectedProfile, hatchetClient := clientFromCmd(cmd)

		if !isJSON {
			tuiM := newTUIModel(selectedProfile, hatchetClient)
			tuiM.currentViewType = ViewTypeFilters
			tuiM.currentView = tui.NewFiltersView(tuiM.ctx)
			p := tea.NewProgram(tuiM, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				cli.Logger.Fatalf("error running TUI: %v", err)
			}
			return
		}

		workflowIDStrs, _ := cmd.Flags().GetStringSlice("workflow-ids")
		scopes, _ := cmd.Flags().GetStringSlice("scopes")
		limit, _ := cmd.Flags().GetInt64("limit")
		offset, _ := cmd.Flags().GetInt64("offset")

		params := &rest.V1FilterListParams{
			Limit:  &limit,
			Offset: &offset,
		}
		if len(workflowIDStrs) > 0 {
			workflowIDs := make([]openapi_types.UUID, 0, len(workflowIDStrs))
			for _, s := range workflowIDStrs {
				id, err := uuid.Parse(s)
				if err != nil {
					cli.Logger.Fatalf("invalid workflow id %q: %v", s, err)
				}
				workflowIDs = append(workflowIDs, id)
			}
			params.WorkflowIds = &workflowIDs
		}
		if len(scopes) > 0 {
			params.Scopes = &scopes
		}

		resp, err := hatchetClient.API().V1FilterListWithResponse(cmd.Context(), clientTenantUUID(hatchetClient), params)
		if err != nil {
			cli.Logger.Fatalf("failed to list filters: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var filtersGetCmd = &cobra.Command{
	Use:   "get <filter-id>",
	Short: "Get a filter",
	Long:  `Get details about a single filter by id. Outputs raw JSON.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filterUUID, err := uuid.Parse(args[0])
		if err != nil {
			cli.Logger.Fatalf("invalid filter id %q: %v", args[0], err)
		}
		_, hatchetClient := clientFromCmd(cmd)

		resp, err := hatchetClient.API().V1FilterGetWithResponse(cmd.Context(), clientTenantUUID(hatchetClient), filterUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to get filter: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var filtersCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a filter",
	Long:  `Create a filter. In --output json mode, all required flags must be set. Otherwise, launches an interactive form for missing values.`,
	Example: `  # Interactive mode
  hatchet filters create --profile local

  # JSON mode (required flags)
  hatchet filters create --workflow-id <uuid> --expression "input.value > 5" --scope my-scope -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)

		workflowIDStr, _ := cmd.Flags().GetString("workflow-id")
		expression, _ := cmd.Flags().GetString("expression")
		scope, _ := cmd.Flags().GetString("scope")
		payloadStr, _ := cmd.Flags().GetString("payload")

		if !isJSON && (workflowIDStr == "" || expression == "" || scope == "") {
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Workflow id").
						Value(&workflowIDStr).
						Validate(func(s string) error {
							if _, err := uuid.Parse(s); err != nil {
								return fmt.Errorf("must be a valid UUID")
							}
							return nil
						}),
				),
				huh.NewGroup(
					huh.NewInput().
						Title("Expression").
						Value(&expression).
						Placeholder("input.value > 5"),
				),
				huh.NewGroup(
					huh.NewInput().
						Title("Scope").
						Value(&scope),
				),
				huh.NewGroup(
					huh.NewText().
						Title("Payload (JSON, optional)").
						Value(&payloadStr).
						Validate(func(s string) error {
							if s == "" {
								return nil
							}
							var m map[string]interface{}
							if err := json.Unmarshal([]byte(s), &m); err != nil {
								return fmt.Errorf("must be a JSON object")
							}
							return nil
						}),
				),
			).WithTheme(styles.HatchetTheme())
			if err := form.Run(); err != nil {
				cli.Logger.Fatalf("form cancelled: %v", err)
			}
		}

		if workflowIDStr == "" {
			cli.Logger.Fatal("--workflow-id is required")
		}
		if expression == "" {
			cli.Logger.Fatal("--expression is required")
		}
		if scope == "" {
			cli.Logger.Fatal("--scope is required")
		}

		workflowID, err := uuid.Parse(workflowIDStr)
		if err != nil {
			cli.Logger.Fatalf("invalid workflow id %q: %v", workflowIDStr, err)
		}

		body := rest.V1CreateFilterRequest{
			WorkflowId: workflowID,
			Expression: expression,
			Scope:      scope,
		}
		if payloadStr != "" {
			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
				cli.Logger.Fatalf("invalid payload JSON: %v", err)
			}
			body.Payload = &payload
		}

		resp, err := hatchetClient.API().V1FilterCreateWithResponse(cmd.Context(), clientTenantUUID(hatchetClient), body)
		if err != nil {
			cli.Logger.Fatalf("failed to create filter: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(resp.JSON200)
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Created filter: %s", resp.JSON200.Metadata.Id)))
		}
	},
}

var filtersUpdateCmd = &cobra.Command{
	Use:   "update <filter-id>",
	Short: "Update a filter",
	Long:  `Update a filter. Only the provided flags are changed.`,
	Example: `  # Update the expression
  hatchet filters update <uuid> --expression "input.value > 10" -o json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		filterUUID, err := uuid.Parse(args[0])
		if err != nil {
			cli.Logger.Fatalf("invalid filter id %q: %v", args[0], err)
		}
		_, hatchetClient := clientFromCmd(cmd)

		body := rest.V1UpdateFilterRequest{}
		if cmd.Flags().Changed("expression") {
			expression, _ := cmd.Flags().GetString("expression")
			body.Expression = &expression
		}
		if cmd.Flags().Changed("scope") {
			scope, _ := cmd.Flags().GetString("scope")
			body.Scope = &scope
		}
		if cmd.Flags().Changed("payload") {
			payloadStr, _ := cmd.Flags().GetString("payload")
			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
				cli.Logger.Fatalf("invalid payload JSON: %v", err)
			}
			body.Payload = &payload
		}

		if body.Expression == nil && body.Scope == nil && body.Payload == nil {
			cli.Logger.Fatal("at least one of --expression, --scope, or --payload is required")
		}

		resp, err := hatchetClient.API().V1FilterUpdateWithResponse(cmd.Context(), clientTenantUUID(hatchetClient), filterUUID, body)
		if err != nil {
			cli.Logger.Fatalf("failed to update filter: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(resp.JSON200)
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Updated filter: %s", resp.JSON200.Metadata.Id)))
		}
	},
}

var filtersDeleteCmd = &cobra.Command{
	Use:   "delete <filter-id>",
	Short: "Delete a filter",
	Long:  `Delete a filter by id. Use --yes to skip confirmation.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		yes, _ := cmd.Flags().GetBool("yes")
		filterUUID, err := uuid.Parse(args[0])
		if err != nil {
			cli.Logger.Fatalf("invalid filter id %q: %v", args[0], err)
		}
		_, hatchetClient := clientFromCmd(cmd)

		if !isJSON && !yes {
			if !confirmAction(fmt.Sprintf("Delete filter '%s'?", filterUUID)) {
				fmt.Println("Aborted.")
				return
			}
		}

		resp, err := hatchetClient.API().V1FilterDeleteWithResponse(cmd.Context(), clientTenantUUID(hatchetClient), filterUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to delete filter: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(map[string]interface{}{"deleted": true, "id": filterUUID.String()})
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Deleted filter: %s", filterUUID)))
		}
	},
}

func init() {
	rootCmd.AddCommand(filtersCmd)
	filtersCmd.AddCommand(filtersListCmd, filtersGetCmd, filtersCreateCmd, filtersUpdateCmd, filtersDeleteCmd)

	filtersCmd.PersistentFlags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	filtersCmd.PersistentFlags().StringP("output", "o", "", "Output format: json (skips interactive TUI)")

	filtersListCmd.Flags().StringSlice("workflow-ids", nil, "Filter by workflow ids")
	filtersListCmd.Flags().StringSlice("scopes", nil, "Filter by scopes")
	filtersListCmd.Flags().Int64("limit", 50, "Number of results to return")
	filtersListCmd.Flags().Int64("offset", 0, "Offset for pagination")

	filtersCreateCmd.Flags().StringP("workflow-id", "w", "", "Workflow id the filter applies to")
	filtersCreateCmd.Flags().StringP("expression", "e", "", "CEL expression for the filter")
	filtersCreateCmd.Flags().StringP("scope", "s", "", "Scope for subsetting candidate filters")
	filtersCreateCmd.Flags().String("payload", "", "Filter payload as a JSON string")

	filtersUpdateCmd.Flags().StringP("expression", "e", "", "CEL expression for the filter")
	filtersUpdateCmd.Flags().StringP("scope", "s", "", "Scope for subsetting candidate filters")
	filtersUpdateCmd.Flags().String("payload", "", "Filter payload as a JSON string")

	filtersDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}
