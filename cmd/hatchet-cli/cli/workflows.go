package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/tui"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

var workflowsCmd = &cobra.Command{
	Use:     "workflows",
	Aliases: []string{"workflow"},
	Short:   "Manage workflows",
	Long:    `Commands for listing and inspecting workflows.`,
	Run:     func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var workflowsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workflows",
	Long:  `List workflows. Without --output json, launches the interactive TUI. With --output json, outputs raw JSON.`,
	Example: `  # Launch interactive TUI (default)
  hatchet workflows list --profile local

  # JSON output
  hatchet workflows list -o json
  hatchet workflows list -o json --search my-workflow --limit 100`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		selectedProfile, hatchetClient := clientFromCmd(cmd)

		if !isJSON {
			tuiM := newTUIModel(selectedProfile, hatchetClient)
			tuiM.currentViewType = ViewTypeWorkflows
			tuiM.currentView = tui.NewWorkflowsView(tuiM.ctx)
			p := tea.NewProgram(tuiM, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				cli.Logger.Fatalf("error running TUI: %v", err)
			}
			return
		}

		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)
		searchStr, _ := cmd.Flags().GetString("search")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		params := &rest.WorkflowListParams{
			Limit:  &limit,
			Offset: &offset,
		}
		if searchStr != "" {
			params.Name = &searchStr
		}

		resp, err := hatchetClient.API().WorkflowListWithResponse(ctx, tenantUUID, params)
		if err != nil {
			cli.Logger.Fatalf("failed to list workflows: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var workflowsGetCmd = &cobra.Command{
	Use:   "get <workflow-id>",
	Short: "Get workflow details",
	Long:  `Get details about a workflow. Without --output json, launches the TUI navigated to the workflow. With --output json, outputs raw JSON.`,
	Args:  cobra.ExactArgs(1),
	Example: `  # Launch TUI for a specific workflow
  hatchet workflows get <workflow-id> --profile local

  # JSON output
  hatchet workflows get <workflow-id> -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		workflowID := args[0]
		isJSON := isJSONOutput(cmd)
		selectedProfile, hatchetClient := clientFromCmd(cmd)

		if !isJSON {
			base := newTUIModel(selectedProfile, hatchetClient)
			base.currentViewType = ViewTypeWorkflows
			base.currentView = tui.NewWorkflowsView(base.ctx)
			model := tuiModelWithInitialWorkflow{
				tuiModel:          base,
				initialWorkflowID: workflowID,
			}
			p := tea.NewProgram(model, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				cli.Logger.Fatalf("error running TUI: %v", err)
			}
			return
		}

		workflowUUID, err := uuid.Parse(workflowID)
		if err != nil {
			cli.Logger.Fatalf("invalid workflow ID %q: %v", workflowID, err)
		}

		ctx := cmd.Context()
		resp, err := hatchetClient.API().WorkflowGetWithResponse(ctx, workflowUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to get workflow: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("workflow not found (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var workflowsDeleteCmd = &cobra.Command{
	Use:   "delete <workflow>",
	Short: "Delete a workflow",
	Long:  `Delete a workflow by name or ID. This deletes the workflow and all of its data (runs, schedules, crons).`,
	Args:  cobra.ExactArgs(1),
	Example: `  # Delete a workflow by name
  hatchet workflows delete my-workflow --profile local

  # Delete without confirmation
  hatchet workflows delete my-workflow -y`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		yes, _ := cmd.Flags().GetBool("yes")
		_, hatchetClient := clientFromCmd(cmd)

		ctx := cmd.Context()
		workflowUUID, err := resolveWorkflowID(ctx, hatchetClient, args[0])
		if err != nil {
			cli.Logger.Fatalf("could not resolve workflow: %v", err)
		}

		if !isJSON && !yes {
			if !confirmAction(fmt.Sprintf("Delete workflow '%s' and all of its data? This cannot be undone.", args[0])) {
				fmt.Println("Aborted.")
				return
			}
		}

		resp, err := hatchetClient.API().WorkflowDeleteWithResponse(ctx, workflowUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to delete workflow: %v", err)
		}
		if resp.StatusCode() >= 300 {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(map[string]any{"deleted": workflowUUID.String()})
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Deleted workflow '%s'", args[0])))
		}
	},
}

func setWorkflowPaused(cmd *cobra.Command, workflowArg string, paused bool) {
	isJSON := isJSONOutput(cmd)
	_, hatchetClient := clientFromCmd(cmd)

	ctx := cmd.Context()
	workflowUUID, err := resolveWorkflowID(ctx, hatchetClient, workflowArg)
	if err != nil {
		cli.Logger.Fatalf("could not resolve workflow: %v", err)
	}

	resp, err := hatchetClient.API().WorkflowUpdateWithResponse(ctx, workflowUUID, rest.WorkflowUpdateRequest{
		IsPaused: &paused,
	})
	if err != nil {
		cli.Logger.Fatalf("failed to update workflow: %v", err)
	}
	if resp.JSON200 == nil {
		cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
	}

	if isJSON {
		printJSON(resp.JSON200)
		return
	}
	verb := "Unpaused"
	if paused {
		verb = "Paused"
	}
	fmt.Println(styles.SuccessMessage(fmt.Sprintf("%s workflow '%s'", verb, workflowArg)))
}

var workflowsPauseCmd = &cobra.Command{
	Use:     "pause <workflow>",
	Short:   "Pause a workflow",
	Long:    `Pause a workflow by name or ID. Paused workflows do not schedule new runs.`,
	Args:    cobra.ExactArgs(1),
	Example: `  hatchet workflows pause my-workflow --profile local`,
	Run: func(cmd *cobra.Command, args []string) {
		setWorkflowPaused(cmd, args[0], true)
	},
}

var workflowsUnpauseCmd = &cobra.Command{
	Use:     "unpause <workflow>",
	Short:   "Unpause a workflow",
	Long:    `Unpause a workflow by name or ID.`,
	Args:    cobra.ExactArgs(1),
	Example: `  hatchet workflows unpause my-workflow --profile local`,
	Run: func(cmd *cobra.Command, args []string) {
		setWorkflowPaused(cmd, args[0], false)
	},
}

var workflowsMetricsCmd = &cobra.Command{
	Use:   "metrics <workflow>",
	Short: "Get workflow metrics",
	Long:  `Get metrics for a workflow by name or ID. Outputs JSON.`,
	Args:  cobra.ExactArgs(1),
	Example: `  hatchet workflows metrics my-workflow --profile local
  hatchet workflows metrics my-workflow --status RUNNING --group-key my-key`,
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)

		ctx := cmd.Context()
		workflowUUID, err := resolveWorkflowID(ctx, hatchetClient, args[0])
		if err != nil {
			cli.Logger.Fatalf("could not resolve workflow: %v", err)
		}

		params := &rest.WorkflowGetMetricsParams{}
		statusStr, _ := cmd.Flags().GetString("status")
		if statusStr != "" {
			status := rest.WorkflowRunStatus(strings.ToUpper(statusStr))
			params.Status = &status
		}
		groupKey, _ := cmd.Flags().GetString("group-key")
		if groupKey != "" {
			params.GroupKey = &groupKey
		}

		resp, err := hatchetClient.API().WorkflowGetMetricsWithResponse(ctx, workflowUUID, params)
		if err != nil {
			cli.Logger.Fatalf("failed to get workflow metrics: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var workflowsWorkersCountCmd = &cobra.Command{
	Use:   "workers-count <workflow>",
	Short: "Get worker slot counts for a workflow",
	Long:  `Get the free and max worker slot counts for a workflow by name or ID.`,
	Args:  cobra.ExactArgs(1),
	Example: `  hatchet workflows workers-count my-workflow --profile local
  hatchet workflows workers-count my-workflow -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)

		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)
		workflowUUID, err := resolveWorkflowID(ctx, hatchetClient, args[0])
		if err != nil {
			cli.Logger.Fatalf("could not resolve workflow: %v", err)
		}

		resp, err := hatchetClient.API().WorkflowGetWorkersCountWithResponse(ctx, tenantUUID, workflowUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to get workers count: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(resp.JSON200)
			return
		}
		freeSlots, maxSlots := 0, 0
		if resp.JSON200.FreeSlotCount != nil {
			freeSlots = *resp.JSON200.FreeSlotCount
		}
		if resp.JSON200.MaxSlotCount != nil {
			maxSlots = *resp.JSON200.MaxSlotCount
		}
		fmt.Printf("Free slots: %d\nMax slots:  %d\n", freeSlots, maxSlots)
	},
}

var workflowsVersionCmd = &cobra.Command{
	Use:   "version <workflow>",
	Short: "Get a workflow version",
	Long:  `Get a workflow version by workflow name or ID. Fetches the latest version unless --version is provided. Outputs JSON.`,
	Args:  cobra.ExactArgs(1),
	Example: `  # Latest version
  hatchet workflows version my-workflow --profile local

  # Specific version
  hatchet workflows version my-workflow --version 8ff4f149-099e-4c16-a8d1-0535f8c79b83`,
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)

		ctx := cmd.Context()
		workflowUUID, err := resolveWorkflowID(ctx, hatchetClient, args[0])
		if err != nil {
			cli.Logger.Fatalf("could not resolve workflow: %v", err)
		}

		params := &rest.WorkflowVersionGetParams{}
		versionStr, _ := cmd.Flags().GetString("version")
		if versionStr != "" {
			versionUUID, err := uuid.Parse(versionStr)
			if err != nil {
				cli.Logger.Fatalf("invalid version ID %q: %v", versionStr, err)
			}
			params.Version = &versionUUID
		}

		resp, err := hatchetClient.API().WorkflowVersionGetWithResponse(ctx, workflowUUID, params)
		if err != nil {
			cli.Logger.Fatalf("failed to get workflow version: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("workflow version not found (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

// tuiModelWithInitialWorkflow wraps tuiModel to navigate to a specific workflow on init
type tuiModelWithInitialWorkflow struct {
	initialWorkflowID string
	tuiModel
}

func (m tuiModelWithInitialWorkflow) Init() tea.Cmd {
	return tea.Batch(
		m.tuiModel.Init(),
		func() tea.Msg { return tui.NavigateToWorkflowMsg{WorkflowID: m.initialWorkflowID} },
	)
}

func init() {
	rootCmd.AddCommand(workflowsCmd)
	workflowsCmd.AddCommand(workflowsListCmd, workflowsGetCmd, workflowsDeleteCmd, workflowsPauseCmd, workflowsUnpauseCmd, workflowsMetricsCmd, workflowsWorkersCountCmd, workflowsVersionCmd)

	workflowsCmd.PersistentFlags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	workflowsCmd.PersistentFlags().StringP("output", "o", "", "Output format: json (skips interactive TUI)")

	workflowsListCmd.Flags().StringP("search", "s", "", "Search workflows by name")
	workflowsListCmd.Flags().Int("limit", 50, "Number of results to return")
	workflowsListCmd.Flags().Int("offset", 0, "Offset for pagination")

	workflowsDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	workflowsMetricsCmd.Flags().String("status", "", "Filter metrics by workflow run status")
	workflowsMetricsCmd.Flags().String("group-key", "", "Filter metrics by concurrency group key")

	workflowsVersionCmd.Flags().String("version", "", "Workflow version ID (default: latest)")
}
