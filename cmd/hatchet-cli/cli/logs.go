package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Search logs across runs",
	Long:  `Commands for searching and aggregating log lines across all runs in a tenant.`,
	Run:   func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var logsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List log lines",
	Long:  `List log lines across runs in the tenant. Without --output json, prints plain lines. With --output json, outputs raw JSON.`,
	Example: `  # Search for errors in the last hour
  hatchet logs list --search "timeout" --levels ERROR --since 2026-01-01T12:00:00Z

  # JSON output
  hatchet logs list -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)
		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)

		limit, _ := cmd.Flags().GetInt64("limit")
		params := &rest.V1TenantLogLineListParams{
			Limit:           &limit,
			Since:           logsTimeFlag(cmd, "since"),
			Until:           logsTimeFlag(cmd, "until"),
			Search:          logsStringFlag(cmd, "search"),
			Levels:          logsLevelsFlag(cmd),
			TaskExternalIds: logsUUIDsFlag(cmd, "task-ids"),
			WorkflowIds:     logsUUIDsFlag(cmd, "workflow-ids"),
			StepIds:         logsUUIDsFlag(cmd, "step-ids"),
		}
		if cmd.Flags().Changed("attempt") {
			attempt, _ := cmd.Flags().GetInt("attempt")
			params.Attempt = &attempt
		}
		if orderStr, _ := cmd.Flags().GetString("order"); orderStr != "" {
			order := rest.V1LogLineOrderByDirection(strings.ToUpper(orderStr))
			if order != rest.V1LogLineOrderByDirectionASC && order != rest.V1LogLineOrderByDirectionDESC {
				cli.Logger.Fatalf("invalid --order value %q (use asc or desc)", orderStr)
			}
			params.OrderByDirection = &order
		}

		resp, err := hatchetClient.API().V1TenantLogLineListWithResponse(ctx, tenantUUID, params)
		if err != nil {
			cli.Logger.Fatalf("failed to list logs: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d): %s", resp.StatusCode(), string(resp.Body))
		}

		if isJSON {
			printJSON(resp.JSON200)
			return
		}

		if resp.JSON200.Rows == nil || len(*resp.JSON200.Rows) == 0 {
			fmt.Println("No log lines found.")
			return
		}
		for _, line := range *resp.JSON200.Rows {
			level := "-"
			if line.Level != nil {
				level = string(*line.Level)
			}
			fmt.Printf("%s  %-5s  %s\n", line.CreatedAt.Format(time.RFC3339), level, line.Message)
		}
	},
}

var logsMetricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Get log point metrics",
	Long:  `Get per-minute log counts by level across runs in the tenant. Outputs raw JSON.`,
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)
		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)

		params := &rest.V1TenantLogLineGetPointMetricsParams{
			Since:           logsTimeFlag(cmd, "since"),
			Until:           logsTimeFlag(cmd, "until"),
			Search:          logsStringFlag(cmd, "search"),
			Levels:          logsLevelsFlag(cmd),
			TaskExternalIds: logsUUIDsFlag(cmd, "task-ids"),
			WorkflowIds:     logsUUIDsFlag(cmd, "workflow-ids"),
			StepIds:         logsUUIDsFlag(cmd, "step-ids"),
		}

		resp, err := hatchetClient.API().V1TenantLogLineGetPointMetricsWithResponse(ctx, tenantUUID, params)
		if err != nil {
			cli.Logger.Fatalf("failed to get log metrics: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d): %s", resp.StatusCode(), string(resp.Body))
		}

		printJSON(resp.JSON200)
	},
}

func logsStringFlag(cmd *cobra.Command, name string) *string {
	s, _ := cmd.Flags().GetString(name)
	if s == "" {
		return nil
	}
	return &s
}

func logsTimeFlag(cmd *cobra.Command, name string) *time.Time {
	s, _ := cmd.Flags().GetString(name)
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		cli.Logger.Fatalf("invalid --%s value (use RFC3339): %v", name, err)
	}
	return &t
}

func logsLevelsFlag(cmd *cobra.Command) *[]rest.V1LogLineLevel {
	vals, _ := cmd.Flags().GetStringSlice("levels")
	if len(vals) == 0 {
		return nil
	}
	levels := make([]rest.V1LogLineLevel, 0, len(vals))
	for _, v := range vals {
		levels = append(levels, rest.V1LogLineLevel(strings.ToUpper(v)))
	}
	return &levels
}

func logsUUIDsFlag(cmd *cobra.Command, name string) *[]uuid.UUID {
	vals, _ := cmd.Flags().GetStringSlice(name)
	if len(vals) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(vals))
	for _, v := range vals {
		u, err := uuid.Parse(v)
		if err != nil {
			cli.Logger.Fatalf("invalid --%s value %q: %v", name, v, err)
		}
		ids = append(ids, u)
	}
	return &ids
}

func addLogsFilterFlags(cmd *cobra.Command) {
	cmd.Flags().String("since", "", "Start time in RFC3339 format")
	cmd.Flags().String("until", "", "End time in RFC3339 format")
	cmd.Flags().String("search", "", "Full-text search query")
	cmd.Flags().StringSlice("levels", nil, "Log levels to include (DEBUG, INFO, WARN, ERROR)")
	cmd.Flags().StringSlice("task-ids", nil, "Task external IDs to filter by")
	cmd.Flags().StringSlice("workflow-ids", nil, "Workflow IDs to filter by")
	cmd.Flags().StringSlice("step-ids", nil, "Step IDs to filter by")
}

func init() {
	rootCmd.AddCommand(logsCmd)
	logsCmd.AddCommand(logsListCmd, logsMetricsCmd)

	logsCmd.PersistentFlags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	logsCmd.PersistentFlags().StringP("output", "o", "", "Output format: json")

	addLogsFilterFlags(logsListCmd)
	logsListCmd.Flags().Int64("limit", 100, "Number of results to return")
	logsListCmd.Flags().Int("attempt", 0, "Attempt number to filter for")
	logsListCmd.Flags().String("order", "", "Order direction: asc or desc")

	addLogsFilterFlags(logsMetricsCmd)
}
