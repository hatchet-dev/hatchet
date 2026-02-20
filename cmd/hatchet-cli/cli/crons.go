package cli

import (
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/tui"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

var cronCmd = &cobra.Command{
	Use:     "cron",
	Aliases: []string{"crons", "cron-job", "cron-jobs"},
	Short:   "Manage cron jobs",
	Long:    `Commands for listing, inspecting, creating, enabling, disabling, and deleting cron jobs.`,
	Run:     func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var cronListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cron jobs",
	Long:  `List cron jobs. Without --output json, launches the interactive TUI. With --output json, outputs raw JSON.`,
	Example: `  # Launch interactive TUI (default)
  hatchet cron list --profile local

  # JSON output
  hatchet cron list -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		selectedProfile, hatchetClient := clientFromCmd(cmd)

		if !isJSON {
			tuiM := newTUIModel(selectedProfile, hatchetClient)
			tuiM.currentViewType = ViewTypeCronJobs
			tuiM.currentView = tui.NewCronJobsView(tuiM.ctx)
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

		resp, err := hatchetClient.API().CronWorkflowListWithResponse(ctx, tenantUUID, &rest.CronWorkflowListParams{
			Limit:  &limit,
			Offset: &offset,
		})
		if err != nil {
			cli.Logger.Fatalf("failed to list cron jobs: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var cronGetCmd = &cobra.Command{
	Use:   "get <cron-id>",
	Short: "Get cron job details",
	Long:  `Get details about a cron job. Outputs raw JSON.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cronID := args[0]
		_, hatchetClient := clientFromCmd(cmd)

		cronUUID, err := uuid.Parse(cronID)
		if err != nil {
			cli.Logger.Fatalf("invalid cron job ID %q: %v", cronID, err)
		}

		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)
		resp, err := hatchetClient.API().WorkflowCronGetWithResponse(ctx, tenantUUID, cronUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to get cron job: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("cron job not found (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var cronCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a cron job",
	Long:  `Create a new cron job. In --output json mode, all required flags must be set. Otherwise, launches an interactive form.`,
	Example: `  # Interactive mode
  hatchet cron create --profile local

  # JSON mode (required flags)
  hatchet cron create --workflow my-workflow --cron "0 * * * *" --name my-cron -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)
		ctx := cmd.Context()

		workflowStr, _ := cmd.Flags().GetString("workflow")
		cronExpr, _ := cmd.Flags().GetString("cron")
		cronName, _ := cmd.Flags().GetString("name")
		inputStr, _ := cmd.Flags().GetString("input")
		inputFile, _ := cmd.Flags().GetString("input-file")

		if !isJSON {
			// Interactive mode: show workflow selector first, then remaining fields
			if workflowStr == "" {
				workflowStr = promptSelectWorkflow(ctx, hatchetClient)
				if workflowStr == "" {
					cli.Logger.Fatal("no workflow selected")
				}
			}

			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Cron expression").
						Value(&cronExpr).
						Placeholder("0 * * * *"),
				),
				huh.NewGroup(
					huh.NewInput().
						Title("Cron job name (optional)").
						Value(&cronName).
						Placeholder("my-cron"),
				),
				huh.NewGroup(
					huh.NewInput().
						Title("Input JSON (optional)").
						Value(&inputStr).
						Placeholder("{}"),
				),
			).WithTheme(styles.HatchetTheme())
			if err := form.Run(); err != nil {
				cli.Logger.Fatalf("form cancelled: %v", err)
			}
		}

		if workflowStr == "" {
			cli.Logger.Fatal("--workflow is required")
		}
		if cronExpr == "" {
			cli.Logger.Fatal("--cron is required")
		}

		// Build input map
		inputData := map[string]interface{}{}
		if inputFile != "" {
			data, err := os.ReadFile(inputFile)
			if err != nil {
				cli.Logger.Fatalf("failed to read --input-file: %v", err)
			}
			if err := json.Unmarshal(data, &inputData); err != nil {
				cli.Logger.Fatalf("failed to parse --input-file as JSON: %v", err)
			}
		} else if inputStr != "" {
			if err := json.Unmarshal([]byte(inputStr), &inputData); err != nil {
				cli.Logger.Fatalf("failed to parse --input as JSON: %v", err)
			}
		}

		tenantUUID := clientTenantUUID(hatchetClient)

		// Resolve workflow name (the create endpoint uses name as path param, not UUID)
		workflowName, err := resolveWorkflowName(ctx, hatchetClient, workflowStr)
		if err != nil {
			cli.Logger.Fatalf("could not resolve workflow: %v", err)
		}

		if cronName == "" {
			cronName = workflowName + "-cron"
		}

		resp, err := hatchetClient.API().CronWorkflowTriggerCreateWithResponse(ctx, tenantUUID, workflowName, rest.CronWorkflowTriggerCreateJSONRequestBody{
			CronExpression:     cronExpr,
			CronName:           cronName,
			Input:              inputData,
			AdditionalMetadata: map[string]interface{}{},
		})
		if err != nil {
			cli.Logger.Fatalf("failed to create cron job: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d): %s", resp.StatusCode(), string(resp.Body))
		}

		if isJSON {
			printJSON(resp.JSON200)
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Created cron job: %s", resp.JSON200.Metadata.Id)))
		}
	},
}

var cronDeleteCmd = &cobra.Command{
	Use:   "delete [cron-id]",
	Short: "Delete a cron job",
	Long:  `Delete a cron job by ID. Omit the ID to pick from a list interactively. Use --yes to skip confirmation.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		yes, _ := cmd.Flags().GetBool("yes")
		_, hatchetClient := clientFromCmd(cmd)
		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)

		var cronID string
		if len(args) == 1 {
			cronID = args[0]
		} else if !isJSON {
			// No ID provided — show an interactive selector
			limit := int64(100)
			listResp, listErr := hatchetClient.API().CronWorkflowListWithResponse(ctx, tenantUUID, &rest.CronWorkflowListParams{
				Limit: &limit,
			})
			if listErr != nil {
				cli.Logger.Fatalf("failed to list cron jobs: %v", listErr)
			}
			if listResp.JSON200 == nil || listResp.JSON200.Rows == nil || len(*listResp.JSON200.Rows) == 0 {
				cli.Logger.Fatal("no cron jobs found")
			}

			var options []huh.Option[string]
			for _, cron := range *listResp.JSON200.Rows {
				name := cron.Metadata.Id
				if cron.Name != nil && *cron.Name != "" {
					name = *cron.Name
				}
				label := fmt.Sprintf("%s  (%s)", name, cron.Cron)
				options = append(options, huh.NewOption(label, cron.Metadata.Id))
			}

			height := len(options)
			if height > 10 {
				height = 10
			}
			form := huh.NewForm(huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select a cron job to delete").
					Options(options...).
					Height(height).
					Value(&cronID),
			)).WithTheme(styles.HatchetTheme())
			if formErr := form.Run(); formErr != nil {
				cli.Logger.Fatalf("selection cancelled: %v", formErr)
			}
		} else {
			cli.Logger.Fatal("cron job ID is required in JSON mode")
		}

		cronUUID, err := uuid.Parse(cronID)
		if err != nil {
			cli.Logger.Fatalf("invalid cron job ID %q: %v", cronID, err)
		}

		if !isJSON && !yes {
			if !confirmAction(fmt.Sprintf("Delete cron job '%s'?", shortID(cronID))) {
				fmt.Println("Aborted.")
				return
			}
		}

		resp, err := hatchetClient.API().WorkflowCronDeleteWithResponse(ctx, tenantUUID, cronUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to delete cron job: %v", err)
		}
		if resp.StatusCode() >= 400 {
			cli.Logger.Fatalf("failed to delete cron job (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(map[string]interface{}{"deleted": true, "id": cronID})
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Deleted cron job: %s", cronID)))
		}
	},
}

var cronEnableCmd = &cobra.Command{
	Use:   "enable <cron-id>",
	Short: "Enable a cron job",
	Long:  `Enable a cron job by ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cronID := args[0]
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)

		cronUUID, err := uuid.Parse(cronID)
		if err != nil {
			cli.Logger.Fatalf("invalid cron job ID %q: %v", cronID, err)
		}

		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)
		enabled := true
		resp, err := hatchetClient.API().WorkflowCronUpdateWithResponse(ctx, tenantUUID, cronUUID, rest.WorkflowCronUpdateJSONRequestBody{
			Enabled: &enabled,
		})
		if err != nil {
			cli.Logger.Fatalf("failed to enable cron job: %v", err)
		}
		if resp.StatusCode() >= 400 {
			cli.Logger.Fatalf("failed to enable cron job (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(map[string]interface{}{"enabled": true, "id": cronID})
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Enabled cron job: %s", cronID)))
		}
	},
}

var cronDisableCmd = &cobra.Command{
	Use:   "disable <cron-id>",
	Short: "Disable a cron job",
	Long:  `Disable a cron job by ID. Use --yes to skip confirmation.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cronID := args[0]
		isJSON := isJSONOutput(cmd)
		yes, _ := cmd.Flags().GetBool("yes")
		_, hatchetClient := clientFromCmd(cmd)

		cronUUID, err := uuid.Parse(cronID)
		if err != nil {
			cli.Logger.Fatalf("invalid cron job ID %q: %v", cronID, err)
		}

		if !isJSON && !yes {
			if !confirmAction(fmt.Sprintf("Disable cron job '%s'?", shortID(cronID))) {
				fmt.Println("Aborted.")
				return
			}
		}

		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)
		enabled := false
		resp, err := hatchetClient.API().WorkflowCronUpdateWithResponse(ctx, tenantUUID, cronUUID, rest.WorkflowCronUpdateJSONRequestBody{
			Enabled: &enabled,
		})
		if err != nil {
			cli.Logger.Fatalf("failed to disable cron job: %v", err)
		}
		if resp.StatusCode() >= 400 {
			cli.Logger.Fatalf("failed to disable cron job (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(map[string]interface{}{"enabled": false, "id": cronID})
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Disabled cron job: %s", cronID)))
		}
	},
}

func init() {
	rootCmd.AddCommand(cronCmd)
	cronCmd.AddCommand(cronListCmd, cronGetCmd, cronCreateCmd, cronDeleteCmd, cronEnableCmd, cronDisableCmd)

	cronCmd.PersistentFlags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	cronCmd.PersistentFlags().StringP("output", "o", "", "Output format: json (skips interactive TUI)")

	cronListCmd.Flags().Int64("limit", 50, "Number of results to return")
	cronListCmd.Flags().Int64("offset", 0, "Offset for pagination")

	cronCreateCmd.Flags().StringP("workflow", "w", "", "Workflow name or ID")
	cronCreateCmd.Flags().StringP("cron", "c", "", "Cron expression (e.g. '0 * * * *')")
	cronCreateCmd.Flags().StringP("name", "n", "", "Cron job name")
	cronCreateCmd.Flags().StringP("input", "i", "", "Input JSON string")
	cronCreateCmd.Flags().String("input-file", "", "Path to a JSON file for input")

	cronDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	cronDisableCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}
