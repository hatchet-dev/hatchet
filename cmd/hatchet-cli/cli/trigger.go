package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/worker"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/pm"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/tui"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	profileconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
)

const (
	// ReservedTriggerManual is the reserved keyword for manual workflow triggering
	ReservedTriggerManual = "manual"

	// ManualTriggerDescription is the description shown in the trigger selector
	ManualTriggerDescription = "Manually trigger a workflow via API"
)

var triggerCmd = &cobra.Command{
	Use:   "trigger [trigger-name]",
	Short: "Run a trigger defined in hatchet.yaml or manually trigger a workflow",
	Long:  `Execute a trigger defined in the triggers section of hatchet.yaml, or use "manual" to trigger any workflow via API. If no trigger name is provided, displays an interactive list of available triggers.`,
	Example: `  # Show interactive list of triggers
  hatchet trigger

  # Run a specific trigger by name
  hatchet trigger simple

  # Manually trigger a workflow interactively
  hatchet trigger manual

  # Manually trigger a workflow non-interactively
  hatchet trigger manual --workflow my-workflow --json ./input.json

  # Non-interactive with JSON output (prints {"runId": "...", "workflow": "..."})
  hatchet trigger manual --workflow my-workflow --json ./input.json -o json

  # Run with a specific profile
  hatchet trigger bulk --profile local`,
	Run: func(cmd *cobra.Command, args []string) {
		var triggerName string
		if len(args) > 0 {
			triggerName = args[0]
		}
		profileFlag, _ := cmd.Flags().GetString("profile")
		workflowFlag, _ := cmd.Flags().GetString("workflow")
		jsonFlag, _ := cmd.Flags().GetString("json")

		executeTrigger(cmd, triggerName, profileFlag, workflowFlag, jsonFlag)
	},
}

func init() {
	rootCmd.AddCommand(triggerCmd)

	// Add flags for trigger command
	triggerCmd.Flags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	triggerCmd.Flags().StringP("workflow", "w", "", "Workflow name for manual triggering (non-interactive mode)")
	triggerCmd.Flags().StringP("json", "j", "", "Path to JSON input file for manual triggering (non-interactive mode)")
	triggerCmd.Flags().StringP("output", "o", "", "Output format: json (prints run ID as JSON, skips TUI prompt)")
}

// executeTrigger is the main entry point for trigger execution
func executeTrigger(cmd *cobra.Command, triggerName string, profileFlag string, workflowFlag string, jsonFlag string) {
	// If workflow or json flags are provided, we're in non-interactive manual mode
	if workflowFlag != "" || jsonFlag != "" {
		if triggerName != "" && triggerName != ReservedTriggerManual {
			cli.Logger.Fatal("--workflow and --json flags can only be used with manual triggering")
		}
		if workflowFlag == "" || jsonFlag == "" {
			cli.Logger.Fatal("both --workflow and --json flags are required for non-interactive manual triggering")
		}
		runManualTrigger(cmd, workflowFlag, jsonFlag, profileFlag, false)
		return
	}

	// Handle manual trigger directly
	if triggerName == ReservedTriggerManual {
		runManualTrigger(cmd, "", "", profileFlag, true)
		return
	}

	// Try to load worker config for non-manual triggers
	workerConfig, err := worker.LoadWorkerConfig()

	// If specific trigger name provided, try to run it
	if triggerName != "" {
		if err != nil || workerConfig == nil {
			cli.Logger.Fatalf("could not load hatchet.yaml: %v\nTip: Use 'hatchet trigger manual' to trigger workflows without a config file", err)
		}

		// Validate trigger names
		if err := validateTriggerNames(workerConfig.Triggers); err != nil {
			cli.Logger.Fatal(err.Error())
		}

		// Find the trigger
		var selectedTrigger *worker.Trigger
		for i := range workerConfig.Triggers {
			trigger := &workerConfig.Triggers[i]
			triggerDisplayName := trigger.Name
			if triggerDisplayName == "" {
				triggerDisplayName = trigger.Command
			}
			if triggerDisplayName == triggerName {
				selectedTrigger = trigger
				break
			}
		}

		if selectedTrigger == nil {
			cli.Logger.Fatalf("trigger '%s' not found in hatchet.yaml", triggerName)
		}

		runConfigTrigger(cmd, selectedTrigger, profileFlag)
		return
	}

	// No trigger name provided - show selector
	var triggers []worker.Trigger
	if workerConfig != nil && err == nil {
		// Validate trigger names
		if err := validateTriggerNames(workerConfig.Triggers); err != nil {
			cli.Logger.Fatal(err.Error())
		}
		triggers = workerConfig.Triggers
	}

	// Show interactive selector (always includes manual option)
	selectedTriggerName := selectTriggerForm(triggers)

	if selectedTriggerName == ReservedTriggerManual {
		runManualTrigger(cmd, "", "", profileFlag, true)
		return
	}

	// Find and run the selected config trigger
	var selectedTrigger *worker.Trigger
	for i := range triggers {
		trigger := &triggers[i]
		triggerDisplayName := trigger.Name
		if triggerDisplayName == "" {
			triggerDisplayName = trigger.Command
		}
		if triggerDisplayName == selectedTriggerName {
			selectedTrigger = trigger
			break
		}
	}

	if selectedTrigger == nil {
		cli.Logger.Fatal("no trigger selected")
	}

	runConfigTrigger(cmd, selectedTrigger, profileFlag)
}

// validateTriggerNames checks that no trigger uses the reserved "manual" keyword
func validateTriggerNames(triggers []worker.Trigger) error {
	for _, trigger := range triggers {
		triggerName := trigger.Name
		if triggerName == "" {
			triggerName = trigger.Command
		}
		if triggerName == ReservedTriggerManual {
			return fmt.Errorf("trigger name '%s' is reserved. Please rename this trigger in your hatchet.yaml file", ReservedTriggerManual)
		}
	}
	return nil
}

// selectTriggerForm displays an interactive form to select a trigger
func selectTriggerForm(triggers []worker.Trigger) string {
	// Build options from triggers
	options := make([]huh.Option[string], 0, len(triggers)+1)

	for _, trigger := range triggers {
		triggerName := trigger.Name
		if triggerName == "" {
			triggerName = trigger.Command
		}

		label := triggerName
		if trigger.Description != "" {
			label = fmt.Sprintf("%s - %s", triggerName, trigger.Description)
		}

		options = append(options, huh.NewOption(label, triggerName))
	}

	// Always add manual option
	options = append(options, huh.NewOption(fmt.Sprintf("%s - %s", ReservedTriggerManual, ManualTriggerDescription), ReservedTriggerManual))

	var selectedName string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a trigger to run:").
				Options(options...).
				Value(&selectedName),
		),
	).WithTheme(styles.HatchetTheme())

	err := form.Run()
	if err != nil {
		cli.Logger.Fatalf("could not run trigger selection form: %v", err)
	}

	return selectedName
}

// runConfigTrigger executes a trigger from the worker configuration
func runConfigTrigger(cmd *cobra.Command, trigger *worker.Trigger, profileFlag string) {
	// Get profile
	var selectedProfile string
	if profileFlag != "" {
		selectedProfile = profileFlag
	} else {
		selectedProfile = selectProfileForm(true)

		if selectedProfile == "" {
			selectedProfile = handleNoProfiles(cmd)
			if selectedProfile == "" {
				cli.Logger.Fatal("no profile selected or created")
			}
		}
	}

	profile, err := cli.GetProfile(selectedProfile)
	if err != nil {
		cli.Logger.Fatalf("could not get profile '%s': %v", selectedProfile, err)
	}

	// Display trigger info
	triggerName := trigger.Name
	if triggerName == "" {
		triggerName = trigger.Command
	}

	fmt.Println(styles.InfoMessage(fmt.Sprintf("Running trigger: %s", triggerName)))
	if trigger.Description != "" {
		fmt.Println(styles.Muted.Render(trigger.Description))
	}
	fmt.Println(styles.Muted.Render(fmt.Sprintf("Command: %s", trigger.Command)))
	fmt.Println()

	// Execute the trigger
	ctx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	err = pm.Exec(ctx, trigger.Command, profile)
	if err != nil {
		cli.Logger.Fatalf("error running trigger '%s': %v", triggerName, err)
	}

	fmt.Println()
	fmt.Println(styles.SuccessMessage("Trigger completed successfully"))
}

// runManualTrigger executes manual workflow triggering
func runManualTrigger(cmd *cobra.Command, workflowFlag string, jsonFlag string, profileFlag string, interactive bool) {
	// Get profile
	var selectedProfile string
	if profileFlag != "" {
		selectedProfile = profileFlag
	} else {
		selectedProfile = selectProfileForm(true)

		if selectedProfile == "" {
			selectedProfile = handleNoProfiles(cmd)
			if selectedProfile == "" {
				cli.Logger.Fatal("no profile selected or created")
			}
		}
	}

	profile, err := cli.GetProfile(selectedProfile)
	if err != nil {
		cli.Logger.Fatalf("could not get profile '%s': %v", selectedProfile, err)
	}

	// Initialize Hatchet client
	nopLogger := zerolog.Nop()
	hatchetClient, err := NewClientFromProfile(profile, &nopLogger)
	if err != nil {
		cli.Logger.Fatalf("could not create Hatchet client: %v", err)
	}

	jsonOutput := isJSONOutput(cmd)

	if interactive {
		runManualInteractive(profile, hatchetClient, jsonOutput)
	} else {
		runManualNonInteractive(profile, hatchetClient, workflowFlag, jsonFlag, jsonOutput)
	}
}

// WorkflowInfo contains information about a workflow
type WorkflowInfo struct {
	ID      string
	Name    string
	Version string
}

// runManualInteractive runs manual workflow triggering in interactive mode
func runManualInteractive(profile *profileconfig.Profile, hatchetClient client.Client, jsonOutput bool) { //nolint:staticcheck
	ctx := context.Background()

	// Get tenant UUID
	tenantID := hatchetClient.TenantId()
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		cli.Logger.Fatalf("invalid tenant ID: %v", err)
	}

	// Fetch workflows
	fmt.Println(styles.InfoMessage("Fetching workflows..."))
	response, err := hatchetClient.API().WorkflowListWithResponse(ctx, tenantUUID, &rest.WorkflowListParams{})
	if err != nil {
		cli.Logger.Fatalf("could not fetch workflows: %v", err)
	}

	if response.JSON200 == nil || response.JSON200.Rows == nil || len(*response.JSON200.Rows) == 0 {
		cli.Logger.Fatal("no workflows available. Deploy a workflow first.")
	}

	// Build workflow list
	workflows := make([]WorkflowInfo, 0)
	for _, wf := range *response.JSON200.Rows {
		version := "latest"
		if wf.Versions != nil && len(*wf.Versions) > 0 {
			firstVersion := (*wf.Versions)[0]
			version = firstVersion.Version
		}
		workflows = append(workflows, WorkflowInfo{
			ID:      wf.Metadata.Id,
			Name:    wf.Name,
			Version: version,
		})
	}

	// Sort workflows by name
	sort.Slice(workflows, func(i, j int) bool {
		return workflows[i].Name < workflows[j].Name
	})

	// Select workflow
	selectedWorkflow := selectWorkflowForm(workflows)

	// Edit JSON input
	fmt.Println()
	fmt.Println(styles.InfoMessage("Opening editor for workflow input..."))
	jsonInput, err := editJSONInEditor("{}")
	if err != nil {
		cli.Logger.Fatalf("error editing JSON: %v", err)
	}

	// Validate JSON
	if err := validateJSON(jsonInput); err != nil {
		cli.Logger.Fatalf("invalid JSON: %v", err)
	}

	// Trigger workflow
	fmt.Println()
	fmt.Println(styles.InfoMessage(fmt.Sprintf("Triggering workflow: %s", selectedWorkflow.Name)))
	runID, err := triggerWorkflowWithClient(hatchetClient, selectedWorkflow.Name, jsonInput)
	if err != nil {
		cli.Logger.Fatalf("error triggering workflow: %v", err)
	}

	if jsonOutput {
		printJSON(struct {
			RunID    string `json:"runId"`
			Workflow string `json:"workflow"`
		}{RunID: runID, Workflow: selectedWorkflow.Name})
		return
	}

	// Display success message with TUI prompt (interactive mode only)
	displaySuccessMessage(runID, selectedWorkflow.Name, profile.Name, hatchetClient)
}

// runManualNonInteractive runs manual workflow triggering in non-interactive mode
func runManualNonInteractive(profile *profileconfig.Profile, hatchetClient client.Client, workflowName string, jsonPath string, jsonOutput bool) { //nolint:staticcheck
	ctx := context.Background()

	// Get tenant UUID
	tenantID := hatchetClient.TenantId()
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		cli.Logger.Fatalf("invalid tenant ID: %v", err)
	}

	// Fetch workflows
	response, err := hatchetClient.API().WorkflowListWithResponse(ctx, tenantUUID, &rest.WorkflowListParams{})
	if err != nil {
		cli.Logger.Fatalf("could not fetch workflows: %v", err)
	}

	if response.JSON200 == nil || response.JSON200.Rows == nil || len(*response.JSON200.Rows) == 0 {
		cli.Logger.Fatal("no workflows available. Deploy a workflow first.")
	}

	// Find workflow by name
	var selectedWorkflow *WorkflowInfo
	for _, wf := range *response.JSON200.Rows {
		if wf.Name == workflowName {
			version := "latest"
			if wf.Versions != nil && len(*wf.Versions) > 0 {
				firstVersion := (*wf.Versions)[0]
				version = firstVersion.Version
			}
			selectedWorkflow = &WorkflowInfo{
				ID:      wf.Metadata.Id,
				Name:    wf.Name,
				Version: version,
			}
			break
		}
	}

	if selectedWorkflow == nil {
		cli.Logger.Fatalf("workflow '%s' not found", workflowName)
		return // Make linter happy
	}

	// Read JSON from file
	jsonBytes, err := os.ReadFile(jsonPath)
	if err != nil {
		cli.Logger.Fatalf("could not read JSON file '%s': %v", jsonPath, err)
	}
	jsonInput := string(jsonBytes)

	// Validate JSON
	if err := validateJSON(jsonInput); err != nil {
		cli.Logger.Fatalf("invalid JSON in file '%s': %v", jsonPath, err)
	}

	// Trigger workflow
	selectedWorkflowName := selectedWorkflow.Name
	if !jsonOutput {
		fmt.Println(styles.InfoMessage(fmt.Sprintf("Triggering workflow: %s", selectedWorkflowName)))
	}
	runID, err := triggerWorkflowWithClient(hatchetClient, selectedWorkflowName, jsonInput)
	if err != nil {
		cli.Logger.Fatalf("error triggering workflow: %v", err)
	}

	if jsonOutput {
		printJSON(struct {
			RunID    string `json:"runId"`
			Workflow string `json:"workflow"`
		}{RunID: runID, Workflow: selectedWorkflowName})
		return
	}

	// Display success message (no TUI prompt in non-interactive mode)
	fmt.Println()
	fmt.Println(styles.SuccessMessage("Workflow triggered successfully"))
	fmt.Println()
	fmt.Println(styles.KeyValue("Run ID", runID))
	fmt.Println(styles.KeyValue("Workflow", selectedWorkflowName))
	fmt.Println()
	fmt.Println(styles.Muted.Render(fmt.Sprintf("View in TUI: hatchet tui --run %s --profile %s", runID, profile.Name)))
}

// selectWorkflowForm displays an interactive form to select a workflow
func selectWorkflowForm(workflows []WorkflowInfo) WorkflowInfo {
	options := make([]huh.Option[int], 0, len(workflows))
	for i, wf := range workflows {
		options = append(options, huh.NewOption(wf.Name, i))
	}

	var selectedIndex int
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Select a workflow to trigger:").
				Options(options...).
				Value(&selectedIndex),
		),
	).WithTheme(styles.HatchetTheme())

	err := form.Run()
	if err != nil {
		cli.Logger.Fatalf("could not run workflow selection form: %v", err)
	}

	return workflows[selectedIndex]
}

// editJSONInEditor opens an editor for the user to edit JSON input
func editJSONInEditor(initialContent string) (string, error) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "hatchet-input-*.json")
	if err != nil {
		return "", fmt.Errorf("could not create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write initial content
	if _, err := tmpFile.WriteString(initialContent); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("could not write to temp file: %w", err)
	}

	// Get original modification time
	originalStat, err := os.Stat(tmpPath)
	if err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("could not stat temp file: %w", err)
	}
	originalModTime := originalStat.ModTime()

	tmpFile.Close()

	// Detect editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		// Try common editors
		for _, e := range []string{"vi", "vim", "nano"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}

	if editor == "" {
		return "", fmt.Errorf("no editor found. Set the EDITOR environment variable or install vi, vim, or nano")
	}

	// Open editor
	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println(styles.Muted.Render("Opening editor... (save and quit to continue, or quit without saving to cancel)"))

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	// Check if file was modified
	newStat, err := os.Stat(tmpPath)
	if err != nil {
		return "", fmt.Errorf("could not stat temp file after edit: %w", err)
	}

	// If file wasn't modified, user likely quit without saving
	if newStat.ModTime().Equal(originalModTime) {
		// Read the content anyway - it might be OK with the default
		content, err := os.ReadFile(tmpPath)
		if err != nil {
			return "", fmt.Errorf("could not read edited file: %w", err)
		}

		// If content is just the initial content and user didn't modify, ask if they want to use it
		if string(content) == initialContent {
			fmt.Println()
			fmt.Println(styles.Muted.Render("Note: File was not modified. Using default input: {}"))
			fmt.Println()
		}

		return string(content), nil
	}

	// Read edited content
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", fmt.Errorf("could not read edited file: %w", err)
	}

	return string(content), nil
}

// validateJSON validates that the input is valid JSON
func validateJSON(content string) error {
	var js interface{}
	if err := json.Unmarshal([]byte(content), &js); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// triggerWorkflowWithClient triggers a workflow using the Hatchet client
func triggerWorkflowWithClient(hatchetClient client.Client, workflowName string, jsonInput string) (string, error) {
	// Parse JSON input
	var inputData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonInput), &inputData); err != nil {
		return "", fmt.Errorf("could not parse JSON input: %w", err)
	}

	// Trigger workflow using the Admin client with a timeout context
	// Create a channel to handle the async workflow trigger
	type result struct {
		err   error
		runID string
	}
	resultChan := make(chan result, 1)

	go func() {
		workflow, err := hatchetClient.Admin().RunWorkflow(workflowName, inputData)
		if err != nil {
			resultChan <- result{err: fmt.Errorf("could not trigger workflow: %w", err)}
			return
		}
		resultChan <- result{runID: workflow.RunId()}
	}()

	// Wait for result with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	select {
	case res := <-resultChan:
		if res.err != nil {
			return "", res.err
		}
		return res.runID, nil
	case <-ctx.Done():
		return "", fmt.Errorf("workflow trigger timed out after 30 seconds - this may indicate a connection issue with the Hatchet server")
	}
}

// displaySuccessMessage displays the success message and prompts to launch TUI
func displaySuccessMessage(runID string, workflowName string, profileName string, hatchetClient client.Client) {
	fmt.Println()
	fmt.Println(styles.SuccessMessage("Workflow triggered successfully"))
	fmt.Println()
	fmt.Println(styles.KeyValue("Run ID", runID))
	fmt.Println(styles.KeyValue("Workflow", workflowName))
	fmt.Println()
	fmt.Println("Press Enter to view in TUI (or Ctrl+C to exit)")

	// Wait for Enter key
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')

	// Launch TUI with run ID
	launchTUIWithRun(runID, profileName, hatchetClient)
}

// launchTUIWithRun launches the TUI with auto-navigation to the specified run
func launchTUIWithRun(runID string, profileName string, hatchetClient client.Client) {
	// Start the TUI with initial run ID by sending NavigateToRunWithDetectionMsg after initialization
	model := newTUIModel(profileName, hatchetClient)

	// Create a custom Init function that navigates to the run
	p := tea.NewProgram(
		tuiModelWithInitialRun{
			tuiModel:     model,
			initialRunID: runID,
		},
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		cli.Logger.Fatalf("error running TUI: %v", err)
	}
}

// tuiModelWithInitialRun wraps tuiModel to add initial run navigation
type tuiModelWithInitialRun struct {
	initialRunID string
	tuiModel
}

func (m tuiModelWithInitialRun) Init() tea.Cmd {
	return tea.Batch(
		m.tuiModel.Init(),
		func() tea.Msg {
			return tui.NavigateToRunWithDetectionMsg{RunID: m.initialRunID}
		},
	)
}
