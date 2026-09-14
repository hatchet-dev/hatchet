package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
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

var eventStatuses = []rest.V1TaskStatus{
	rest.V1TaskStatusQUEUED,
	rest.V1TaskStatusRUNNING,
	rest.V1TaskStatusCOMPLETED,
	rest.V1TaskStatusFAILED,
	rest.V1TaskStatusCANCELLED,
}

var eventsCmd = &cobra.Command{
	Use:     "events",
	Aliases: []string{"event", "ev"},
	Short:   "Manage events",
	Long:    `Commands for pushing, listing, and inspecting events.`,
	Run:     func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var eventsPushCmd = &cobra.Command{
	Use:   "push [key]",
	Short: "Push an event",
	Long:  `Push an event with a payload. Payload comes from --data (JSON string) or --file (path, or - for stdin). Without --output json and missing arguments, launches an interactive form.`,
	Example: `  # Push with inline payload
  hatchet events push user:created --data '{"id": 123}'

  # Push with payload from a file
  hatchet events push user:created --file payload.json

  # Push with payload from stdin
  echo '{"id": 123}' | hatchet events push user:created --file - -o json`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)

		key := ""
		if len(args) == 1 {
			key = args[0]
		}
		dataStr, _ := cmd.Flags().GetString("data")
		file, _ := cmd.Flags().GetString("file")
		metadataStr, _ := cmd.Flags().GetString("metadata")
		priority, _ := cmd.Flags().GetInt32("priority")
		scope, _ := cmd.Flags().GetString("scope")

		if file != "" {
			raw, err := readInputFile(file)
			if err != nil {
				cli.Logger.Fatalf("failed to read payload file: %v", err)
			}
			dataStr = string(raw)
		}

		if !isJSON && (key == "" || dataStr == "") {
			if dataStr == "" {
				dataStr = "{}"
			}
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Event key").
						Value(&key).
						Placeholder("user:created"),
				),
				huh.NewGroup(
					huh.NewText().
						Title("Payload (JSON)").
						Value(&dataStr).
						Validate(func(s string) error {
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

		if key == "" {
			cli.Logger.Fatal("event key is required")
		}
		if dataStr == "" {
			cli.Logger.Fatal("--data or --file is required")
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
			cli.Logger.Fatalf("invalid payload JSON: %v", err)
		}

		body := rest.CreateEventRequest{
			Key:  key,
			Data: data,
		}
		if metadataStr != "" {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
				cli.Logger.Fatalf("invalid metadata JSON: %v", err)
			}
			body.AdditionalMetadata = &metadata
		}
		if cmd.Flags().Changed("priority") {
			body.Priority = &priority
		}
		if scope != "" {
			body.Scope = &scope
		}

		resp, err := hatchetClient.API().EventCreateWithResponse(cmd.Context(), clientTenantUUID(hatchetClient), body)
		if err != nil {
			cli.Logger.Fatalf("failed to push event: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(resp.JSON200)
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Pushed event: %s", key)))
		}
	},
}

var eventsBulkPushCmd = &cobra.Command{
	Use:   "bulk-push",
	Short: "Push multiple events",
	Long:  `Push multiple events at once from a JSON file containing an array of {key, data, additionalMetadata} entries. Use - to read from stdin.`,
	Example: `  # Push events from a file
  hatchet events bulk-push --file events.json

  # Push events from stdin
  cat events.json | hatchet events bulk-push --file -`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)

		file, _ := cmd.Flags().GetString("file")
		if file == "" {
			cli.Logger.Fatal("--file is required")
		}

		raw, err := readInputFile(file)
		if err != nil {
			cli.Logger.Fatalf("failed to read events file: %v", err)
		}

		var events []rest.CreateEventRequest
		if err := json.Unmarshal(raw, &events); err != nil {
			cli.Logger.Fatalf("invalid events JSON (expected an array of {key, data, additionalMetadata}): %v", err)
		}
		if len(events) == 0 {
			cli.Logger.Fatal("no events to push")
		}

		resp, err := hatchetClient.API().EventCreateBulkWithResponse(cmd.Context(), clientTenantUUID(hatchetClient), rest.BulkCreateEventRequest{
			Events: events,
		})
		if err != nil {
			cli.Logger.Fatalf("failed to push events: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(resp.JSON200)
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Pushed %d events", len(events))))
		}
	},
}

var eventsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List events",
	Long:  `List events. Without --output json, launches the interactive TUI. With --output json, outputs raw JSON.`,
	Example: `  # Launch interactive TUI (default)
  hatchet events list --profile local

  # JSON output with filters
  hatchet events list --keys user:created --statuses COMPLETED -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		selectedProfile, hatchetClient := clientFromCmd(cmd)

		if !isJSON {
			tuiM := newTUIModel(selectedProfile, hatchetClient)
			tuiM.currentViewType = ViewTypeEvents
			tuiM.currentView = tui.NewEventsView(tuiM.ctx)
			p := tea.NewProgram(tuiM, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				cli.Logger.Fatalf("error running TUI: %v", err)
			}
			return
		}

		keys, _ := cmd.Flags().GetStringSlice("keys")
		sinceStr, _ := cmd.Flags().GetString("since")
		untilStr, _ := cmd.Flags().GetString("until")
		statusStrs, _ := cmd.Flags().GetStringSlice("statuses")
		metadata, _ := cmd.Flags().GetStringArray("metadata")
		limit, _ := cmd.Flags().GetInt64("limit")
		offset, _ := cmd.Flags().GetInt64("offset")

		params := &rest.V1EventListParams{
			Limit:  &limit,
			Offset: &offset,
		}
		if len(keys) > 0 {
			params.Keys = &keys
		}
		if sinceStr != "" {
			since, err := time.Parse(time.RFC3339, sinceStr)
			if err != nil {
				cli.Logger.Fatalf("invalid --since (expected RFC3339, e.g. 2026-01-02T15:04:05Z): %v", err)
			}
			params.Since = &since
		}
		if untilStr != "" {
			until, err := time.Parse(time.RFC3339, untilStr)
			if err != nil {
				cli.Logger.Fatalf("invalid --until (expected RFC3339, e.g. 2026-01-02T15:04:05Z): %v", err)
			}
			params.Until = &until
		}
		if len(statusStrs) > 0 {
			statuses := make([]rest.V1TaskStatus, 0, len(statusStrs))
			for _, s := range statusStrs {
				status, err := parseEventStatus(s)
				if err != nil {
					cli.Logger.Fatalf("%v", err)
				}
				statuses = append(statuses, status)
			}
			params.WorkflowRunStatuses = &statuses
		}
		if len(metadata) > 0 {
			pairs := make([]string, 0, len(metadata))
			for _, m := range metadata {
				pair, err := parseMetadataPair(m)
				if err != nil {
					cli.Logger.Fatalf("%v", err)
				}
				pairs = append(pairs, pair)
			}
			params.AdditionalMetadata = &pairs
		}

		resp, err := hatchetClient.API().V1EventListWithResponse(cmd.Context(), clientTenantUUID(hatchetClient), params)
		if err != nil {
			cli.Logger.Fatalf("failed to list events: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var eventsGetCmd = &cobra.Command{
	Use:   "get <event-id>",
	Short: "Get an event",
	Long:  `Get details about a single event by id. Outputs raw JSON.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		eventUUID, err := uuid.Parse(args[0])
		if err != nil {
			cli.Logger.Fatalf("invalid event id %q: %v", args[0], err)
		}
		_, hatchetClient := clientFromCmd(cmd)

		resp, err := hatchetClient.API().V1EventGetWithResponse(cmd.Context(), clientTenantUUID(hatchetClient), eventUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to get event: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var eventsKeysCmd = &cobra.Command{
	Use:   "keys",
	Short: "List event keys",
	Long:  `List all event keys for the tenant. With --output json, outputs raw JSON; otherwise prints one key per line.`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)

		resp, err := hatchetClient.API().V1EventKeyListWithResponse(cmd.Context(), clientTenantUUID(hatchetClient))
		if err != nil {
			cli.Logger.Fatalf("failed to list event keys: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(resp.JSON200)
			return
		}
		if resp.JSON200.Rows != nil {
			for _, key := range *resp.JSON200.Rows {
				fmt.Println(key)
			}
		}
	},
}

func readInputFile(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func parseEventStatus(s string) (rest.V1TaskStatus, error) {
	status := rest.V1TaskStatus(strings.ToUpper(strings.TrimSpace(s)))
	for _, valid := range eventStatuses {
		if status == valid {
			return status, nil
		}
	}
	names := make([]string, len(eventStatuses))
	for i, valid := range eventStatuses {
		names[i] = string(valid)
	}
	return "", fmt.Errorf("invalid status %q (must be one of: %s)", s, strings.Join(names, ", "))
}

func parseMetadataPair(m string) (string, error) {
	if key, value, ok := strings.Cut(m, "="); ok && key != "" && value != "" {
		return key + ":" + value, nil
	}
	if key, value, ok := strings.Cut(m, ":"); ok && key != "" && value != "" {
		return m, nil
	}
	return "", fmt.Errorf("invalid metadata %q (expected key=value)", m)
}

func init() {
	rootCmd.AddCommand(eventsCmd)
	eventsCmd.AddCommand(eventsPushCmd, eventsBulkPushCmd, eventsListCmd, eventsGetCmd, eventsKeysCmd)

	eventsCmd.PersistentFlags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	eventsCmd.PersistentFlags().StringP("output", "o", "", "Output format: json (skips interactive TUI)")

	eventsPushCmd.Flags().StringP("data", "d", "", "Event payload as a JSON string")
	eventsPushCmd.Flags().StringP("file", "f", "", "Path to a JSON file with the event payload (- for stdin)")
	eventsPushCmd.Flags().StringP("metadata", "m", "", "Additional metadata as a JSON string")
	eventsPushCmd.Flags().Int32("priority", 0, "Priority of the event")
	eventsPushCmd.Flags().String("scope", "", "Scope for event filtering")

	eventsBulkPushCmd.Flags().StringP("file", "f", "", "Path to a JSON file with an array of {key, data, additionalMetadata} entries (- for stdin)")

	eventsListCmd.Flags().StringSlice("keys", nil, "Filter by event keys")
	eventsListCmd.Flags().String("since", "", "Only events after this time (RFC3339)")
	eventsListCmd.Flags().String("until", "", "Only events before this time (RFC3339)")
	eventsListCmd.Flags().StringSlice("statuses", nil, "Filter by workflow run statuses: QUEUED, RUNNING, COMPLETED, FAILED, CANCELLED")
	eventsListCmd.Flags().StringArrayP("metadata", "m", nil, "Filter by additional metadata as key=value (repeatable)")
	eventsListCmd.Flags().Int64("limit", 50, "Number of results to return")
	eventsListCmd.Flags().Int64("offset", 0, "Offset for pagination")
}
