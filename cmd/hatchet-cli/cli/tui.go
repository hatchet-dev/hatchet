package cli

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/tui"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

var tuiCmd = &cobra.Command{
	Use:     "tui",
	Aliases: []string{"terminal-ui"},
	Short:   "Start the Hatchet terminal UI",
	Long:    `Start an interactive terminal UI for viewing and managing Hatchet tasks.`,
	Example: `  # Start TUI (prompts for profile selection)
  hatchet tui

  # Start TUI with a specific profile
  hatchet tui --profile production`,
	Run: func(cmd *cobra.Command, args []string) {
		profileFlag, _ := cmd.Flags().GetString("profile")

		var selectedProfile string

		// Use profile from flag if provided, otherwise show selection form
		if profileFlag != "" {
			selectedProfile = profileFlag
		} else {
			selectedProfile = selectProfileForm()

			if selectedProfile == "" {
				cli.Logger.Fatal("no profile selected")
			}
		}

		profile, err := cli.GetProfile(selectedProfile)
		if err != nil {
			cli.Logger.Fatalf("could not get profile '%s': %v", selectedProfile, err)
		}

		// Initialize Hatchet client
		nopLogger := zerolog.Nop()
		hatchetClient, err := client.New(
			client.WithToken(profile.Token),
			client.WithLogger(&nopLogger),
		)
		if err != nil {
			cli.Logger.Fatalf("could not create Hatchet client: %v", err)
		}

		// Start the TUI
		p := tea.NewProgram(
			newTUIModel(selectedProfile, hatchetClient),
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(),
		)

		if _, err := p.Run(); err != nil {
			cli.Logger.Fatalf("error running TUI: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
	tuiCmd.Flags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
}

// tuiModel is the root model that manages different views
type tuiModel struct {
	currentView tui.View
	viewStack   []tui.View // Stack for back navigation
	ctx         tui.ViewContext
	width       int
	height      int
}

func newTUIModel(profileName string, hatchetClient client.Client) tuiModel {
	// Create view context
	ctx := tui.ViewContext{
		ProfileName: profileName,
		Client:      hatchetClient,
	}

	// Initialize with runs list view
	currentView := tui.NewRunsListView(ctx)

	return tuiModel{
		currentView: currentView,
		viewStack:   []tui.View{},
		ctx:         ctx,
	}
}

func (m tuiModel) Init() tea.Cmd {
	return m.currentView.Init()
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.currentView.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tui.NavigateToRunWithDetectionMsg:
		// Trigger run type detection
		return m, m.detectRunType(msg.RunID)

	case tui.RunTypeDetectedMsg:
		// Handle detection result
		if msg.Error != nil {
			// Detection failed - for now, just stay on current view
			// TODO: could show error in the current view
			return m, nil
		}

		// Push current view onto stack for back navigation
		m.viewStack = append(m.viewStack, m.currentView)

		// Navigate to appropriate view based on detected type
		var newView tui.View
		if msg.Type == "task" {
			// Single task view
			taskView := tui.NewSingleTaskView(m.ctx, msg.TaskData.Metadata.Id)
			taskView.SetSize(m.width, m.height)
			newView = taskView
		} else {
			// DAG workflow run view
			dagView := tui.NewRunDetailsView(m.ctx, msg.DAGData.Run.Metadata.Id)
			dagView.SetSize(m.width, m.height)
			newView = dagView
		}

		m.currentView = newView
		return m, newView.Init()

	case tui.NavigateToRunMsg:
		// Deprecated navigation - kept for compatibility
		// Push current view onto stack for back navigation
		m.viewStack = append(m.viewStack, m.currentView)

		// Create and initialize run details view
		runView := tui.NewRunDetailsView(m.ctx, msg.WorkflowRunID)
		runView.SetSize(m.width, m.height)
		m.currentView = runView

		return m, runView.Init()

	case tui.NavigateBackMsg:
		// Pop view from stack
		if len(m.viewStack) > 0 {
			m.currentView = m.viewStack[len(m.viewStack)-1]
			m.viewStack = m.viewStack[:len(m.viewStack)-1]
			m.currentView.SetSize(m.width, m.height)
		}
		return m, nil
	}

	// Delegate to current view
	var cmd tea.Cmd
	m.currentView, cmd = m.currentView.Update(msg)

	return m, cmd
}

func (m tuiModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	return m.currentView.View()
}

// detectRunType performs parallel API calls to detect whether a run ID is a task or workflow run
func (m tuiModel) detectRunType(runID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Parse run ID as UUID
		runUUID, err := uuid.Parse(runID)
		if err != nil {
			return tui.RunTypeDetectedMsg{
				Error: err,
			}
		}

		// Channel to collect results from parallel goroutines
		type apiResult struct {
			taskData *rest.V1TaskSummary
			dagData  *rest.V1WorkflowRunDetails
			err      error
			isTask   bool
		}

		resultChan := make(chan apiResult, 2)

		// Goroutine 1: Try to fetch as a task
		go func() {
			response, err := m.ctx.Client.API().V1TaskGetWithResponse(
				ctx,
				runUUID,
				&rest.V1TaskGetParams{},
			)

			if err != nil || response.JSON200 == nil {
				resultChan <- apiResult{err: err, isTask: true}
				return
			}

			resultChan <- apiResult{
				taskData: response.JSON200,
				isTask:   true,
			}
		}()

		// Goroutine 2: Try to fetch as a workflow run
		go func() {
			response, err := m.ctx.Client.API().V1WorkflowRunGetWithResponse(
				ctx,
				runUUID,
			)

			if err != nil || response.JSON200 == nil {
				resultChan <- apiResult{err: err, isTask: false}
				return
			}

			resultChan <- apiResult{
				dagData: response.JSON200,
				isTask:  false,
			}
		}()

		// Collect both results
		var taskResult, dagResult *apiResult
		for i := 0; i < 2; i++ {
			result := <-resultChan
			if result.isTask {
				taskResult = &result
			} else {
				dagResult = &result
			}
		}

		// Determine which type succeeded
		// Priority: task > dag (if both succeed, task wins)
		if taskResult != nil && taskResult.err == nil && taskResult.taskData != nil {
			return tui.RunTypeDetectedMsg{
				Type:     "task",
				TaskData: taskResult.taskData,
			}
		}

		if dagResult != nil && dagResult.err == nil && dagResult.dagData != nil {
			return tui.RunTypeDetectedMsg{
				Type:    "dag",
				DAGData: dagResult.dagData,
			}
		}

		// Both failed
		if taskResult != nil && taskResult.err != nil {
			return tui.RunTypeDetectedMsg{
				Error: taskResult.err,
			}
		}

		return tui.RunTypeDetectedMsg{
			Error: dagResult.err,
		}
	}
}
