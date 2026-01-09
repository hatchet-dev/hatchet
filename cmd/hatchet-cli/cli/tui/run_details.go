package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// DAGTab represents different tabs in the DAG workflow run view
type DAGTab int

const (
	DAGTabEvents DAGTab = iota // Default tab
	DAGTabTasks
	DAGTabInput
)

// RunDetailsView displays details for a DAG workflow run (multiple tasks)
type RunDetailsView struct {
	BaseModel
	workflowRunID string
	details       *rest.V1WorkflowRunDetails
	loading       bool
	activeTab     DAGTab
	selectedTask  int // Selected task index in Tasks tab
	debugLogger   *DebugLogger
	showDebug     bool // Whether to show debug overlay
	inputViewer   *ContentViewer
	eventsViewer  *ContentViewer
	viewerActive  bool // Whether content viewer is active
}

// workflowRunMsg contains the fetched workflow run details
type workflowRunMsg struct {
	details *rest.V1WorkflowRunDetails
	err     error
}

// NewRunDetailsView creates a new DAG workflow run details view
func NewRunDetailsView(ctx ViewContext, workflowRunID string) *RunDetailsView {
	v := &RunDetailsView{
		BaseModel: BaseModel{
			Ctx: ctx,
		},
		workflowRunID: workflowRunID,
		loading:       true,
		activeTab:     DAGTabEvents, // Default to Events tab
		debugLogger:   NewDebugLogger(5000),
		showDebug:     false,
	}

	v.debugLogger.Log("WorkflowRunView (DAG) initialized for run ID: %s", workflowRunID)

	return v
}

// Init initializes the view
func (v *RunDetailsView) Init() tea.Cmd {
	return v.fetchWorkflowRun()
}

// Update handles messages and updates the view state
func (v *RunDetailsView) Update(msg tea.Msg) (View, tea.Cmd) {
	// If viewer is active, delegate to it first
	if v.viewerActive {
		if v.inputViewer != nil && v.inputViewer.IsActive() {
			cmd := v.inputViewer.Update(msg)
			if !v.inputViewer.IsActive() {
				v.viewerActive = false
			}
			return v, cmd
		}
		if v.eventsViewer != nil && v.eventsViewer.IsActive() {
			cmd := v.eventsViewer.Update(msg)
			if !v.eventsViewer.IsActive() {
				v.viewerActive = false
			}
			return v, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.SetSize(msg.Width, msg.Height)
		// Update viewer sizes if they exist
		if v.inputViewer != nil {
			v.inputViewer.SetSize(v.Width-6, v.Height-25-2)
		}
		if v.eventsViewer != nil {
			v.eventsViewer.SetSize(v.Width-6, v.Height-25-2)
		}
		return v, nil

	case tea.MouseMsg:
		// Handle mouse events for content viewers in preview mode
		if !v.viewerActive {
			if v.activeTab == DAGTabInput && v.inputViewer != nil {
				v.inputViewer.HandleMouse(msg)
			} else if v.activeTab == DAGTabEvents && v.eventsViewer != nil {
				v.eventsViewer.HandleMouse(msg)
			}
		}
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Go back to previous view
			v.debugLogger.Log("Navigating back to tasks view")
			return v, NewNavigateBackMsg()
		case "enter":
			// Activate content viewer for Input/Events tab, or navigate for Tasks tab
			switch {
			case v.activeTab == DAGTabInput && v.inputViewer != nil:
				v.inputViewer.Activate()
				v.viewerActive = true
				v.debugLogger.Log("Activated input viewer")
				return v, nil
			case v.activeTab == DAGTabEvents && v.eventsViewer != nil:
				v.eventsViewer.Activate()
				v.viewerActive = true
				v.debugLogger.Log("Activated events viewer")
				return v, nil
			case v.activeTab == DAGTabTasks && v.details != nil && len(v.details.Tasks) > 0:
				// Navigate to selected task
				if v.selectedTask >= 0 && v.selectedTask < len(v.details.Tasks) {
					task := v.details.Tasks[v.selectedTask]
					taskID := task.Metadata.Id
					v.debugLogger.Log("Navigating to task: %s", taskID)
					return v, NewNavigateToRunWithDetectionMsg(taskID)
				}
			}
			return v, nil
		case "r":
			// Refresh the data
			v.debugLogger.Log("Manual refresh triggered")
			v.loading = true
			return v, v.fetchWorkflowRun()
		case "tab":
			// Switch between tabs (Events -> Tasks -> Input)
			v.activeTab = (v.activeTab + 1) % 3
			v.debugLogger.Log("Switched to tab: %v", v.activeTab)
			return v, nil
		case "1":
			v.activeTab = DAGTabEvents
			v.debugLogger.Log("Switched to Events tab")
			return v, nil
		case "2":
			v.activeTab = DAGTabTasks
			v.debugLogger.Log("Switched to Tasks tab")
			return v, nil
		case "3":
			v.activeTab = DAGTabInput
			v.debugLogger.Log("Switched to Input tab")
			return v, nil
		case "d":
			// Toggle debug view
			v.showDebug = !v.showDebug
			v.debugLogger.Log("Debug view toggled: %v", v.showDebug)
			return v, nil
		case "c":
			// Clear debug logs (only when in debug view)
			if v.showDebug {
				v.debugLogger.Clear()
				v.debugLogger.Log("Debug logs cleared")
			}
			return v, nil
		case "up", "k":
			// Navigate up in Tasks tab
			if v.activeTab == DAGTabTasks && v.details != nil && len(v.details.Tasks) > 0 {
				if v.selectedTask > 0 {
					v.selectedTask--
					v.debugLogger.Log("Selected task index: %d", v.selectedTask)
				}
			}
			return v, nil
		case "down", "j":
			// Navigate down in Tasks tab
			if v.activeTab == DAGTabTasks && v.details != nil && len(v.details.Tasks) > 0 {
				if v.selectedTask < len(v.details.Tasks)-1 {
					v.selectedTask++
					v.debugLogger.Log("Selected task index: %d", v.selectedTask)
				}
			}
			return v, nil
		}

	case workflowRunMsg:
		v.loading = false
		if msg.err != nil {
			v.HandleError(msg.err)
			v.debugLogger.Log("Error fetching workflow run: %v", msg.err)
		} else {
			v.details = msg.details
			v.ClearError()
			v.debugLogger.Log("Successfully fetched workflow run: %s", v.details.Run.DisplayName)

			// Initialize viewers
			viewerHeight := v.Height - 25 - 2
			viewerWidth := v.Width - 6

			inputContent := v.getInputContent()
			v.inputViewer = NewContentViewer(inputContent, viewerWidth, viewerHeight)

			eventsContent := v.getEventsContent()
			v.eventsViewer = NewContentViewer(eventsContent, viewerWidth, viewerHeight)
		}
		return v, nil
	}

	return v, nil
}

// View renders the view to a string
func (v *RunDetailsView) View() string {
	if v.Width == 0 {
		return "Initializing..."
	}

	// If debug view is enabled, show debug overlay
	if v.showDebug {
		return v.renderDebugView()
	}

	var b strings.Builder

	// Header with logo
	header := RenderHeaderWithLogo(
		fmt.Sprintf("Workflow Run Details - Profile: %s", v.Ctx.ProfileName),
		v.Width,
	)
	b.WriteString(header)
	b.WriteString("\n\n")

	if v.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(styles.AccentColor).
			Padding(0, 1)
		b.WriteString(loadingStyle.Render("Loading workflow run..."))
		b.WriteString("\n")
		return b.String()
	}

	if v.Err != nil {
		b.WriteString(RenderError(fmt.Sprintf("Error: %v", v.Err)))
		b.WriteString("\n")
		return b.String()
	}

	if v.details == nil {
		b.WriteString("No workflow run data available\n")
		return b.String()
	}

	// Status and timing section
	b.WriteString(v.renderStatusSection())
	b.WriteString("\n\n")

	// DAG placeholder
	b.WriteString(v.renderDAGPlaceholder())
	b.WriteString("\n\n")

	// Tab navigation
	b.WriteString(v.renderTabs())
	b.WriteString("\n\n")

	// Tab content
	b.WriteString(v.renderActiveTabContent())
	b.WriteString("\n\n")

	// Footer with controls
	footerStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(styles.AccentColor).
		Width(v.Width-4).
		Padding(0, 1)

	// Show different controls based on active tab
	var controlsText string
	if v.activeTab == DAGTabTasks {
		controlsText = "↑/↓: Navigate  •  enter: View Task  •  tab/1/2/3: Switch Tabs  •  r: Refresh  •  esc: Back  •  q: Quit"
	} else {
		controlsText = "tab/1/2/3: Switch Tabs  •  r: Refresh  •  d: Debug  •  esc: Back  •  q: Quit"
	}
	controls := footerStyle.Render(controlsText)
	b.WriteString(controls)

	return b.String()
}

// renderDebugView renders the debug log overlay
func (v *RunDetailsView) renderDebugView() string {
	logs := v.debugLogger.GetLogs()

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.AccentColor).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(styles.AccentColor).
		Width(v.Width-4).
		Padding(0, 1)

	header := headerStyle.Render(fmt.Sprintf(
		"Debug Logs - %d/%d entries",
		v.debugLogger.Size(),
		v.debugLogger.Capacity(),
	))

	// Log entries
	logStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(v.Width - 4)

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n\n")

	// Calculate how many logs we can show
	maxLines := v.Height - 8 // Reserve space for header, footer, controls
	if maxLines < 1 {
		maxLines = 1
	}

	// Show most recent logs first
	startIdx := 0
	if len(logs) > maxLines {
		startIdx = len(logs) - maxLines
	}

	for i := startIdx; i < len(logs); i++ {
		log := logs[i]
		timestamp := log.Timestamp.Format("15:04:05.000")
		logLine := fmt.Sprintf("[%s] %s", timestamp, log.Message)
		b.WriteString(logStyle.Render(logLine))
		b.WriteString("\n")
	}

	// Footer with controls
	footerStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(styles.AccentColor).
		Width(v.Width-4).
		Padding(0, 1)

	controls := footerStyle.Render("d: Close Debug  |  c: Clear Logs  |  q: Quit")
	b.WriteString("\n")
	b.WriteString(controls)

	return b.String()
}

// renderStatusSection renders the status and timing information
func (v *RunDetailsView) renderStatusSection() string {
	if v.details == nil {
		return ""
	}

	run := v.details.Run

	valueStyle := lipgloss.NewStyle().
		Foreground(styles.PrimaryColor).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		Bold(true)

	var b strings.Builder

	// Line 1: Name and Status (maximally space separated)
	nameStr := valueStyle.Render(run.DisplayName)
	statusStr := RenderV1TaskStatus(run.Status)

	// Calculate spacing between name and status
	line1Width := v.Width - 6 // Account for padding
	nameLen := lipgloss.Width(nameStr)
	statusLen := lipgloss.Width(statusStr)
	spacingLen := line1Width - nameLen - statusLen
	if spacingLen < 2 {
		spacingLen = 2
	}
	spacing := strings.Repeat(" ", spacingLen)

	b.WriteString(nameStr)
	b.WriteString(spacing)
	b.WriteString(statusStr)
	b.WriteString("\n\n")

	// Line 2: Timings
	var timings []string

	// Created At
	if run.CreatedAt != nil {
		timings = append(timings,
			labelStyle.Render("Created: ")+
				valueStyle.Bold(false).Render(formatRelativeTime(*run.CreatedAt)))
	}

	// Started At
	if run.StartedAt != nil {
		timings = append(timings,
			labelStyle.Render("Started: ")+
				valueStyle.Bold(false).Render(formatRelativeTime(*run.StartedAt)))
	}

	// Duration
	if run.Duration != nil {
		timings = append(timings,
			labelStyle.Render("Duration: ")+
				valueStyle.Bold(false).Render(formatDuration(*run.Duration)))
	}

	b.WriteString(strings.Join(timings, "  "))

	return b.String()
}

// renderDAGPlaceholder renders a placeholder for the DAG visualization
func (v *RunDetailsView) renderDAGPlaceholder() string {
	placeholderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.MutedColor).
		Foreground(styles.MutedColor).
		Align(lipgloss.Center).
		Width(v.Width-6).
		Height(8).
		Padding(2, 1)

	return placeholderStyle.Render("[ DAG Visualization Placeholder ]\n\nWorkflow execution graph will be displayed here")
}

// renderTabs renders the tab navigation
func (v *RunDetailsView) renderTabs() string {
	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#0A1029"}).
		Background(styles.Blue).
		Padding(0, 2)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		Padding(0, 2)

	var tabs []string

	// Events tab
	if v.activeTab == DAGTabEvents {
		tabs = append(tabs, activeTabStyle.Render("Events"))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render("Events"))
	}

	// Tasks tab
	if v.activeTab == DAGTabTasks {
		tabs = append(tabs, activeTabStyle.Render("Tasks"))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render("Tasks"))
	}

	// Input tab
	if v.activeTab == DAGTabInput {
		tabs = append(tabs, activeTabStyle.Render("Input"))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render("Input"))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

// renderActiveTabContent renders the content of the active tab
func (v *RunDetailsView) renderActiveTabContent() string {
	var content string

	switch v.activeTab {
	case DAGTabEvents:
		content = v.renderEventsContent()
	case DAGTabTasks:
		content = v.renderTasksContent()
	case DAGTabInput:
		content = v.renderInputContent()
	default:
		content = "Unknown tab"
	}

	// Check if we're in dive mode - if so, don't wrap in border (ContentViewer handles its own display)
	if v.viewerActive {
		return content
	}

	// Otherwise, wrap in styled border
	contentStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(styles.AccentColor).
		Padding(0, 1).
		Width(v.Width - 6).
		Height(v.Height - 25) // Reserve space for header, status, DAG, tabs, and footer

	return contentStyle.Render(content)
}

// renderTasksContent renders the tasks list for the DAG workflow run
func (v *RunDetailsView) renderTasksContent() string {
	if v.details == nil || v.details.Tasks == nil || len(v.details.Tasks) == 0 {
		return lipgloss.NewStyle().
			Foreground(styles.MutedColor).
			Render("No tasks available")
	}

	var b strings.Builder

	labelStyle := lipgloss.NewStyle().Foreground(styles.MutedColor).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(styles.PrimaryColor)

	// Selected row style
	selectedRowStyle := lipgloss.NewStyle().
		Background(styles.Blue).
		Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#0A1029"}).
		Bold(true)

	// Render each task
	for i, task := range v.details.Tasks {
		var taskLine strings.Builder

		// Task name
		taskLine.WriteString(labelStyle.Render("Task: "))
		taskLine.WriteString(valueStyle.Render(task.DisplayName))
		taskLine.WriteString("  ")

		// Status
		taskLine.WriteString(labelStyle.Render("Status: "))
		taskLine.WriteString(RenderV1TaskStatus(task.Status))
		taskLine.WriteString("  ")

		// Duration if available
		if task.Duration != nil {
			taskLine.WriteString(labelStyle.Render("Duration: "))
			taskLine.WriteString(valueStyle.Render(formatDuration(*task.Duration)))
		}

		// Apply selection highlight if this is the selected task
		line := taskLine.String()
		if i == v.selectedTask {
			// Add padding and apply selected style
			line = "▸ " + line
			line = selectedRowStyle.Render(line)
		} else {
			line = "  " + line
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

// getEventsContent gets the events content as a string
func (v *RunDetailsView) getEventsContent() string {
	if v.details == nil || v.details.TaskEvents == nil || len(v.details.TaskEvents) == 0 {
		return "No events available"
	}

	// Sort events by timestamp (newest first)
	sortedEvents := SortEventsByTimestamp(v.details.TaskEvents)

	var b strings.Builder

	// Render all events (ContentViewer will handle scrolling)
	for _, event := range sortedEvents {
		// Get severity for color coding
		severity := GetEventSeverity(event.EventType)
		dot := RenderEventSeverityDot(severity)

		// Format timestamp
		timestamp := formatRelativeTime(event.Timestamp)

		// Format event type
		eventType := FormatEventType(event.EventType)

		// Task name
		taskName := ""
		if event.TaskDisplayName != nil {
			taskName = *event.TaskDisplayName
		}

		// Build event line
		labelStyle := lipgloss.NewStyle().Foreground(styles.MutedColor)
		valueStyle := lipgloss.NewStyle().Foreground(styles.PrimaryColor)

		b.WriteString(dot)
		b.WriteString(" ")
		b.WriteString(labelStyle.Render(timestamp))
		b.WriteString(" ")
		b.WriteString(valueStyle.Render(eventType))

		if taskName != "" {
			b.WriteString(" ")
			b.WriteString(labelStyle.Render("•"))
			b.WriteString(" ")
			b.WriteString(taskName)
		}

		// Show message if present
		if event.Message != "" {
			b.WriteString("\n  ")
			b.WriteString(labelStyle.Render(event.Message))
		}

		// Show error if present (for FAILED events)
		if event.ErrorMessage != nil && *event.ErrorMessage != "" {
			errorStyle := lipgloss.NewStyle().Foreground(styles.ErrorColor)
			b.WriteString("\n  ")
			b.WriteString(errorStyle.Render("Error: " + *event.ErrorMessage))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// renderEventsContent renders the events list using ContentViewer
func (v *RunDetailsView) renderEventsContent() string {
	if v.eventsViewer == nil {
		return lipgloss.NewStyle().
			Foreground(styles.MutedColor).
			Render("No events available")
	}

	// If viewer is active, show full viewer UI
	if v.eventsViewer.IsActive() {
		return v.eventsViewer.View()
	}

	// Otherwise show preview with dive hint
	return v.eventsViewer.RenderPreview()
}

// getInputContent gets the input content as a string
func (v *RunDetailsView) getInputContent() string {
	if v.details == nil {
		return "No input data available"
	}

	// Pretty print the JSON input
	jsonBytes, err := json.MarshalIndent(v.details.Run.Input, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting input: %v", err)
	}

	return string(jsonBytes)
}

// renderInputContent renders the input data using ContentViewer
func (v *RunDetailsView) renderInputContent() string {
	if v.inputViewer == nil {
		return lipgloss.NewStyle().
			Foreground(styles.MutedColor).
			Render("No input data available")
	}

	// If viewer is active, show full viewer UI
	if v.inputViewer.IsActive() {
		return v.inputViewer.View()
	}

	// Otherwise show preview with dive hint
	return v.inputViewer.RenderPreview()
}

// fetchWorkflowRun fetches the workflow run details from the API
func (v *RunDetailsView) fetchWorkflowRun() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		v.debugLogger.Log("Fetching workflow run: %s", v.workflowRunID)

		// Parse workflow run ID as UUID
		workflowRunUUID, err := uuid.Parse(v.workflowRunID)
		if err != nil {
			v.debugLogger.Log("Failed to parse workflow run UUID: %v", err)
			return workflowRunMsg{
				details: nil,
				err:     fmt.Errorf("invalid workflow run ID: %w", err),
			}
		}

		// Call the API to get workflow run details
		v.debugLogger.Log("API call: V1WorkflowRunGetWithResponse(run=%s)", workflowRunUUID.String())
		response, err := v.Ctx.Client.API().V1WorkflowRunGetWithResponse(
			ctx,
			workflowRunUUID,
		)

		if err != nil {
			v.debugLogger.Log("API error: %v", err)
			return workflowRunMsg{
				details: nil,
				err:     fmt.Errorf("failed to fetch workflow run: %w", err),
			}
		}

		if response.JSON200 == nil {
			v.debugLogger.Log("Unexpected API response: status=%d", response.StatusCode())
			return workflowRunMsg{
				details: nil,
				err:     fmt.Errorf("unexpected response from API: status %d", response.StatusCode()),
			}
		}

		v.debugLogger.Log("API response: run=%s, tasks=%d", response.JSON200.Run.DisplayName, len(response.JSON200.Tasks))

		return workflowRunMsg{
			details: response.JSON200,
			err:     nil,
		}
	}
}
