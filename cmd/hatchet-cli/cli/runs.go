package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

var runsCmd = &cobra.Command{
	Use:     "runs",
	Aliases: []string{"run"},
	Short:   "Manage runs",
	Long:    `Commands for listing, inspecting, cancelling, replaying, and viewing logs/events for runs.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var runsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List runs",
	Long:  `List runs. Without --output json, launches the interactive TUI. With --output json, outputs raw JSON to stdout.`,
	Example: `  # Launch interactive TUI (default)
  hatchet runs list --profile local

  # JSON output with filters
  hatchet runs list -o json --since 24h --status FAILED
  hatchet runs list -o json --since 1h --workflow my-workflow --limit 100`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		selectedProfile, hatchetClient := clientFromCmd(cmd)

		if !isJSON {
			model := newTUIModel(selectedProfile, hatchetClient)
			p := tea.NewProgram(model, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				cli.Logger.Fatalf("error running TUI: %v", err)
			}
			return
		}

		// JSON mode: parse flags and call API
		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)

		sinceStr, _ := cmd.Flags().GetString("since")
		sinceTime, err := parseSinceDuration(sinceStr)
		if err != nil {
			cli.Logger.Fatalf("invalid --since value: %v", err)
		}

		untilStr, _ := cmd.Flags().GetString("until")
		var untilPtr *time.Time
		if untilStr != "" {
			var until time.Time
			until, err = parseSinceDuration(untilStr)
			if err != nil {
				cli.Logger.Fatalf("invalid --until value: %v", err)
			}
			untilPtr = &until
		}

		workflowStr, _ := cmd.Flags().GetString("workflow")
		var workflowUUIDs *[]openapi_types.UUID
		if workflowStr != "" {
			var wfUUID openapi_types.UUID
			wfUUID, err = resolveWorkflowID(ctx, hatchetClient, workflowStr)
			if err != nil {
				cli.Logger.Fatalf("could not resolve workflow: %v", err)
			}
			workflowUUIDs = &[]openapi_types.UUID{wfUUID}
		}

		statusStrs, _ := cmd.Flags().GetStringSlice("status")
		statuses, err := parseStatuses(statusStrs)
		if err != nil {
			cli.Logger.Fatalf("%v", err)
		}
		var statusesPtr *[]rest.V1TaskStatus
		if len(statuses) > 0 {
			statusesPtr = &statuses
		}

		limit, _ := cmd.Flags().GetInt64("limit")
		offset, _ := cmd.Flags().GetInt64("offset")
		onlyTasks, _ := cmd.Flags().GetBool("only-tasks")

		params := &rest.V1WorkflowRunListParams{
			Since:     sinceTime,
			Until:     untilPtr,
			Statuses:  statusesPtr,
			Limit:     &limit,
			Offset:    &offset,
			OnlyTasks: onlyTasks,
		}
		if workflowUUIDs != nil {
			params.WorkflowIds = workflowUUIDs
		}

		resp, err := hatchetClient.API().V1WorkflowRunListWithResponse(ctx, tenantUUID, params)
		if err != nil {
			cli.Logger.Fatalf("failed to list runs: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var runsGetCmd = &cobra.Command{
	Use:   "get <run-id>",
	Short: "Inspect a run",
	Long:  `Get details about a run. Without --output json, launches the interactive TUI navigated to the run. With --output json, outputs raw JSON.`,
	Args:  cobra.ExactArgs(1),
	Example: `  # Launch TUI for a specific run
  hatchet runs get 8ff4f149-099e-4c16-a8d1-0535f8c79b83 --profile local

  # JSON output
  hatchet runs get 8ff4f149-099e-4c16-a8d1-0535f8c79b83 -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		runID := args[0]
		isJSON := isJSONOutput(cmd)
		selectedProfile, hatchetClient := clientFromCmd(cmd)

		if !isJSON {
			model := tuiModelWithInitialRun{
				tuiModel:     newTUIModel(selectedProfile, hatchetClient),
				initialRunID: runID,
			}
			p := tea.NewProgram(model, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				cli.Logger.Fatalf("error running TUI: %v", err)
			}
			return
		}

		// JSON mode
		ctx := cmd.Context()
		runUUID, err := uuid.Parse(runID)
		if err != nil {
			cli.Logger.Fatalf("invalid run ID %q: %v", runID, err)
		}

		resp, err := hatchetClient.API().V1WorkflowRunGetWithResponse(ctx, runUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to get run: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("run not found (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var runsCancelCmd = &cobra.Command{
	Use:   "cancel [run-id]",
	Short: "Cancel a run or bulk cancel by filter",
	Long: `Cancel a specific run by ID, or cancel multiple runs matching filter criteria.

If a run ID is provided, cancels that specific run (task or DAG).
If no run ID is provided, requires --since flag and cancels runs matching the filter.`,
	Args: cobra.MaximumNArgs(1),
	Example: `  # Cancel a specific run
  hatchet runs cancel 8ff4f149-099e-4c16-a8d1-0535f8c79b83 --profile local

  # Bulk cancel by filter
  hatchet runs cancel --since 1h --status FAILED --profile local

  # Bulk cancel JSON mode (no confirmation)
  hatchet runs cancel --since 24h --workflow my-workflow -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		yes, _ := cmd.Flags().GetBool("yes")
		_, hatchetClient := clientFromCmd(cmd)

		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)

		if len(args) > 0 {
			// Single run cancel
			runID := args[0]
			runUUID, err := uuid.Parse(runID)
			if err != nil {
				cli.Logger.Fatalf("invalid run ID %q: %v", runID, err)
			}

			if !isJSON && !yes {
				// Try to get run name for confirmation
				runName := shortID(runID)
				runResp, _ := hatchetClient.API().V1WorkflowRunGetWithResponse(ctx, runUUID)
				if runResp != nil && runResp.JSON200 != nil {
					runName = runResp.JSON200.Run.DisplayName
				}
				if !confirmAction(fmt.Sprintf("Cancel run '%s'?", runName)) {
					fmt.Println("Aborted.")
					return
				}
			}

			externalID := runUUID
			resp, err := hatchetClient.API().V1TaskCancelWithResponse(ctx, tenantUUID, rest.V1CancelTaskRequest{
				ExternalIds: &[]openapi_types.UUID{externalID},
			})
			if err != nil {
				cli.Logger.Fatalf("failed to cancel run: %v", err)
			}
			if resp.JSON200 == nil {
				cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
			}

			if isJSON {
				printJSON(resp.JSON200)
			} else {
				count := 0
				if resp.JSON200.Ids != nil {
					count = len(*resp.JSON200.Ids)
				}
				fmt.Println(styles.SuccessMessage(fmt.Sprintf("Cancelled %d run(s)", count)))
			}
			return
		}

		// Bulk cancel via filters
		sinceStr, _ := cmd.Flags().GetString("since")
		if sinceStr == "" {
			cli.Logger.Fatal("--since is required for bulk cancel (or provide a run ID as argument)")
		}

		filter, listParams, err := buildFilterAndParams(ctx, cmd, hatchetClient)
		if err != nil {
			cli.Logger.Fatalf("%v", err)
		}

		if !isJSON && !yes {
			count := countMatchingRuns(ctx, hatchetClient, tenantUUID, listParams)
			if count == 0 {
				fmt.Println(styles.Muted.Render("No runs match your filters."))
				return
			}
			if !confirmAction(fmt.Sprintf("This will cancel %d runs matching your filters. Continue?", count)) {
				fmt.Println("Aborted.")
				return
			}
		}

		resp, err := hatchetClient.API().V1TaskCancelWithResponse(ctx, tenantUUID, rest.V1CancelTaskRequest{
			Filter: filter,
		})
		if err != nil {
			cli.Logger.Fatalf("failed to cancel runs: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(resp.JSON200)
		} else {
			count := 0
			if resp.JSON200.Ids != nil {
				count = len(*resp.JSON200.Ids)
			}
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Cancelled %d run(s)", count)))
		}
	},
}

var runsReplayCmd = &cobra.Command{
	Use:   "replay [run-id]",
	Short: "Replay a run or bulk replay by filter",
	Long: `Replay a specific run by ID, or replay multiple runs matching filter criteria.

If a run ID is provided, replays that specific run (task or DAG).
If no run ID is provided, requires --since flag and replays runs matching the filter.`,
	Args: cobra.MaximumNArgs(1),
	Example: `  # Replay a specific run
  hatchet runs replay 8ff4f149-099e-4c16-a8d1-0535f8c79b83 --profile local

  # Bulk replay by filter
  hatchet runs replay --since 1h --status FAILED --profile local

  # Bulk replay JSON mode (no confirmation)
  hatchet runs replay --since 24h -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		yes, _ := cmd.Flags().GetBool("yes")
		_, hatchetClient := clientFromCmd(cmd)

		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)

		if len(args) > 0 {
			// Single run replay
			runID := args[0]
			runUUID, err := uuid.Parse(runID)
			if err != nil {
				cli.Logger.Fatalf("invalid run ID %q: %v", runID, err)
			}

			if !isJSON && !yes {
				runName := shortID(runID)
				runResp, _ := hatchetClient.API().V1WorkflowRunGetWithResponse(ctx, runUUID)
				if runResp != nil && runResp.JSON200 != nil {
					runName = runResp.JSON200.Run.DisplayName
				}
				if !confirmAction(fmt.Sprintf("Replay run '%s'?", runName)) {
					fmt.Println("Aborted.")
					return
				}
			}

			externalID := runUUID
			resp, err := hatchetClient.API().V1TaskReplayWithResponse(ctx, tenantUUID, rest.V1ReplayTaskRequest{
				ExternalIds: &[]openapi_types.UUID{externalID},
			})
			if err != nil {
				cli.Logger.Fatalf("failed to replay run: %v", err)
			}
			if resp.JSON200 == nil {
				cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
			}

			if isJSON {
				printJSON(resp.JSON200)
			} else {
				count := 0
				if resp.JSON200.Ids != nil {
					count = len(*resp.JSON200.Ids)
				}
				fmt.Println(styles.SuccessMessage(fmt.Sprintf("Replayed %d run(s)", count)))
			}
			return
		}

		// Bulk replay via filters
		sinceStr, _ := cmd.Flags().GetString("since")
		if sinceStr == "" {
			cli.Logger.Fatal("--since is required for bulk replay (or provide a run ID as argument)")
		}

		filter, listParams, err := buildFilterAndParams(ctx, cmd, hatchetClient)
		if err != nil {
			cli.Logger.Fatalf("%v", err)
		}

		if !isJSON && !yes {
			count := countMatchingRuns(ctx, hatchetClient, tenantUUID, listParams)
			if count == 0 {
				fmt.Println(styles.Muted.Render("No runs match your filters."))
				return
			}
			if !confirmAction(fmt.Sprintf("This will replay %d runs matching your filters. Continue?", count)) {
				fmt.Println("Aborted.")
				return
			}
		}

		resp, err := hatchetClient.API().V1TaskReplayWithResponse(ctx, tenantUUID, rest.V1ReplayTaskRequest{
			Filter: filter,
		})
		if err != nil {
			cli.Logger.Fatalf("failed to replay runs: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(resp.JSON200)
		} else {
			count := 0
			if resp.JSON200.Ids != nil {
				count = len(*resp.JSON200.Ids)
			}
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Replayed %d run(s)", count)))
		}
	},
}

var runsLogsCmd = &cobra.Command{
	Use:   "logs <run-id>",
	Short: "Print logs from a run",
	Long:  `Fetch and print logs from a run to stdout, sorted by timestamp. Works for both task runs and DAG workflow runs.`,
	Args:  cobra.ExactArgs(1),
	Example: `  # Print logs to stdout
  hatchet runs logs 8ff4f149-099e-4c16-a8d1-0535f8c79b83 --profile local

  # Show only the last 50 lines
  hatchet runs logs 8ff4f149-099e-4c16-a8d1-0535f8c79b83 --tail 50

  # Show logs from the last 5 minutes
  hatchet runs logs 8ff4f149-099e-4c16-a8d1-0535f8c79b83 --since 5m

  # Follow (poll for new logs)
  hatchet runs logs 8ff4f149-099e-4c16-a8d1-0535f8c79b83 -f

  # JSON output
  hatchet runs logs 8ff4f149-099e-4c16-a8d1-0535f8c79b83 -o json | jq .`,
	Run: func(cmd *cobra.Command, args []string) {
		runID := args[0]
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)

		ctx := cmd.Context()
		runUUID, err := uuid.Parse(runID)
		if err != nil {
			cli.Logger.Fatalf("invalid run ID %q: %v", runID, err)
		}

		tail, _ := cmd.Flags().GetInt64("tail")
		sinceStr, _ := cmd.Flags().GetString("since")
		follow, _ := cmd.Flags().GetBool("follow")

		var sinceTime *time.Time
		if sinceStr != "" {
			var t time.Time
			t, err = parseSinceDuration(sinceStr)
			if err != nil {
				cli.Logger.Fatalf("invalid --since value: %v", err)
			}
			sinceTime = &t
		}

		type logEntry struct {
			Task      string    `json:"task"`
			Timestamp time.Time `json:"timestamp"`
			Level     string    `json:"level,omitempty"`
			Message   string    `json:"message"`
		}

		// fetchLogs fetches logs for the run. pollLimit is set during follow-mode polling
		// to avoid dropping logs between polls (nil means apply --tail logic instead).
		fetchLogs := func(since *time.Time, pollLimit *int64) ([]logEntry, error) {
			// Try to fetch as a workflow run (DAG)
			runResp, fetchErr := hatchetClient.API().V1WorkflowRunGetWithResponse(ctx, runUUID)
			if fetchErr == nil && runResp.JSON200 != nil {
				// DAG run: fetch all logs per task then apply --tail to the merged list,
				// rather than limiting per-task (which would give wrong "newest N overall").
				params := &rest.V1LogLineListParams{Since: since}
				if pollLimit != nil {
					params.Limit = pollLimit
				}
				var allLogs []logEntry
				for _, task := range runResp.JSON200.Tasks {
					taskUUID, parseErr := uuid.Parse(task.Metadata.Id)
					if parseErr != nil {
						continue
					}
					logsResp, logsErr := hatchetClient.API().V1LogLineListWithResponse(
						ctx,
						taskUUID,
						params,
					)
					if logsErr != nil || logsResp.JSON200 == nil || logsResp.JSON200.Rows == nil {
						continue
					}
					for _, logLine := range *logsResp.JSON200.Rows {
						level := ""
						if logLine.Level != nil {
							level = string(*logLine.Level)
						}
						allLogs = append(allLogs, logEntry{
							Task:      task.DisplayName,
							Timestamp: logLine.CreatedAt,
							Level:     level,
							Message:   logLine.Message,
						})
					}
				}
				sort.Slice(allLogs, func(i, j int) bool {
					return allLogs[i].Timestamp.Before(allLogs[j].Timestamp)
				})
				// Apply --tail to merged result after sorting
				if tail > 0 && pollLimit == nil && int64(len(allLogs)) > tail {
					allLogs = allLogs[int64(len(allLogs))-tail:]
				}
				return allLogs, nil
			}

			// Fall back: treat run ID as a task ID directly (single task run).
			// Use DESC ordering when --tail is set so the API returns the newest N lines,
			// then reverse for chronological display.
			var params *rest.V1LogLineListParams
			if tail > 0 && pollLimit == nil {
				descDir := rest.V1LogLineOrderByDirectionDESC
				params = &rest.V1LogLineListParams{
					Since:            since,
					Limit:            &tail,
					OrderByDirection: &descDir,
				}
			} else {
				params = &rest.V1LogLineListParams{Since: since}
				if pollLimit != nil {
					params.Limit = pollLimit
				}
			}
			logsResp, logsErr := hatchetClient.API().V1LogLineListWithResponse(
				ctx,
				runUUID,
				params,
			)
			if logsErr != nil {
				return nil, fmt.Errorf("failed to fetch logs: %w", logsErr)
			}
			if logsResp.JSON200 == nil || logsResp.JSON200.Rows == nil {
				return nil, nil
			}
			var allLogs []logEntry
			for _, logLine := range *logsResp.JSON200.Rows {
				level := ""
				if logLine.Level != nil {
					level = string(*logLine.Level)
				}
				allLogs = append(allLogs, logEntry{
					Task:      "",
					Timestamp: logLine.CreatedAt,
					Level:     level,
					Message:   logLine.Message,
				})
			}
			// Reverse DESC results back to chronological order
			if tail > 0 && pollLimit == nil {
				for i, j := 0, len(allLogs)-1; i < j; i, j = i+1, j-1 {
					allLogs[i], allLogs[j] = allLogs[j], allLogs[i]
				}
			}
			return allLogs, nil
		}

		printLogEntries := func(logs []logEntry) {
			for _, l := range logs {
				ts := l.Timestamp.Format("15:04:05.000")
				if l.Task != "" {
					if l.Level != "" {
						fmt.Printf("[%s] %s %-8s %s\n", l.Task, ts, l.Level, l.Message)
					} else {
						fmt.Printf("[%s] %s %s\n", l.Task, ts, l.Message)
					}
				} else {
					if l.Level != "" {
						fmt.Printf("%s %-8s %s\n", ts, l.Level, l.Message)
					} else {
						fmt.Printf("%s %s\n", ts, l.Message)
					}
				}
			}
		}

		// Initial fetch
		allLogs, err := fetchLogs(sinceTime, nil)
		if err != nil {
			cli.Logger.Fatalf("%v", err)
		}

		if isJSON {
			printJSON(allLogs)
			return
		}

		if len(allLogs) == 0 && !follow {
			fmt.Println("No logs found for this run.")
			return
		}

		printLogEntries(allLogs)

		if !follow {
			return
		}

		// Determine the timestamp to use as the "since" cursor for polling
		var lastTimestamp time.Time
		if len(allLogs) > 0 {
			lastTimestamp = allLogs[len(allLogs)-1].Timestamp
		} else {
			lastTimestamp = time.Now()
		}

		// Poll for new logs
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigCh)

		for {
			select {
			case <-ctx.Done():
				return
			case <-sigCh:
				return
			case <-ticker.C:
				// Add a small buffer to avoid re-fetching the last line.
				// Use a high explicit limit so fast-producing tasks don't drop logs.
				pollSince := lastTimestamp.Add(time.Millisecond)
				highLimit := int64(10000)
				newLogs, err := fetchLogs(&pollSince, &highLimit)
				if err != nil || len(newLogs) == 0 {
					continue
				}
				printLogEntries(newLogs)
				lastTimestamp = newLogs[len(newLogs)-1].Timestamp
			}
		}
	},
}

var runsListChildrenCmd = &cobra.Command{
	Use:     "list-children <run-id>",
	Aliases: []string{"children"},
	Short:   "List children of a run",
	Long: `List the children of a run.

If the run is a DAG, lists the constituent tasks that belong to it.
If the run is a task, lists spawned child workflow runs.`,
	Args: cobra.ExactArgs(1),
	Example: `  # List children of a DAG run
  hatchet runs list-children 8ff4f149-099e-4c16-a8d1-0535f8c79b83 --profile local

  # List spawned child runs of a task
  hatchet runs list-children abc12345-... --profile local

  # JSON output
  hatchet runs list-children 8ff4f149-099e-4c16-a8d1-0535f8c79b83 -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		runID := args[0]
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)

		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)

		runUUID, err := uuid.Parse(runID)
		if err != nil {
			cli.Logger.Fatalf("invalid run ID %q: %v", runID, err)
		}

		limit, _ := cmd.Flags().GetInt64("limit")
		offset, _ := cmd.Flags().GetInt64("offset")

		// Detect run type: try V1TaskGet first. If it succeeds the ID is a task;
		// if it fails or returns nothing, fall through to treat it as a DAG run.
		taskResp, taskErr := hatchetClient.API().V1TaskGetWithResponse(ctx, runUUID, &rest.V1TaskGetParams{})
		if taskErr == nil && taskResp.JSON200 != nil {
			// It's a task — list spawned child workflow runs using the task's external ID.
			taskExternalID := taskResp.JSON200.TaskExternalId
			childRunsResp, childErr := hatchetClient.API().V1WorkflowRunListWithResponse(ctx, tenantUUID, &rest.V1WorkflowRunListParams{
				ParentTaskExternalId: &taskExternalID,
				Limit:                &limit,
				Offset:               &offset,
			})
			if childErr != nil {
				cli.Logger.Fatalf("failed to list child runs: %v", childErr)
			}
			if childRunsResp.JSON200 == nil {
				cli.Logger.Fatalf("unexpected response from API (status %d): %s", childRunsResp.StatusCode(), string(childRunsResp.Body))
			}

			if isJSON {
				printJSON(childRunsResp.JSON200)
				return
			}

			rows := childRunsResp.JSON200.Rows
			if len(rows) == 0 {
				fmt.Println("No child runs found.")
				return
			}
			fmt.Printf("%-38s  %-25s  %-12s  %s\n", "Run ID", "Name", "Status", "Duration")
			for _, run := range rows {
				dur := ""
				if run.Duration != nil {
					dur = fmt.Sprintf("%dms", *run.Duration)
				}
				fmt.Printf("%-38s  %-25s  %-12s  %s\n",
					run.Metadata.Id,
					run.DisplayName,
					string(run.Status),
					dur,
				)
			}
			return
		}

		// Not a task (or task fetch failed) — treat as a DAG run and list its constituent tasks.
		dagTasksResp, err := hatchetClient.API().V1DagListTasksWithResponse(ctx, &rest.V1DagListTasksParams{
			DagIds: []openapi_types.UUID{runUUID},
			Tenant: tenantUUID,
		})
		if err != nil {
			cli.Logger.Fatalf("failed to list children: %v", err)
		}
		if dagTasksResp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d): %s", dagTasksResp.StatusCode(), string(dagTasksResp.Body))
		}

		var tasks []rest.V1TaskSummary
		for _, dag := range *dagTasksResp.JSON200 {
			if dag.Children != nil {
				tasks = append(tasks, *dag.Children...)
			}
		}

		if isJSON {
			printJSON(map[string]any{"rows": tasks})
			return
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found for this DAG run.")
			return
		}
		fmt.Printf("%-38s  %-25s  %-12s  %s\n", "Task ID", "Name", "Status", "Duration")
		for _, task := range tasks {
			dur := ""
			if task.Duration != nil {
				dur = fmt.Sprintf("%dms", *task.Duration)
			}
			fmt.Printf("%-38s  %-25s  %-12s  %s\n",
				task.Metadata.Id,
				task.DisplayName,
				string(task.Status),
				dur,
			)
		}
	},
}

var runsEventsCmd = &cobra.Command{
	Use:   "events <run-id>",
	Short: "Print events from a run",
	Long:  `Fetch and print lifecycle events from a run to stdout. Works for both task runs and DAG workflow runs.`,
	Args:  cobra.ExactArgs(1),
	Example: `  # Print events to stdout
  hatchet runs events 8ff4f149-099e-4c16-a8d1-0535f8c79b83 --profile local

  # JSON output
  hatchet runs events 8ff4f149-099e-4c16-a8d1-0535f8c79b83 -o json | jq .`,
	Run: func(cmd *cobra.Command, args []string) {
		runID := args[0]
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)

		ctx := cmd.Context()
		runUUID, err := uuid.Parse(runID)
		if err != nil {
			cli.Logger.Fatalf("invalid run ID %q: %v", runID, err)
		}

		var events []rest.V1TaskEvent

		// Try as a workflow run (DAG) first — TaskEvents are embedded in the run details
		runResp, err := hatchetClient.API().V1WorkflowRunGetWithResponse(ctx, runUUID)
		if err == nil && runResp.JSON200 != nil && len(runResp.JSON200.TaskEvents) > 0 {
			events = runResp.JSON200.TaskEvents
		} else {
			// Fall back: treat run ID as a task ID and use the task events endpoint
			limit := int64(1000)
			offset := int64(0)
			evtResp, err := hatchetClient.API().V1TaskEventListWithResponse(
				ctx,
				runUUID,
				&rest.V1TaskEventListParams{
					Limit:  &limit,
					Offset: &offset,
				},
			)
			if err != nil {
				cli.Logger.Fatalf("failed to fetch events: %v", err)
			}
			if evtResp.JSON200 != nil && evtResp.JSON200.Rows != nil {
				events = *evtResp.JSON200.Rows
			}
		}

		if isJSON {
			printJSON(events)
			return
		}

		if len(events) == 0 {
			fmt.Println("No events found for this run.")
			return
		}

		for _, evt := range events {
			ts := evt.Timestamp.Format("15:04:05.000")
			taskName := ""
			if evt.TaskDisplayName != nil {
				taskName = *evt.TaskDisplayName
			}
			fmt.Printf("%s  %-30s  %-25s  %s\n", ts, string(evt.EventType), taskName, evt.Message)
		}
	},
}

func init() {
	rootCmd.AddCommand(runsCmd)
	runsCmd.AddCommand(runsListCmd, runsGetCmd, runsCancelCmd, runsReplayCmd, runsLogsCmd, runsEventsCmd, runsListChildrenCmd)

	// Persistent flags on parent (inherited by all subcommands)
	runsCmd.PersistentFlags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	runsCmd.PersistentFlags().StringP("output", "o", "", "Output format: json (skips interactive TUI)")

	// runs list flags
	runsListCmd.Flags().StringP("since", "s", "24h", "Show runs since this duration ago (e.g. 1h, 24h, 7d)")
	runsListCmd.Flags().String("until", "", "Show runs until this duration ago (e.g. 30m)")
	runsListCmd.Flags().StringP("workflow", "w", "", "Filter by workflow name or ID")
	runsListCmd.Flags().StringSlice("status", nil, "Filter by status (QUEUED,RUNNING,COMPLETED,FAILED,CANCELLED)")
	runsListCmd.Flags().Int64("limit", 50, "Number of results to return")
	runsListCmd.Flags().Int64("offset", 0, "Offset for pagination")
	runsListCmd.Flags().Bool("only-tasks", false, "Show only task runs, not DAG runs")

	// runs cancel flags
	runsCancelCmd.Flags().StringP("since", "s", "", "Cancel runs since this duration ago (e.g. 1h, 24h) [required for bulk cancel]")
	runsCancelCmd.Flags().String("until", "", "Cancel runs until this duration ago")
	runsCancelCmd.Flags().StringP("workflow", "w", "", "Filter by workflow name or ID")
	runsCancelCmd.Flags().StringSlice("status", nil, "Filter by status (QUEUED,RUNNING,COMPLETED,FAILED,CANCELLED)")
	runsCancelCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	// runs replay flags (same as cancel)
	runsReplayCmd.Flags().StringP("since", "s", "", "Replay runs since this duration ago (e.g. 1h, 24h) [required for bulk replay]")
	runsReplayCmd.Flags().String("until", "", "Replay runs until this duration ago")
	runsReplayCmd.Flags().StringP("workflow", "w", "", "Filter by workflow name or ID")
	runsReplayCmd.Flags().StringSlice("status", nil, "Filter by status (QUEUED,RUNNING,COMPLETED,FAILED,CANCELLED)")
	runsReplayCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	// runs list-children flags
	runsListChildrenCmd.Flags().Int64("limit", 50, "Number of results to return (for task children)")
	runsListChildrenCmd.Flags().Int64("offset", 0, "Offset for pagination (for task children)")

	// runs logs flags
	runsLogsCmd.Flags().Int64("tail", 0, "Number of most recent log lines to show (0 = all)")
	runsLogsCmd.Flags().String("since", "", "Only show logs newer than this duration ago (e.g. 5m, 1h, 24h)")
	runsLogsCmd.Flags().BoolP("follow", "f", false, "Poll for new logs and print them as they arrive (Ctrl+C to stop)")
}

// isJSONOutput returns true if --output json was specified
func isJSONOutput(cmd *cobra.Command) bool {
	output, _ := cmd.Flags().GetString("output")
	return strings.ToLower(output) == "json"
}

// parseSinceDuration parses a duration string (e.g. "1h", "24h", "7d") into a time.Time
func parseSinceDuration(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("duration cannot be empty")
	}

	// Try standard Go duration (e.g., "1h", "30m", "2h30m")
	d, err := time.ParseDuration(s)
	if err == nil {
		if d <= 0 {
			return time.Time{}, fmt.Errorf("duration must be positive (e.g. 1h, 24h)")
		}
		return time.Now().Add(-d), nil
	}

	// Try days format (e.g., "7d", "30d")
	if strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(s, "d")
		days, err := strconv.Atoi(daysStr)
		if err == nil && days > 0 {
			return time.Now().Add(-time.Duration(days) * 24 * time.Hour), nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid duration %q (use e.g. 1h, 24h, 7d)", s)
}

// parseStatuses parses a list of status strings into V1TaskStatus values
func parseStatuses(strs []string) ([]rest.V1TaskStatus, error) {
	var statuses []rest.V1TaskStatus
	for _, s := range strs {
		switch strings.ToUpper(s) {
		case "QUEUED":
			statuses = append(statuses, rest.V1TaskStatusQUEUED)
		case "RUNNING":
			statuses = append(statuses, rest.V1TaskStatusRUNNING)
		case "COMPLETED":
			statuses = append(statuses, rest.V1TaskStatusCOMPLETED)
		case "FAILED":
			statuses = append(statuses, rest.V1TaskStatusFAILED)
		case "CANCELLED":
			statuses = append(statuses, rest.V1TaskStatusCANCELLED)
		default:
			return nil, fmt.Errorf("invalid status %q (valid: QUEUED,RUNNING,COMPLETED,FAILED,CANCELLED)", s)
		}
	}
	return statuses, nil
}

// resolveWorkflowID resolves a workflow name or UUID string to an openapi_types.UUID
func resolveWorkflowID(ctx context.Context, hatchetClient client.Client, nameOrID string) (openapi_types.UUID, error) { //nolint:staticcheck
	// Try as UUID first
	parsed, err := uuid.Parse(nameOrID)
	if err == nil {
		return parsed, nil
	}

	// Search by name
	tenantUUID, err := uuid.Parse(hatchetClient.TenantId())
	if err != nil {
		return openapi_types.UUID{}, fmt.Errorf("invalid tenant ID: %w", err)
	}

	resp, err := hatchetClient.API().WorkflowListWithResponse(ctx, tenantUUID, &rest.WorkflowListParams{
		Name: &nameOrID,
	})
	if err != nil {
		return openapi_types.UUID{}, fmt.Errorf("failed to fetch workflows: %w", err)
	}

	if resp.JSON200 != nil && resp.JSON200.Rows != nil {
		for _, wf := range *resp.JSON200.Rows {
			if wf.Name == nameOrID {
				wfUUID, err := uuid.Parse(wf.Metadata.Id)
				if err == nil {
					return wfUUID, nil
				}
			}
		}
	}

	return openapi_types.UUID{}, fmt.Errorf("workflow %q not found", nameOrID)
}

// resolveWorkflowName returns the workflow name for a given name-or-UUID string.
// The cron/scheduled create endpoints use the workflow name as a path parameter
// (not the UUID), so this is needed when the user provides a UUID.
func resolveWorkflowName(ctx context.Context, hatchetClient client.Client, nameOrID string) (string, error) { //nolint:staticcheck
	// If it's not a UUID, assume it's already a name
	parsed, err := uuid.Parse(nameOrID)
	if err != nil {
		return nameOrID, nil
	}

	// It's a UUID — look up the workflow to get its name
	resp, err := hatchetClient.API().WorkflowGetWithResponse(ctx, parsed)
	if err != nil {
		return "", fmt.Errorf("failed to fetch workflow %q: %w", nameOrID, err)
	}
	if resp.JSON200 == nil {
		return "", fmt.Errorf("workflow %q not found (status %d)", nameOrID, resp.StatusCode())
	}

	return resp.JSON200.Name, nil
}

// promptSelectWorkflow fetches the list of workflows and shows an interactive selector.
// Returns the selected workflow name, or "" if no workflows are found or the form is cancelled.
func promptSelectWorkflow(ctx context.Context, hatchetClient client.Client) string { //nolint:staticcheck
	tenantUUID := clientTenantUUID(hatchetClient)
	limit := 200
	offset := 0
	resp, err := hatchetClient.API().WorkflowListWithResponse(ctx, tenantUUID, &rest.WorkflowListParams{
		Limit:  &limit,
		Offset: &offset,
	})
	if err != nil || resp.JSON200 == nil || resp.JSON200.Rows == nil || len(*resp.JSON200.Rows) == 0 {
		return ""
	}

	var options []huh.Option[string]
	for _, wf := range *resp.JSON200.Rows {
		options = append(options, huh.NewOption(wf.Name, wf.Name))
	}

	height := len(options)
	if height > 10 {
		height = 10
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a workflow").
				Options(options...).
				Height(height).
				Value(&selected),
		),
	).WithTheme(styles.HatchetTheme())

	if err := form.Run(); err != nil {
		return ""
	}
	return selected
}

// parseDurationFuture parses a duration string (e.g. "1h", "30m", "7d") for use with future times.
// Returns the duration to add to time.Now() to get the trigger time.
func parseDurationFuture(s string) (time.Duration, error) {
	// Try standard Go duration
	d, err := time.ParseDuration(s)
	if err == nil {
		if d <= 0 {
			return 0, fmt.Errorf("duration must be positive (e.g. 1h, 30m)")
		}
		return d, nil
	}
	// Try days (e.g. "7d")
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}
	return 0, fmt.Errorf("invalid duration %q (use e.g. 5m, 2h, 7d)", s)
}

// buildFilterAndParams builds a V1TaskFilter and V1WorkflowRunListParams from command flags
func buildFilterAndParams(ctx context.Context, cmd *cobra.Command, hatchetClient client.Client) (*rest.V1TaskFilter, *rest.V1WorkflowRunListParams, error) { //nolint:staticcheck
	sinceStr, _ := cmd.Flags().GetString("since")
	sinceTime, err := parseSinceDuration(sinceStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid --since value: %w", err)
	}

	untilStr, _ := cmd.Flags().GetString("until")
	var untilPtr *time.Time
	if untilStr != "" {
		var until time.Time
		until, err = parseSinceDuration(untilStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid --until value: %w", err)
		}
		untilPtr = &until
	}

	workflowStr, _ := cmd.Flags().GetString("workflow")
	var workflowUUIDs *[]openapi_types.UUID
	if workflowStr != "" {
		var wfUUID openapi_types.UUID
		wfUUID, err = resolveWorkflowID(ctx, hatchetClient, workflowStr)
		if err != nil {
			return nil, nil, fmt.Errorf("could not resolve workflow: %w", err)
		}
		workflowUUIDs = &[]openapi_types.UUID{wfUUID}
	}

	statusStrs, _ := cmd.Flags().GetStringSlice("status")
	statuses, err := parseStatuses(statusStrs)
	if err != nil {
		return nil, nil, err
	}
	var statusesPtr *[]rest.V1TaskStatus
	if len(statuses) > 0 {
		statusesPtr = &statuses
	}

	filter := &rest.V1TaskFilter{
		Since:       sinceTime,
		Until:       untilPtr,
		Statuses:    statusesPtr,
		WorkflowIds: workflowUUIDs,
	}

	listParams := &rest.V1WorkflowRunListParams{
		Since:    sinceTime,
		Until:    untilPtr,
		Statuses: statusesPtr,
	}
	if workflowUUIDs != nil {
		listParams.WorkflowIds = workflowUUIDs
	}

	return filter, listParams, nil
}

// countMatchingRuns estimates the total number of runs matching the given params.
// It uses limit=1 and reads NumPages as a proxy for total count. Note: this is an
// approximation — the actual number affected by cancel/replay may differ slightly if
// run statuses change between this count call and the subsequent operation.
func countMatchingRuns(ctx context.Context, hatchetClient client.Client, tenantUUID openapi_types.UUID, params *rest.V1WorkflowRunListParams) int64 { //nolint:staticcheck
	limit := int64(1)
	countParams := *params
	countParams.Limit = &limit

	resp, err := hatchetClient.API().V1WorkflowRunListWithResponse(ctx, tenantUUID, &countParams)
	if err != nil || resp.JSON200 == nil {
		return 0
	}

	if resp.JSON200.Pagination.NumPages != nil {
		return *resp.JSON200.Pagination.NumPages
	}
	return 0
}

// confirmAction prompts the user to confirm an action, returns true if confirmed
func confirmAction(message string) bool {
	fmt.Printf("%s [y/N]: ", message)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

// printJSON marshals and prints v as indented JSON to stdout
func printJSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		cli.Logger.Fatalf("failed to marshal JSON: %v", err)
	}
	fmt.Println(string(data))
}

// shortID returns the first 8 characters of a UUID string followed by "..."
func shortID(id string) string {
	if len(id) > 8 {
		return id[:8] + "..."
	}
	return id
}
