package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/tui"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

var scheduledCmd = &cobra.Command{
	Use:     "scheduled",
	Aliases: []string{"schedule", "scheduled-runs", "schedules"},
	Short:   "Manage scheduled runs",
	Long:    `Commands for listing, inspecting, creating, and deleting scheduled workflow runs.`,
	Run:     func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var scheduledListCmd = &cobra.Command{
	Use:   "list",
	Short: "List scheduled runs",
	Long:  `List scheduled runs. Without --output json, launches the interactive TUI. With --output json, outputs raw JSON.`,
	Example: `  # Launch interactive TUI (default)
  hatchet scheduled list --profile local

  # JSON output
  hatchet scheduled list -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		selectedProfile, hatchetClient := clientFromCmd(cmd)

		if !isJSON {
			tuiM := newTUIModel(selectedProfile, hatchetClient)
			tuiM.currentViewType = ViewTypeScheduledRuns
			tuiM.currentView = tui.NewScheduledRunsView(tuiM.ctx)
			p := tea.NewProgram(tuiM, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				cli.Logger.Fatalf("error running TUI: %v", err)
			}
			return
		}

		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)
		limit, _ := cmd.Flags().GetInt64("limit")
		offset, _ := cmd.Flags().GetInt64("offset")

		params := &rest.WorkflowScheduledListParams{
			Limit:  &limit,
			Offset: &offset,
		}

		resp, err := hatchetClient.API().WorkflowScheduledListWithResponse(ctx, tenantUUID, params)
		if err != nil {
			cli.Logger.Fatalf("failed to list scheduled runs: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var scheduledGetCmd = &cobra.Command{
	Use:   "get <scheduled-run-id>",
	Short: "Get scheduled run details",
	Long:  `Get details about a scheduled run. Outputs raw JSON.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scheduledID := args[0]
		_, hatchetClient := clientFromCmd(cmd)

		scheduledUUID, err := uuid.Parse(scheduledID)
		if err != nil {
			cli.Logger.Fatalf("invalid scheduled run ID %q: %v", scheduledID, err)
		}

		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)
		resp, err := hatchetClient.API().WorkflowScheduledGetWithResponse(ctx, tenantUUID, scheduledUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to get scheduled run: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("scheduled run not found (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var scheduledCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a scheduled run",
	Long:  `Create a new scheduled workflow run. In --output json mode, all flags are required. Otherwise, launches an interactive form.`,
	Example: `  # Interactive mode
  hatchet scheduled create --profile local

  # JSON mode (all flags required)
  hatchet scheduled create --workflow my-workflow --trigger-at 2026-01-01T12:00:00Z --input '{}' -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)

		workflowStr, _ := cmd.Flags().GetString("workflow")
		triggerAtStr, _ := cmd.Flags().GetString("trigger-at")
		inputStr, _ := cmd.Flags().GetString("input")
		inputFile, _ := cmd.Flags().GetString("input-file")

		if !isJSON {
			// Interactive mode
			if workflowStr == "" {
				var wfChoice string
				form := huh.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("Workflow name or ID").
							Value(&workflowStr).
							Placeholder("my-workflow"),
					),
					huh.NewGroup(
						huh.NewInput().
							Title("Trigger at (RFC3339)").
							Value(&triggerAtStr).
							Placeholder("2026-01-01T12:00:00Z"),
					),
					huh.NewGroup(
						huh.NewInput().
							Title("Input JSON (optional)").
							Value(&inputStr).
							Placeholder("{}"),
					),
				).WithTheme(styles.HatchetTheme())
				_ = wfChoice
				if err := form.Run(); err != nil {
					cli.Logger.Fatalf("form cancelled: %v", err)
				}
			}
		}

		if workflowStr == "" {
			cli.Logger.Fatal("--workflow is required")
		}
		if triggerAtStr == "" {
			cli.Logger.Fatal("--trigger-at is required")
		}

		triggerAt, err := time.Parse(time.RFC3339, triggerAtStr)
		if err != nil {
			cli.Logger.Fatalf("invalid --trigger-at value (must be RFC3339): %v", err)
		}

		// Build input map
		inputData := map[string]interface{}{}
		if inputFile != "" {
			data, readErr := os.ReadFile(inputFile)
			if readErr != nil {
				cli.Logger.Fatalf("failed to read --input-file: %v", readErr)
			}
			if readErr = json.Unmarshal(data, &inputData); readErr != nil {
				cli.Logger.Fatalf("failed to parse --input-file as JSON: %v", readErr)
			}
		} else if inputStr != "" {
			if parseErr := json.Unmarshal([]byte(inputStr), &inputData); parseErr != nil {
				cli.Logger.Fatalf("failed to parse --input as JSON: %v", parseErr)
			}
		}

		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)

		// Resolve workflow name (the create endpoint uses name as path param, not UUID)
		workflowName, resolveErr := resolveWorkflowName(ctx, hatchetClient, workflowStr)
		if resolveErr != nil {
			cli.Logger.Fatalf("could not resolve workflow: %v", resolveErr)
		}

		resp, err := hatchetClient.API().ScheduledWorkflowRunCreateWithResponse(ctx, tenantUUID, workflowName, rest.ScheduledWorkflowRunCreateJSONRequestBody{
			TriggerAt:          triggerAt,
			Input:              inputData,
			AdditionalMetadata: map[string]interface{}{},
		})
		if err != nil {
			cli.Logger.Fatalf("failed to create scheduled run: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d): %s", resp.StatusCode(), string(resp.Body))
		}

		if isJSON {
			printJSON(resp.JSON200)
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Created scheduled run: %s", resp.JSON200.Metadata.Id)))
		}
	},
}

var scheduledDeleteCmd = &cobra.Command{
	Use:   "delete <scheduled-run-id>",
	Short: "Delete a scheduled run",
	Long:  `Delete a scheduled run by ID. Use --yes to skip confirmation.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scheduledID := args[0]
		isJSON := isJSONOutput(cmd)
		yes, _ := cmd.Flags().GetBool("yes")
		_, hatchetClient := clientFromCmd(cmd)

		scheduledUUID, err := uuid.Parse(scheduledID)
		if err != nil {
			cli.Logger.Fatalf("invalid scheduled run ID %q: %v", scheduledID, err)
		}

		if !isJSON && !yes {
			if !confirmAction(fmt.Sprintf("Delete scheduled run '%s'?", shortID(scheduledID))) {
				fmt.Println("Aborted.")
				return
			}
		}

		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)
		resp, err := hatchetClient.API().WorkflowScheduledDeleteWithResponse(ctx, tenantUUID, scheduledUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to delete scheduled run: %v", err)
		}
		if resp.StatusCode() >= 400 {
			cli.Logger.Fatalf("failed to delete scheduled run (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(map[string]interface{}{"deleted": true, "id": scheduledID})
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Deleted scheduled run: %s", scheduledID)))
		}
	},
}

func init() {
	rootCmd.AddCommand(scheduledCmd)
	scheduledCmd.AddCommand(scheduledListCmd, scheduledGetCmd, scheduledCreateCmd, scheduledDeleteCmd)

	scheduledCmd.PersistentFlags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	scheduledCmd.PersistentFlags().StringP("output", "o", "", "Output format: json (skips interactive TUI)")

	scheduledListCmd.Flags().Int64("limit", 50, "Number of results to return")
	scheduledListCmd.Flags().Int64("offset", 0, "Offset for pagination")

	scheduledCreateCmd.Flags().StringP("workflow", "w", "", "Workflow name or ID")
	scheduledCreateCmd.Flags().StringP("trigger-at", "t", "", "Trigger time in RFC3339 format (e.g. 2026-01-01T12:00:00Z)")
	scheduledCreateCmd.Flags().StringP("input", "i", "", "Input JSON string")
	scheduledCreateCmd.Flags().String("input-file", "", "Path to a JSON file for input")

	scheduledDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}
