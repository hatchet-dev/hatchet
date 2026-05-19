package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/tui/dag"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// DAGTab represents different tabs in the DAG workflow run view
type DAGTab int

const (
	DAGTabDAG DAGTab = iota // DAG visualization tab (default)
	DAGTabEvents
	DAGTabTasks
	DAGTabInput
)

// RunDetailsView displays details for a DAG workflow run (multiple tasks)
type RunDetailsView struct {
	dagError          error
	dagGraph          *dag.Graph
	eventsViewer      *ContentViewer
	debugLogger       *DebugLogger
	inputViewer       *ContentViewer
	details           *rest.V1WorkflowRunDetails
	dagRendered       string
	dagSelectedStepID string
	workflowRunID     string
	BaseModel
	activeTab    DAGTab
	selectedTask int
	loading      bool
	showDebug    bool
	viewerActive bool
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
		activeTab:     DAGTabDAG, // Default to DAG tab
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
			v.inputViewer.SetSize(v.Width-6, v.Height-18-2)
		}
		if v.eventsViewer != nil {
			v.eventsViewer.SetSize(v.Width-6, v.Height-18-2)
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
		// If debug view is showing, try handling debug-specific keys first
		if v.showDebug {
			if handled, cmd := HandleDebugKeyboard(v.debugLogger, msg.String()); handled {
				return v, cmd
			}
		}

		switch msg.String() {
		case "esc":
			// Go back to previous view
			v.debugLogger.Log("Navigating back to tasks view")
			return v, NewNavigateBackMsg()
		case "enter":
			// Activate content viewer or navigate to task depending on active tab
			switch v.activeTab {
			case DAGTabInput:
				if v.inputViewer != nil {
					v.inputViewer.Activate()
					v.viewerActive = true
					v.debugLogger.Log("Activated input viewer")
				}
				return v, nil
			case DAGTabEvents:
				if v.eventsViewer != nil {
					v.eventsViewer.Activate()
					v.viewerActive = true
					v.debugLogger.Log("Activated events viewer")
				}
				return v, nil
			case DAGTabTasks:
				// Navigate to selected task from tasks list
				if v.details != nil && len(v.details.Tasks) > 0 {
					if v.selectedTask >= 0 && v.selectedTask < len(v.details.Tasks) {
						task := v.details.Tasks[v.selectedTask]
						taskID := task.Metadata.Id
						v.debugLogger.Log("Navigating to task: %s", taskID)
						return v, NewNavigateToRunWithDetectionMsg(taskID)
					}
				}
				return v, nil
			case DAGTabDAG:
				// Navigate to selected task from DAG
				if v.dagGraph != nil && v.dagSelectedStepID != "" {
					node := v.dagGraph.GetNode(v.dagSelectedStepID)
					if node != nil {
						v.debugLogger.Log("Navigating to task from DAG: %s", node.TaskExternalID)
						return v, NewNavigateToRunWithDetectionMsg(node.TaskExternalID)
					}
				}
				return v, nil
			}
			return v, nil
		case "r":
			// Refresh the data
			v.debugLogger.Log("Manual refresh triggered")
			v.loading = true
			return v, v.fetchWorkflowRun()
		case "tab":
			// Switch between tabs (DAG -> Events -> Tasks -> Input)
			v.activeTab = (v.activeTab + 1) % 4
			v.debugLogger.Log("Switched to tab: %v", v.activeTab)
			return v, nil
		case "1":
			v.activeTab = DAGTabDAG
			v.debugLogger.Log("Switched to DAG tab")
			return v, nil
		case "2":
			v.activeTab = DAGTabEvents
			v.debugLogger.Log("Switched to Events tab")
			return v, nil
		case "3":
			v.activeTab = DAGTabTasks
			v.debugLogger.Log("Switched to Tasks tab")
			return v, nil
		case "4":
			v.activeTab = DAGTabInput
			v.debugLogger.Log("Switched to Input tab")
			return v, nil
		case "d":
			// Toggle debug view
			v.showDebug = !v.showDebug
			v.debugLogger.Log("Debug view toggled: %v", v.showDebug)
			return v, nil
		case "c":
			// Clear debug logs (only when in debug view and not prompting)
			if v.showDebug && !v.debugLogger.IsPromptingFile() {
				v.debugLogger.Clear()
				v.debugLogger.Log("Debug logs cleared")
			}
			return v, nil
		case "w":
			// Write logs to file (only when in debug view and not already prompting)
			if v.showDebug && !v.debugLogger.IsPromptingFile() {
				v.debugLogger.StartFilePrompt()
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
		case "left", "h":
			// Navigate left in DAG (previous node in visual order)
			if v.activeTab == DAGTabDAG && v.dagGraph != nil {
				v.navigateDAG("left")
			}
			return v, nil
		case "right", "l":
			// Navigate right in DAG (next node in visual order)
			if v.activeTab == DAGTabDAG && v.dagGraph != nil {
				v.navigateDAG("right")
			}
			return v, nil
		case "x":
			// Export DAG and shape data to file
			if v.details != nil {
				filename, err := v.exportDAGData()
				if err != nil {
					v.HandleError(fmt.Errorf("failed to export: %w", err))
					v.debugLogger.Log("Export failed: %v", err)
				} else {
					v.debugLogger.Log("Exported to: %s", filename)
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
			viewerHeight := v.Height - 18 - 2
			viewerWidth := v.Width - 6

			inputContent := v.getInputContent()
			v.inputViewer = NewContentViewer(inputContent, viewerWidth, viewerHeight)

			eventsContent := v.getEventsContent()
			v.eventsViewer = NewContentViewer(eventsContent, viewerWidth, viewerHeight)

			// Build and render DAG
			v.buildDAG()
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

	// Header with run name/workflow name
	title := "Run Details"
	if v.details != nil {
		if v.details.Run.DisplayName != "" {
			title = fmt.Sprintf("Run Details: %s", v.details.Run.DisplayName)
		} else if len(v.details.Tasks) > 0 && v.details.Tasks[0].WorkflowName != nil && *v.details.Tasks[0].WorkflowName != "" {
			title = fmt.Sprintf("Run Details: %s", *v.details.Tasks[0].WorkflowName)
		}
	}
	header := RenderHeader(title, v.Ctx.ProfileName, v.Width)
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
		b.WriteString(RenderError(fmt.Sprintf("Error: %v", v.Err), v.Width))
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

	// Tab navigation
	b.WriteString(v.renderTabs())
	b.WriteString("\n\n")

	// Tab content (includes DAG when that tab is active)
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
	switch v.activeTab {
	case DAGTabDAG:
		if v.dagGraph != nil {
			controlsText = "←/→: Navigate DAG  •  enter: View Task  •  tab/1/2/3/4: Switch Tabs  •  x: Export  •  r: Refresh  •  esc: Back  •  q: Quit"
		} else {
			controlsText = "tab/1/2/3/4: Switch Tabs  •  x: Export  •  r: Refresh  •  d: Debug  •  esc: Back  •  q: Quit"
		}
	case DAGTabTasks:
		controlsText = "↑/↓: Navigate  •  enter: View Task  •  tab/1/2/3/4: Switch Tabs  •  x: Export  •  r: Refresh  •  esc: Back  •  q: Quit"
	default:
		controlsText = "tab/1/2/3/4: Switch Tabs  •  x: Export  •  r: Refresh  •  d: Debug  •  esc: Back  •  q: Quit"
	}
	controls := footerStyle.Render(controlsText)
	b.WriteString(controls)

	return b.String()
}

// SetSize updates the view dimensions and rebuilds the DAG
func (v *RunDetailsView) SetSize(width, height int) {
	v.BaseModel.SetSize(width, height)

	// Update viewer sizes if they exist
	viewerHeight := height - 18 - 2
	viewerWidth := width - 6
	if v.inputViewer != nil {
		v.inputViewer.SetSize(viewerWidth, viewerHeight)
	}
	if v.eventsViewer != nil {
		v.eventsViewer.SetSize(viewerWidth, viewerHeight)
	}

	// Rebuild DAG with new dimensions
	if v.details != nil && len(v.details.Shape) > 0 {
		v.buildDAG()
	}
}

// renderDebugView renders the debug log overlay
func (v *RunDetailsView) renderDebugView() string {
	// Build view-specific context info
	extraInfo := fmt.Sprintf("Workflow Run ID: %s", v.workflowRunID)
	if v.details != nil {
		extraInfo += fmt.Sprintf(" | Status: %s", v.details.Run.Status)
	}

	return RenderDebugView(v.debugLogger, v.Width, v.Height, extraInfo)
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

// renderDAG renders the DAG visualization
func (v *RunDetailsView) renderDAG() string {
	if v.dagError != nil {
		errorStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.ErrorColor).
			Foreground(styles.ErrorColor).
			Padding(1, 2).
			Width(v.Width - 6)

		return errorStyle.Render(fmt.Sprintf("DAG Error: %v", v.dagError))
	}

	if v.dagRendered == "" {
		placeholderStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.MutedColor).
			Foreground(styles.MutedColor).
			Align(lipgloss.Center).
			Width(v.Width-6).
			Height(8).
			Padding(2, 1)

		return placeholderStyle.Render("[ DAG Visualization ]\n\nNo workflow graph available")
	}

	// Wrap DAG in a styled border
	dagStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.AccentColor).
		Padding(1, 2).
		Width(v.Width - 6)

	return dagStyle.Render(v.dagRendered)
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

	// DAG tab
	if v.activeTab == DAGTabDAG {
		tabs = append(tabs, activeTabStyle.Render("DAG"))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render("DAG"))
	}

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
	case DAGTabDAG:
		content = v.renderDAG()
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

	// DAG tab already has its own border styling in renderDAG()
	if v.activeTab == DAGTabDAG {
		return content
	}

	// Otherwise, wrap in styled border
	contentStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(styles.AccentColor).
		Padding(0, 1).
		Width(v.Width - 6).
		Height(v.Height - 18) // Reserve space for header, status, tabs, and footer

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

	// Header - using 2-char spacing for selector to match row rendering
	headerStyle := lipgloss.NewStyle().Foreground(styles.AccentColor).Bold(true)
	b.WriteString(headerStyle.Render(fmt.Sprintf("%-2s %-30s %-12s %-16s %-16s %-12s",
		"", "TASK NAME", "STATUS", "CREATED AT", "STARTED AT", "DURATION")))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 90))
	b.WriteString("\n")

	// Selected row style
	selectedRowStyle := lipgloss.NewStyle().
		Background(styles.Blue).
		Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#0A1029"}).
		Bold(true)

	// Render each task - matching runs_list.go column widths and format
	for i, task := range v.details.Tasks {
		// Task name - use full name, let terminal handle any overflow
		taskName := task.DisplayName

		// Status - use plain text, will be styled
		statusStyle := GetV1TaskStatusStyle(task.Status)
		status := statusStyle.Text

		// Created At
		createdAt := formatRelativeTime(task.TaskInsertedAt)

		// Started At
		startedAt := "N/A"
		if task.StartedAt != nil {
			startedAt = formatRelativeTime(*task.StartedAt)
		}

		// Duration
		duration := formatTaskDuration(&task)

		// Ensure exact column widths matching runs_list.go: 30, 12, 16, 16, 12
		// Truncate or pad each field to exact width
		if len(taskName) > 30 {
			taskName = taskName[:30]
		}
		taskNamePadded := taskName + strings.Repeat(" ", 30-len(taskName))

		if len(status) > 12 {
			status = status[:12]
		}
		statusPadded := status + strings.Repeat(" ", 12-len(status))

		if len(createdAt) > 16 {
			createdAt = createdAt[:16]
		}
		createdAtPadded := createdAt + strings.Repeat(" ", 16-len(createdAt))

		if len(startedAt) > 16 {
			startedAt = startedAt[:16]
		}
		startedAtPadded := startedAt + strings.Repeat(" ", 16-len(startedAt))

		if len(duration) > 12 {
			duration = duration[:12]
		}
		durationPadded := duration + strings.Repeat(" ", 12-len(duration))

		// Render row with selection highlight only on task name
		if i == v.selectedTask {
			b.WriteString("▸ ")
			b.WriteString(selectedRowStyle.Render(taskNamePadded))
			b.WriteString(lipgloss.NewStyle().Foreground(statusStyle.Foreground).Render(statusPadded))
			b.WriteString(createdAtPadded)
			b.WriteString(startedAtPadded)
			b.WriteString(durationPadded)
		} else {
			b.WriteString("  ")
			b.WriteString(taskNamePadded)
			b.WriteString(lipgloss.NewStyle().Foreground(statusStyle.Foreground).Render(statusPadded))
			b.WriteString(createdAtPadded)
			b.WriteString(startedAtPadded)
			b.WriteString(durationPadded)
		}

		b.WriteString("\n")
	}

	return b.String()
}

// getEventsContent gets the events content as a string with table formatting
func (v *RunDetailsView) getEventsContent() string {
	if v.details == nil || v.details.TaskEvents == nil || len(v.details.TaskEvents) == 0 {
		return "No events available"
	}

	// Sort events by timestamp (newest first)
	sortedEvents := SortEventsByTimestamp(v.details.TaskEvents)

	var b strings.Builder

	// Header - wider columns to reduce truncation
	headerStyle := lipgloss.NewStyle().Foreground(styles.AccentColor).Bold(true)
	b.WriteString(headerStyle.Render(fmt.Sprintf("%-3s %-18s %-35s %-20s %s", "", "TIME", "EVENT", "TASK", "DETAILS")))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 100))
	b.WriteString("\n")

	// Render all events (ContentViewer will handle scrolling)
	for _, event := range sortedEvents {
		// Get severity for color coding
		severity := GetEventSeverity(event.EventType)
		dot := RenderEventSeverityDot(severity)

		// Format timestamp (plain text, pad manually)
		timestamp := formatRelativeTime(event.Timestamp)
		timestampPadded := timestamp + strings.Repeat(" ", 18-len(timestamp))

		// Format event type (plain text, pad manually)
		eventType := FormatEventType(event.EventType)
		if len(eventType) > 35 {
			eventType = eventType[:35]
		}
		eventTypePadded := eventType + strings.Repeat(" ", 35-len(eventType))

		// Task name (plain text, pad manually)
		taskName := ""
		if event.TaskDisplayName != nil {
			taskName = *event.TaskDisplayName
			if len(taskName) > 18 {
				taskName = taskName[:15] + "..."
			}
		}
		taskNamePadded := taskName + strings.Repeat(" ", 20-len(taskName))

		// Build details column - no hard truncation, let it flow
		details := ""
		if event.Message != "" {
			details = event.Message
		}
		if event.ErrorMessage != nil && *event.ErrorMessage != "" {
			errorMsg := *event.ErrorMessage
			if details != "" {
				details += " | "
			}
			errorStyle := lipgloss.NewStyle().Foreground(styles.ErrorColor)
			details += errorStyle.Render("Error: " + errorMsg)
		}

		// Render row with manual alignment and styles applied after padding
		labelStyle := lipgloss.NewStyle().Foreground(styles.MutedColor)
		valueStyle := lipgloss.NewStyle().Foreground(styles.PrimaryColor)

		b.WriteString(dot)
		b.WriteString(" ")
		b.WriteString(labelStyle.Render(timestampPadded))
		b.WriteString(valueStyle.Render(eventTypePadded))
		b.WriteString(taskNamePadded)
		b.WriteString(details)
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

// calculateAvailableDAGHeight dynamically calculates how much vertical space is available for the DAG
// by measuring the actual rendered chrome elements
func (v *RunDetailsView) calculateAvailableDAGHeight() int {
	if v.Height == 0 {
		return 10 // Fallback minimum
	}

	title := "Run Details"
	if v.details != nil {
		if v.details.Run.DisplayName != "" {
			title = fmt.Sprintf("Run Details: %s", v.details.Run.DisplayName)
		} else if len(v.details.Tasks) > 0 && v.details.Tasks[0].WorkflowName != nil && *v.details.Tasks[0].WorkflowName != "" {
			title = fmt.Sprintf("Run Details: %s", *v.details.Tasks[0].WorkflowName)
		}
	}
	header := RenderHeader(title, v.Ctx.ProfileName, v.Width)
	headerHeight := lipgloss.Height(header) + 2 // +2 for spacing after header

	statusSection := v.renderStatusSection()
	statusHeight := lipgloss.Height(statusSection) + 2 // +2 for spacing after status

	tabs := v.renderTabs()
	tabsHeight := lipgloss.Height(tabs) + 2 // +2 for spacing after tabs

	footerHeight := 3 // Footer typically has border + padding + content

	// Account for DAG border and padding (from renderDAG style)
	// Border: 2 lines (top + bottom), Padding(1, 2): 2 lines (top + bottom)
	dagBorderPadding := 4

	usedHeight := headerHeight + statusHeight + tabsHeight + footerHeight + dagBorderPadding
	availableHeight := max(v.Height-usedHeight, 10) // Ensure minimum height of 10

	v.debugLogger.Log("DAG height calculation: total=%d, header=%d, status=%d, tabs=%d, footer=%d, border=%d, available=%d",
		v.Height, headerHeight, statusHeight, tabsHeight, footerHeight, dagBorderPadding, availableHeight)

	return availableHeight
}

// buildDAG builds and renders the DAG visualization
func (v *RunDetailsView) buildDAG() {
	if v.details == nil || len(v.details.Shape) == 0 {
		v.debugLogger.Log("No shape data available for DAG")
		v.dagRendered = ""
		v.dagError = nil
		return
	}

	dagHeight := v.calculateAvailableDAGHeight()
	dagWidth := v.Width - 10 // Account for border and padding

	v.debugLogger.Log("Building DAG graph: nodes=%d, dagWidth=%d, dagHeight=%d", len(v.details.Shape), dagWidth, dagHeight)

	// Build graph
	g, err := dag.BuildGraph(v.details.Shape, v.details.Tasks, dagWidth, dagHeight)
	if err != nil {
		v.debugLogger.Log("Failed to build DAG graph: %v", err)
		v.dagError = fmt.Errorf("failed to build graph: %w", err)
		return
	}

	v.dagGraph = g
	v.debugLogger.Log("DAG graph built: nodes=%d, edges=%d, components=%d", g.NodeCount(), g.EdgeCount(), g.ComponentCount())

	// Render DAG
	rendered, err := dag.Render(g, v.dagSelectedStepID)
	if err != nil {
		v.debugLogger.Log("Failed to render DAG: %v", err)
		// Try compact mode
		rendered, err = dag.RenderCompact(g, v.dagSelectedStepID)
		if err != nil {
			v.debugLogger.Log("Failed to render DAG in compact mode: %v", err)
			v.dagError = fmt.Errorf("graph too large to display: %w", err)
			return
		}
		v.debugLogger.Log("DAG rendered in compact mode")
	} else {
		v.debugLogger.Log("DAG rendered successfully")
	}

	v.dagRendered = rendered
	v.dagError = nil
}

// renderDAGWithSelection re-renders the DAG with updated selection without rebuilding
func (v *RunDetailsView) renderDAGWithSelection() {
	if v.dagGraph == nil {
		return
	}

	// Render DAG with new selection
	rendered, err := dag.Render(v.dagGraph, v.dagSelectedStepID)
	if err != nil {
		v.debugLogger.Log("Failed to re-render DAG with selection: %v", err)
		// Try compact mode
		rendered, err = dag.RenderCompact(v.dagGraph, v.dagSelectedStepID)
		if err != nil {
			v.debugLogger.Log("Failed to re-render DAG in compact mode: %v", err)
			return
		}
	}

	v.dagRendered = rendered
}

// navigateDAG navigates between nodes in the DAG using arrow keys
func (v *RunDetailsView) navigateDAG(direction string) {
	if v.dagGraph == nil {
		return
	}

	// Get navigable nodes in visual order
	navigableNodes := dag.GetNavigableNodes(v.dagGraph)
	if len(navigableNodes) == 0 {
		return
	}

	// Find current selection index
	currentIndex := -1
	for i, node := range navigableNodes {
		if node.StepID == v.dagSelectedStepID {
			currentIndex = i
			break
		}
	}

	// If no selection, select first node
	if currentIndex == -1 {
		v.dagSelectedStepID = navigableNodes[0].StepID
		v.debugLogger.Log("DAG: Selected first node: %s", v.dagSelectedStepID)
		v.renderDAGWithSelection() // Only re-render, don't rebuild
		return
	}

	var newIndex int
	switch direction {
	case "left":
		// Move to previous node in visual order
		newIndex = max(currentIndex-1, 0)
	case "right":
		// Move to next node in visual order
		newIndex = min(currentIndex+1, len(navigableNodes)-1)
	default:
		return
	}

	v.dagSelectedStepID = navigableNodes[newIndex].StepID
	v.debugLogger.Log("DAG: Navigated %s to: %s", direction, v.dagSelectedStepID)
	v.renderDAGWithSelection() // Only re-render, don't rebuild
}

// exportDAGData exports the DAG visualization and shape data to a file
func (v *RunDetailsView) exportDAGData() (string, error) {
	if v.details == nil {
		return "", fmt.Errorf("no workflow run data available")
	}

	// Create filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	runName := strings.ReplaceAll(v.details.Run.DisplayName, " ", "_")
	runName = strings.ReplaceAll(runName, "/", "-")
	filename := fmt.Sprintf("dag-export-%s-%s.txt", runName, timestamp)

	// Use current directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "/tmp"
	}
	filepath := filepath.Join(cwd, filename)

	// Build export content
	var b strings.Builder

	separator := strings.Repeat("=", 80)

	// Header
	b.WriteString(separator)
	b.WriteString("\n")
	b.WriteString("HATCHET WORKFLOW DAG EXPORT\n")
	b.WriteString(separator)
	b.WriteString("\n\n")

	// Metadata
	b.WriteString("Workflow Run: ")
	b.WriteString(v.details.Run.DisplayName)
	b.WriteString("\n")
	b.WriteString("Run ID: ")
	b.WriteString(v.workflowRunID)
	b.WriteString("\n")
	b.WriteString("Exported: ")
	b.WriteString(time.Now().Format(time.RFC3339))
	b.WriteString("\n")
	b.WriteString("Status: ")
	b.WriteString(string(v.details.Run.Status))
	b.WriteString("\n\n")

	// DAG Visualization
	b.WriteString(separator)
	b.WriteString("\n")
	b.WriteString("DAG VISUALIZATION\n")
	b.WriteString(separator)
	b.WriteString("\n\n")

	switch {
	case v.dagRendered != "":
		b.WriteString(v.dagRendered)
		b.WriteString("\n\n")
	case v.dagError != nil:
		b.WriteString("DAG Error: ")
		b.WriteString(v.dagError.Error())
		b.WriteString("\n\n")
	default:
		b.WriteString("No DAG visualization available\n\n")
	}

	// Shape Data (JSON)
	b.WriteString(separator)
	b.WriteString("\n")
	b.WriteString("WORKFLOW SHAPE (API Data)\n")
	b.WriteString(separator)
	b.WriteString("\n\n")

	shapeJSON, err := json.MarshalIndent(v.details.Shape, "", "  ")
	if err != nil {
		b.WriteString("Error marshaling shape: ")
		b.WriteString(err.Error())
		b.WriteString("\n")
	} else {
		b.WriteString(string(shapeJSON))
		b.WriteString("\n\n")
	}

	// Tasks Data (JSON)
	b.WriteString(separator)
	b.WriteString("\n")
	b.WriteString("TASKS DATA\n")
	b.WriteString(separator)
	b.WriteString("\n\n")

	tasksJSON, err := json.MarshalIndent(v.details.Tasks, "", "  ")
	if err != nil {
		b.WriteString("Error marshaling tasks: ")
		b.WriteString(err.Error())
		b.WriteString("\n")
	} else {
		b.WriteString(string(tasksJSON))
		b.WriteString("\n\n")
	}

	// Graph Statistics
	if v.dagGraph != nil {
		b.WriteString(separator)
		b.WriteString("\n")
		b.WriteString("GRAPH STATISTICS\n")
		b.WriteString(separator)
		b.WriteString("\n\n")

		fmt.Fprintf(&b, "Nodes: %d\n", v.dagGraph.NodeCount())
		fmt.Fprintf(&b, "Edges: %d\n", v.dagGraph.EdgeCount())
		fmt.Fprintf(&b, "Components: %d\n", v.dagGraph.ComponentCount())
		fmt.Fprintf(&b, "Actual Width: %d\n", v.dagGraph.ActualWidth)
		fmt.Fprintf(&b, "Actual Height: %d\n", v.dagGraph.ActualHeight)
		b.WriteString("\n")

		stats := v.dagGraph.GetComponentStats()
		fmt.Fprintf(&b, "Total Components: %d\n", stats.TotalComponents)
		fmt.Fprintf(&b, "Largest Component: %d nodes\n", stats.LargestComponent)
		fmt.Fprintf(&b, "Smallest Component: %d nodes\n", stats.SmallestComponent)
		fmt.Fprintf(&b, "Isolated Nodes: %d\n", stats.IsolatedNodes)
		b.WriteString("\n")
	}

	// Write to file
	if err := os.WriteFile(filepath, []byte(b.String()), 0600); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filepath, nil
}
