package cli

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
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
	workflowsCmd.AddCommand(workflowsListCmd, workflowsGetCmd)

	workflowsCmd.PersistentFlags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	workflowsCmd.PersistentFlags().StringP("output", "o", "", "Output format: json (skips interactive TUI)")

	workflowsListCmd.Flags().StringP("search", "s", "", "Search workflows by name")
	workflowsListCmd.Flags().Int("limit", 50, "Number of results to return")
	workflowsListCmd.Flags().Int("offset", 0, "Offset for pagination")
}
