package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
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
  hatchet tui --profile production

  # Start TUI and navigate to a specific workflow run
  hatchet tui --run 8ff4f149-099e-4c16-a8d1-0535f8c79b83 --profile local`,
	Run: func(cmd *cobra.Command, args []string) {
		profileFlag, _ := cmd.Flags().GetString("profile")
		runFlag, _ := cmd.Flags().GetString("run")

		var selectedProfile string

		// Use profile from flag if provided, otherwise use default or show selection form
		if profileFlag != "" {
			selectedProfile = profileFlag
		} else {
			selectedProfile = selectProfileForm(true)

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
		hatchetClient, err := NewClientFromProfile(profile, &nopLogger)
		if err != nil {
			cli.Logger.Fatalf("could not create Hatchet client: %v", err)
		}

		// Start the TUI with optional run navigation
		var model tea.Model
		if runFlag != "" {
			model = tuiModelWithInitialRun{
				tuiModel:     newTUIModel(selectedProfile, hatchetClient),
				initialRunID: runFlag,
			}
		} else {
			model = newTUIModel(selectedProfile, hatchetClient)
		}

		p := tea.NewProgram(
			model,
			tea.WithAltScreen(),
		)

		if _, err := p.Run(); err != nil {
			cli.Logger.Fatalf("error running TUI: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
	tuiCmd.Flags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	tuiCmd.Flags().StringP("run", "r", "", "Navigate to a specific workflow run by ID")
}

// ViewType represents the type of primary view
type ViewType int

const (
	ViewTypeRuns ViewType = iota
	ViewTypeWorkflows
	ViewTypeWorkers
	ViewTypeRateLimits
	ViewTypeScheduledRuns
	ViewTypeCronJobs
	ViewTypeWebhooks
)

// viewOption represents a selectable view in the view selector
type viewOption struct {
	Name        string
	Description string
	Type        ViewType
}

// availableViews is the list of all primary views that can be selected
var availableViews = []viewOption{
	{Type: ViewTypeRuns, Name: "Runs", Description: "View task runs"},
	{Type: ViewTypeWorkflows, Name: "Workflows", Description: "View workflows"},
	{Type: ViewTypeWorkers, Name: "Workers", Description: "View workers"},
	{Type: ViewTypeRateLimits, Name: "Rate Limits", Description: "View rate limit usage"},
	{Type: ViewTypeScheduledRuns, Name: "Scheduled Runs", Description: "View scheduled runs"},
	{Type: ViewTypeCronJobs, Name: "Cron Jobs", Description: "View cron jobs"},
	{Type: ViewTypeWebhooks, Name: "Webhooks", Description: "View webhooks"},
}

// tuiModel is the root model that manages different views
type tuiModel struct {
	currentView          tui.View
	viewStack            []tui.View
	availableProfiles    []string
	ctx                  tui.ViewContext
	currentViewType      ViewType
	width                int
	height               int
	selectedViewIndex    int
	selectedProfileIndex int
	showViewSelector     bool
	showProfileSelector  bool
	showQuitConfirmation bool
}

func newTUIModel(profileName string, hatchetClient client.Client) tuiModel {
	// Create view context
	ctx := tui.ViewContext{
		ProfileName: profileName,
		Client:      hatchetClient,
	}

	// Initialize with runs list view (default)
	currentView := tui.NewRunsListView(ctx)

	// Get available profiles
	profiles := cli.GetProfiles()
	profileNames := make([]string, 0, len(profiles))
	for name := range profiles {
		profileNames = append(profileNames, name)
	}
	sort.Strings(profileNames)

	return tuiModel{
		currentView:          currentView,
		currentViewType:      ViewTypeRuns,
		viewStack:            []tui.View{},
		ctx:                  ctx,
		showViewSelector:     false,
		selectedViewIndex:    0,
		showProfileSelector:  false,
		selectedProfileIndex: 0,
		showQuitConfirmation: false,
		availableProfiles:    profileNames,
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
		// Handle quit confirmation modal input
		if m.showQuitConfirmation {
			switch msg.String() {
			case "y", "Y", "enter":
				// Confirm quit
				return m, tea.Quit
			case "n", "N", "esc", "q":
				// Cancel quit
				m.showQuitConfirmation = false
				return m, nil
			case "ctrl+c":
				// Hard quit bypasses confirmation
				return m, tea.Quit
			}
			// Ignore other keys when modal is open
			return m, nil
		}

		// Handle profile selector modal input
		if m.showProfileSelector {
			switch msg.String() {
			case "ctrl+c":
				// Hard quit from profile selector
				return m, tea.Quit
			case "q":
				// Show quit confirmation
				m.showQuitConfirmation = true
				return m, nil
			case "down", "j":
				// Cycle forward through profile options
				m.selectedProfileIndex = (m.selectedProfileIndex + 1) % len(m.availableProfiles)
				return m, nil
			case "up", "k":
				// Cycle backward through profile options
				m.selectedProfileIndex = (m.selectedProfileIndex - 1 + len(m.availableProfiles)) % len(m.availableProfiles)
				return m, nil
			case "enter":
				// Confirm selection and switch profile
				selectedProfileName := m.availableProfiles[m.selectedProfileIndex]
				m.showProfileSelector = false

				// Switch to the selected profile
				return m, m.switchProfile(selectedProfileName)
			case "esc":
				// Cancel without switching
				m.showProfileSelector = false
				return m, nil
			}
			// Ignore other keys when modal is open
			return m, nil
		}

		// Handle view selector modal input
		if m.showViewSelector {
			switch msg.String() {
			case "ctrl+c":
				// Hard quit from view selector
				return m, tea.Quit
			case "q":
				// Show quit confirmation
				m.showQuitConfirmation = true
				return m, nil
			case "shift+tab", "tab", "down", "j":
				// Cycle forward through view options
				m.selectedViewIndex = (m.selectedViewIndex + 1) % len(availableViews)
				return m, nil
			case "up", "k":
				// Cycle backward through view options
				m.selectedViewIndex = (m.selectedViewIndex - 1 + len(availableViews)) % len(availableViews)
				return m, nil
			case "enter":
				// Confirm selection and switch view
				selectedType := availableViews[m.selectedViewIndex].Type
				if selectedType != m.currentViewType {
					// Only switch if in a primary view
					if m.isInPrimaryView() {
						m.currentViewType = selectedType
						m.currentView = m.createViewForType(selectedType)
						m.currentView.SetSize(m.width, m.height)
						m.showViewSelector = false
						return m, m.currentView.Init()
					}
				}
				// Same view or in detail view, just close modal
				m.showViewSelector = false
				return m, nil
			case "esc":
				// Cancel without switching
				m.showViewSelector = false
				return m, nil
			}
			// Ignore other keys when modal is open
			return m, nil
		}

		// Handle global keyboard shortcuts when not in modals
		switch msg.String() {
		case "ctrl+c":
			// Hard quit - no confirmation
			return m, tea.Quit
		case "q":
			// Show quit confirmation modal
			m.showQuitConfirmation = true
			return m, nil
		case "shift+p", "P":
			// Open profile selector modal (shift+p or capital P)
			if len(m.availableProfiles) == 0 {
				// No profiles available
				return m, nil
			}

			// Find current profile in the list to set initial selection
			for i, profileName := range m.availableProfiles {
				if profileName == m.ctx.ProfileName {
					m.selectedProfileIndex = i
					break
				}
			}
			m.showProfileSelector = true
			return m, nil
		case "h":
			// Only open help view if not already in help view
			// If already in help view, let it handle the key to close itself
			if _, isHelpView := m.currentView.(*tui.HelpView); !isHelpView {
				// Open help view
				// Push current view onto stack for back navigation
				m.viewStack = append(m.viewStack, m.currentView)

				// Create and initialize help view
				helpView := tui.NewHelpView(m.ctx)
				helpView.SetSize(m.width, m.height)
				m.currentView = helpView

				return m, helpView.Init()
			}
		case "shift+tab", "v":
			// Open view selector modal (shift+tab or v)
			// Find current view type in the list to set initial selection
			for i, opt := range availableViews {
				if opt.Type == m.currentViewType {
					m.selectedViewIndex = i
					break
				}
			}
			m.showViewSelector = true
			return m, nil
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
		switch {
		case msg.Type == "task" && msg.TaskData != nil:
			// Single task view
			taskView := tui.NewSingleTaskView(m.ctx, msg.TaskData.Metadata.Id)
			taskView.SetSize(m.width, m.height)
			newView = taskView
		case msg.Type == "dag" && msg.DAGData != nil:
			// DAG workflow run view
			dagView := tui.NewRunDetailsView(m.ctx, msg.DAGData.Run.Metadata.Id)
			dagView.SetSize(m.width, m.height)
			newView = dagView
		default:
			// Unknown type or missing data — stay on current view
			return m, nil
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

	case tui.NavigateToWorkflowMsg:
		// Push current view onto stack for back navigation
		m.viewStack = append(m.viewStack, m.currentView)

		// Create and initialize workflow details view
		workflowView := tui.NewWorkflowDetailsView(m.ctx, msg.WorkflowID)
		workflowView.SetSize(m.width, m.height)
		m.currentView = workflowView

		return m, workflowView.Init()

	case tui.NavigateToWorkerMsg:
		// Push current view onto stack for back navigation
		m.viewStack = append(m.viewStack, m.currentView)

		// Create and initialize worker details view
		workerView := tui.NewWorkerDetailsView(m.ctx, msg.WorkerID)
		workerView.SetSize(m.width, m.height)
		m.currentView = workerView

		return m, workerView.Init()

	case profileSwitchedMsg:
		// Profile switch successful - update context and reset views
		m.ctx.ProfileName = msg.profileName
		m.ctx.Client = msg.hatchetClient

		// Clear view stack and reset to default view
		m.viewStack = []tui.View{}
		m.currentViewType = ViewTypeRuns
		m.currentView = m.createViewForType(ViewTypeRuns)
		m.currentView.SetSize(m.width, m.height)

		return m, m.currentView.Init()

	case profileSwitchErrorMsg:
		// Profile switch failed - stay on current view
		// TODO: Show error message to user
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

	// Show quit confirmation modal if open (highest priority)
	if m.showQuitConfirmation {
		return m.renderQuitConfirmation()
	}

	// Show profile selector modal if open
	if m.showProfileSelector {
		return m.renderProfileSelector()
	}

	// Show view selector modal if open
	if m.showViewSelector {
		return m.renderViewSelector()
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

		// Both failed — return the most informative error available
		if taskResult != nil && taskResult.err != nil {
			return tui.RunTypeDetectedMsg{
				Error: taskResult.err,
			}
		}
		if dagResult != nil && dagResult.err != nil {
			return tui.RunTypeDetectedMsg{
				Error: dagResult.err,
			}
		}
		// Both returned non-200 without a network error (e.g. run not yet started)
		return tui.RunTypeDetectedMsg{
			Error: fmt.Errorf("run %q not found", runID),
		}
	}
}

// createViewForType creates a new view instance for the given view type
func (m tuiModel) createViewForType(viewType ViewType) tui.View {
	switch viewType {
	case ViewTypeRuns:
		return tui.NewRunsListView(m.ctx)
	case ViewTypeWorkflows:
		return tui.NewWorkflowsView(m.ctx)
	case ViewTypeWorkers:
		return tui.NewWorkersView(m.ctx)
	case ViewTypeRateLimits:
		return tui.NewRateLimitsView(m.ctx)
	case ViewTypeScheduledRuns:
		return tui.NewScheduledRunsView(m.ctx)
	case ViewTypeCronJobs:
		return tui.NewCronJobsView(m.ctx)
	case ViewTypeWebhooks:
		return tui.NewWebhooksView(m.ctx)
	default:
		return tui.NewRunsListView(m.ctx)
	}
}

// isInPrimaryView checks if the current view is a primary view (not a detail view)
func (m tuiModel) isInPrimaryView() bool {
	// Check if we're in a primary view by checking if viewStack is empty
	// Detail views push the previous view onto the stack
	return len(m.viewStack) == 0
}

// renderViewSelector renders the view selector modal
func (m tuiModel) renderViewSelector() string {
	var b strings.Builder

	// Header (using reusable component)
	header := tui.RenderHeader("Select View", m.ctx.ProfileName, m.width)
	b.WriteString(header)
	b.WriteString("\n\n")

	// Instructions
	instructions := tui.RenderInstructions(
		"↑/↓ or Tab: Navigate  •  Enter: Confirm  •  Esc: Cancel",
		m.width,
	)
	b.WriteString(instructions)
	b.WriteString("\n\n")

	// View options list
	optionsStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Width(m.width - 4)

	for i, opt := range availableViews {
		var optionLine string
		if i == m.selectedViewIndex {
			// Highlighted option
			selectedStyle := lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#0A1029"}).
				Background(styles.Blue).
				Bold(true).
				Padding(0, 2).
				Width(m.width - 8)

			optionLine = selectedStyle.Render(fmt.Sprintf("▶ %s - %s", opt.Name, opt.Description))
		} else {
			// Non-highlighted option
			normalStyle := lipgloss.NewStyle().
				Foreground(styles.MutedColor).
				Padding(0, 2).
				Width(m.width - 8)

			optionLine = normalStyle.Render(fmt.Sprintf("  %s - %s", opt.Name, opt.Description))
		}

		b.WriteString(optionsStyle.Render(optionLine))
		b.WriteString("\n")
	}

	// Footer with controls (using reusable component)
	footer := tui.RenderFooter([]string{
		"Tab: Cycle",
		"Enter: Confirm",
		"Esc: Cancel",
		"q: Quit",
	}, m.width)
	b.WriteString("\n")
	b.WriteString(footer)

	return b.String()
}

// renderProfileSelector renders the profile selector modal
func (m tuiModel) renderProfileSelector() string {
	var b strings.Builder

	// Header (using reusable component)
	header := tui.RenderHeader("Switch Profile", m.ctx.ProfileName, m.width)
	b.WriteString(header)
	b.WriteString("\n\n")

	// Instructions
	instructions := tui.RenderInstructions(
		"↑/↓ or j/k: Navigate  •  Enter: Switch  •  Esc: Cancel",
		m.width,
	)
	b.WriteString(instructions)
	b.WriteString("\n\n")

	// Profile options list
	optionsStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Width(m.width - 4)

	for i, profileName := range m.availableProfiles {
		var optionLine string
		if i == m.selectedProfileIndex {
			// Highlighted option
			selectedStyle := lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#0A1029"}).
				Background(styles.Blue).
				Bold(true).
				Padding(0, 2).
				Width(m.width - 8)

			// Mark current profile
			prefix := "▶ "
			if profileName == m.ctx.ProfileName {
				prefix = "▶ ★ "
			}
			optionLine = selectedStyle.Render(fmt.Sprintf("%s%s", prefix, profileName))
		} else {
			// Non-highlighted option
			normalStyle := lipgloss.NewStyle().
				Foreground(styles.MutedColor).
				Padding(0, 2).
				Width(m.width - 8)

			// Mark current profile
			prefix := "  "
			if profileName == m.ctx.ProfileName {
				prefix = "  ★ "
			}
			optionLine = normalStyle.Render(fmt.Sprintf("%s%s", prefix, profileName))
		}

		b.WriteString(optionsStyle.Render(optionLine))
		b.WriteString("\n")
	}

	// Footer with controls (using reusable component)
	footer := tui.RenderFooter([]string{
		"↑/↓ j/k: Navigate",
		"Enter: Switch",
		"Esc: Cancel",
		"q: Quit",
	}, m.width)
	b.WriteString("\n")
	b.WriteString(footer)

	return b.String()
}

// switchProfile switches to a different profile
func (m tuiModel) switchProfile(profileName string) tea.Cmd {
	return func() tea.Msg {
		// Load the selected profile
		profile, err := cli.GetProfile(profileName)
		if err != nil {
			// Return error message
			return profileSwitchErrorMsg{err: err}
		}

		// Initialize new Hatchet client with the new profile's token
		nopLogger := zerolog.Nop()
		hatchetClient, err := NewClientFromProfile(profile, &nopLogger)
		if err != nil {
			return profileSwitchErrorMsg{err: err}
		}

		return profileSwitchedMsg{
			profileName:   profileName,
			hatchetClient: hatchetClient,
		}
	}
}

// renderQuitConfirmation renders the quit confirmation modal
func (m tuiModel) renderQuitConfirmation() string {
	var b strings.Builder

	// Header (using reusable component)
	header := tui.RenderHeader("Quit Hatchet TUI?", m.ctx.ProfileName, m.width)
	b.WriteString(header)
	b.WriteString("\n\n")

	// Confirmation message
	messageStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		Padding(1, 2).
		Width(m.width - 4).
		Align(lipgloss.Center)

	message := "Are you sure you want to quit?"
	b.WriteString(messageStyle.Render(message))
	b.WriteString("\n\n")

	// Options
	optionsStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Width(m.width - 4).
		Align(lipgloss.Center)

	yesStyle := lipgloss.NewStyle().
		Foreground(styles.AccentColor).
		Bold(true)

	options := fmt.Sprintf("%s to confirm  •  %s to cancel",
		yesStyle.Render("Y/Enter"),
		yesStyle.Render("N/Esc"))

	b.WriteString(optionsStyle.Render(options))
	b.WriteString("\n")

	// Footer hint
	footer := tui.RenderFooter([]string{
		"Y/Enter: Quit",
		"N/Esc: Cancel",
		"Ctrl+C: Force Quit",
	}, m.width)
	b.WriteString("\n")
	b.WriteString(footer)

	return b.String()
}

// profileSwitchedMsg is sent when a profile switch is successful
type profileSwitchedMsg struct {
	hatchetClient client.Client
	profileName   string
}

// profileSwitchErrorMsg is sent when a profile switch fails
type profileSwitchErrorMsg struct {
	err error
}
