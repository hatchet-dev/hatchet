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

// SingleTaskTab represents different tabs in the single task view
type SingleTaskTab int

const (
	SingleTaskTabOutput SingleTaskTab = iota
	SingleTaskTabInput
	SingleTaskTabEvents
	SingleTaskTabLogs
)

// SingleTaskView displays details for a single task run
type SingleTaskView struct {
	outputViewer *ContentViewer
	debugLogger  *DebugLogger
	task         *rest.V1TaskSummary
	eventsViewer *ContentViewer
	inputViewer  *ContentViewer
	taskID       string
	events       []rest.V1TaskEvent
	logs         []rest.V1LogLine
	BaseModel
	activeTab     SingleTaskTab
	loadingLogs   bool
	showDebug     bool
	loadingEvents bool
	loading       bool
	viewerActive  bool
}

// singleTaskMsg contains the fetched task details
type singleTaskMsg struct {
	task *rest.V1TaskSummary
	err  error
}

// eventsMsg contains the fetched task events
type eventsMsg struct {
	err    error
	events []rest.V1TaskEvent
}

// logsMsg contains the fetched log lines
type logsMsg struct {
	err  error
	logs []rest.V1LogLine
}

// NewSingleTaskView creates a new single task details view
func NewSingleTaskView(ctx ViewContext, taskID string) *SingleTaskView {
	v := &SingleTaskView{
		BaseModel: BaseModel{
			Ctx: ctx,
		},
		taskID:      taskID,
		loading:     true,
		activeTab:   SingleTaskTabOutput, // Default to Output tab
		debugLogger: NewDebugLogger(5000),
		showDebug:   false,
	}

	v.debugLogger.Log("SingleTaskView initialized for task ID: %s", taskID)

	return v
}

// Init initializes the view
func (v *SingleTaskView) Init() tea.Cmd {
	v.debugLogger.Log("=== SingleTaskView initialized, taskID: %s ===", v.taskID)
	return v.fetchTask()
}

// Update handles messages and updates the view state
func (v *SingleTaskView) Update(msg tea.Msg) (View, tea.Cmd) {
	// If a viewer is active, delegate to it first
	if v.viewerActive {
		var activeViewer *ContentViewer
		if v.activeTab == SingleTaskTabOutput && v.outputViewer != nil {
			activeViewer = v.outputViewer
		} else if v.activeTab == SingleTaskTabInput && v.inputViewer != nil {
			activeViewer = v.inputViewer
		} else if v.activeTab == SingleTaskTabEvents && v.eventsViewer != nil {
			activeViewer = v.eventsViewer
		}

		if activeViewer != nil && activeViewer.IsActive() {
			cmd := activeViewer.Update(msg)
			if !activeViewer.IsActive() {
				v.viewerActive = false
			}
			return v, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.SetSize(msg.Width, msg.Height)
		// Update viewer sizes if they exist
		if v.outputViewer != nil {
			v.outputViewer.SetSize(v.Width-6, v.Height-20-2)
		}
		if v.inputViewer != nil {
			v.inputViewer.SetSize(v.Width-6, v.Height-20-2)
		}
		if v.eventsViewer != nil {
			v.eventsViewer.SetSize(v.Width-6, v.Height-20-2)
		}
		return v, nil

	case tea.MouseMsg:
		// Handle mouse events for content viewers in preview mode
		if !v.viewerActive {
			if v.activeTab == SingleTaskTabOutput && v.outputViewer != nil {
				v.outputViewer.HandleMouse(msg)
			} else if v.activeTab == SingleTaskTabInput && v.inputViewer != nil {
				v.inputViewer.HandleMouse(msg)
			} else if v.activeTab == SingleTaskTabEvents && v.eventsViewer != nil {
				v.eventsViewer.HandleMouse(msg)
			}
		}
		return v, nil

	case tea.KeyMsg:
		// Don't log keyboard events when debug view is showing to avoid infinite loop
		if !v.showDebug {
			v.debugLogger.Log("KeyMsg received: %s", msg.String())
		}

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
			// Activate content viewer for current tab
			if v.activeTab == SingleTaskTabOutput && v.outputViewer != nil {
				v.outputViewer.Activate()
				v.viewerActive = true
				v.debugLogger.Log("Activated output viewer")
				return v, nil
			} else if v.activeTab == SingleTaskTabInput && v.inputViewer != nil {
				v.inputViewer.Activate()
				v.viewerActive = true
				v.debugLogger.Log("Activated input viewer")
				return v, nil
			} else if v.activeTab == SingleTaskTabEvents && v.eventsViewer != nil {
				v.eventsViewer.Activate()
				v.viewerActive = true
				v.debugLogger.Log("Activated events viewer")
				return v, nil
			}
			return v, nil
		case "r":
			// Refresh the data
			v.debugLogger.Log("Manual refresh triggered")
			v.loading = true
			return v, v.fetchTask()
		case "tab":
			// Switch between tabs
			v.activeTab = (v.activeTab + 1) % 4
			v.debugLogger.Log("Switched to tab: %v", v.activeTab)

			// Fetch data if switching to events or logs tab and data not loaded
			if v.activeTab == SingleTaskTabEvents && len(v.events) == 0 && !v.loadingEvents {
				v.debugLogger.Log("Tab switched to Events - fetching events")
				v.loadingEvents = true
				return v, v.fetchEvents()
			} else if v.activeTab == SingleTaskTabLogs && len(v.logs) == 0 && !v.loadingLogs {
				v.debugLogger.Log("Tab switched to Logs - fetching logs")
				v.loadingLogs = true
				return v, v.fetchLogs()
			}
			return v, nil
		case "1":
			v.activeTab = SingleTaskTabOutput
			v.debugLogger.Log("Switched to Output tab")
			return v, nil
		case "2":
			v.activeTab = SingleTaskTabInput
			v.debugLogger.Log("Switched to Input tab")
			return v, nil
		case "3":
			v.activeTab = SingleTaskTabEvents
			v.debugLogger.Log("Switched to Events tab")
			v.debugLogger.Log("Current task ID: %s", v.taskID)
			if v.task != nil {
				v.debugLogger.Log("Current task name: %s, task.Metadata.Id: %s", v.task.DisplayName, v.task.Metadata.Id)
			}
			// Fetch events if not already loaded
			if len(v.events) == 0 && !v.loadingEvents {
				v.debugLogger.Log("Fetching events (events currently empty)")
				v.loadingEvents = true
				return v, v.fetchEvents()
			}
			v.debugLogger.Log("Not fetching events: events count=%d, loadingEvents=%v", len(v.events), v.loadingEvents)
			return v, nil
		case "4":
			v.activeTab = SingleTaskTabLogs
			v.debugLogger.Log("Switched to Logs tab")
			// Fetch logs if not already loaded
			if len(v.logs) == 0 && !v.loadingLogs {
				v.loadingLogs = true
				return v, v.fetchLogs()
			}
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
		}

	case singleTaskMsg:
		v.loading = false
		if msg.err != nil {
			v.HandleError(msg.err)
			v.debugLogger.Log("Error fetching task: %v", msg.err)
		} else {
			v.task = msg.task
			v.ClearError()
			v.debugLogger.Log("Successfully fetched task: %s (ID: %s)", v.task.DisplayName, v.task.Metadata.Id)

			// Initialize content viewers
			viewerHeight := v.Height - 20 - 2
			viewerWidth := v.Width - 6

			// Output viewer
			outputContent := v.getOutputContent()
			v.outputViewer = NewContentViewer(outputContent, viewerWidth, viewerHeight)

			// Input viewer
			inputContent := v.getInputContent()
			v.inputViewer = NewContentViewer(inputContent, viewerWidth, viewerHeight)
		}
		return v, nil

	case eventsMsg:
		v.debugLogger.Log("Received eventsMsg: err=%v, events count=%d", msg.err != nil, len(msg.events))
		v.loadingEvents = false
		if msg.err != nil {
			v.debugLogger.Log("Error fetching events: %v", msg.err)
		} else {
			v.events = msg.events
			v.debugLogger.Log("Successfully fetched %d events", len(v.events))

			// Log first few events for debugging
			if len(v.events) > 0 {
				for i, evt := range v.events {
					if i < 3 {
						v.debugLogger.Log("Event %d: type=%s, timestamp=%v", i, evt.EventType, evt.Timestamp)
					}
				}
			}

			// Initialize events viewer
			viewerHeight := v.Height - 20 - 2
			viewerWidth := v.Width - 6
			v.debugLogger.Log("Creating events viewer: width=%d, height=%d", viewerWidth, viewerHeight)
			eventsContent := v.getEventsContent()
			v.debugLogger.Log("Events content length: %d", len(eventsContent))
			v.eventsViewer = NewContentViewer(eventsContent, viewerWidth, viewerHeight)
			v.debugLogger.Log("Events viewer created successfully")
		}
		return v, nil

	case logsMsg:
		v.loadingLogs = false
		if msg.err != nil {
			v.debugLogger.Log("Error fetching logs: %v", msg.err)
		} else {
			v.logs = msg.logs
			v.debugLogger.Log("Successfully fetched %d log lines", len(v.logs))
		}
		return v, nil
	}

	return v, nil
}

// View renders the view to a string
func (v *SingleTaskView) View() string {
	if v.Width == 0 {
		return "Initializing..."
	}

	// If debug view is enabled, show debug overlay
	if v.showDebug {
		return v.renderDebugView()
	}

	var b strings.Builder

	// Header with task name
	title := "Task Details"
	if v.task != nil {
		title = fmt.Sprintf("Task Details: %s", v.task.DisplayName)
	}
	header := RenderHeader(title, v.Ctx.ProfileName, v.Width)
	b.WriteString(header)
	b.WriteString("\n\n")

	if v.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(styles.AccentColor).
			Padding(0, 1)
		b.WriteString(loadingStyle.Render("Loading task..."))
		b.WriteString("\n")
		return b.String()
	}

	if v.Err != nil {
		b.WriteString(RenderError(fmt.Sprintf("Error: %v", v.Err), v.Width))
		b.WriteString("\n")
		return b.String()
	}

	if v.task == nil {
		b.WriteString("No task data available\n")
		return b.String()
	}

	// Status and timing section
	b.WriteString(v.renderStatusSection())
	b.WriteString("\n\n")

	// Tab navigation
	b.WriteString(v.renderTabs())
	b.WriteString("\n\n")

	// Tab content
	b.WriteString(v.renderActiveTabContent())
	b.WriteString("\n\n")

	// Footer with controls
	footer := RenderFooter([]string{
		"tab/1/2/3/4: Switch Tabs",
		"r: Refresh",
		"d: Debug",
		"esc: Back",
		"q: Quit",
	}, v.Width)
	b.WriteString(footer)

	return b.String()
}

// renderStatusSection renders the status and timing information
func (v *SingleTaskView) renderStatusSection() string {
	if v.task == nil {
		return ""
	}

	valueStyle := lipgloss.NewStyle().
		Foreground(styles.PrimaryColor).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		Bold(true)

	var b strings.Builder

	// Line 1: Name and Status (maximally space separated)
	nameStr := valueStyle.Render(v.task.DisplayName)
	statusStr := RenderV1TaskStatus(v.task.Status)

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

	// Created At - only show if not zero value
	if !v.task.CreatedAt.IsZero() {
		timings = append(timings,
			labelStyle.Render("Created: ")+
				valueStyle.Bold(false).Render(formatRelativeTime(v.task.CreatedAt)))
	}

	// Started At
	if v.task.StartedAt != nil {
		timings = append(timings,
			labelStyle.Render("Started: ")+
				valueStyle.Bold(false).Render(formatRelativeTime(*v.task.StartedAt)))
	}

	// Duration
	if v.task.Duration != nil {
		timings = append(timings,
			labelStyle.Render("Duration: ")+
				valueStyle.Bold(false).Render(formatDuration(*v.task.Duration)))
	}

	b.WriteString(strings.Join(timings, "  "))

	return b.String()
}

// renderTabs renders the tab navigation
func (v *SingleTaskView) renderTabs() string {
	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#0A1029"}).
		Background(styles.Blue).
		Padding(0, 2)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		Padding(0, 2)

	var tabs []string

	// Output tab
	if v.activeTab == SingleTaskTabOutput {
		tabs = append(tabs, activeTabStyle.Render("Output"))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render("Output"))
	}

	// Input tab
	if v.activeTab == SingleTaskTabInput {
		tabs = append(tabs, activeTabStyle.Render("Input"))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render("Input"))
	}

	// Events tab
	if v.activeTab == SingleTaskTabEvents {
		tabs = append(tabs, activeTabStyle.Render("Events"))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render("Events"))
	}

	// Logs tab
	if v.activeTab == SingleTaskTabLogs {
		tabs = append(tabs, activeTabStyle.Render("Logs"))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render("Logs"))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

// renderActiveTabContent renders the content of the active tab
func (v *SingleTaskView) renderActiveTabContent() string {
	var content string

	switch v.activeTab {
	case SingleTaskTabOutput:
		content = v.renderOutputContent()
	case SingleTaskTabInput:
		content = v.renderInputContent()
	case SingleTaskTabEvents:
		content = v.renderEventsContent()
	case SingleTaskTabLogs:
		content = v.renderLogsContent()
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
		Padding(1, 2).
		Width(v.Width - 6).
		Height(v.Height - 20) // Reserve space for header, status, tabs, and footer

	return contentStyle.Render(content)
}

// getOutputContent gets the output content as a string
func (v *SingleTaskView) getOutputContent() string {
	if v.task == nil {
		return "No output data available"
	}

	// Check if task failed - show error message
	if v.task.ErrorMessage != nil && *v.task.ErrorMessage != "" {
		return *v.task.ErrorMessage
	}

	// Show output JSON
	jsonBytes, err := json.MarshalIndent(v.task.Output, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting output: %v", err)
	}

	return string(jsonBytes)
}

// renderOutputContent renders the output or error message using ContentViewer
func (v *SingleTaskView) renderOutputContent() string {
	if v.outputViewer == nil {
		return lipgloss.NewStyle().
			Foreground(styles.MutedColor).
			Render("No output data available")
	}

	// If viewer is active, show full viewer UI
	if v.outputViewer.IsActive() {
		return v.outputViewer.View()
	}

	// Otherwise show preview with dive hint
	return v.outputViewer.RenderPreview()
}

// getInputContent gets the input content as a string
func (v *SingleTaskView) getInputContent() string {
	if v.task == nil {
		return "No input data available"
	}

	// Pretty print the JSON input
	jsonBytes, err := json.MarshalIndent(v.task.Input, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting input: %v", err)
	}

	return string(jsonBytes)
}

// renderInputContent renders the input data using ContentViewer
func (v *SingleTaskView) renderInputContent() string {
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

// getEventsContent gets the events content as a string with table formatting
func (v *SingleTaskView) getEventsContent() string {
	v.debugLogger.Log("getEventsContent: events count=%d", len(v.events))

	if len(v.events) == 0 {
		v.debugLogger.Log("getEventsContent: returning 'No events available'")
		return "No events available"
	}

	// Sort events by timestamp (newest first)
	sortedEvents := SortEventsByTimestamp(v.events)
	v.debugLogger.Log("getEventsContent: sorted %d events", len(sortedEvents))

	var b strings.Builder

	// Header - wider columns to reduce truncation
	headerStyle := lipgloss.NewStyle().Foreground(styles.AccentColor).Bold(true)
	b.WriteString(headerStyle.Render(fmt.Sprintf("%-3s %-18s %-35s %s", "", "TIME", "EVENT", "DETAILS")))
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
		b.WriteString(details)
		b.WriteString("\n")
	}

	return b.String()
}

// renderEventsContent renders the events list using ContentViewer
func (v *SingleTaskView) renderEventsContent() string {
	v.debugLogger.Log("renderEventsContent: loadingEvents=%v, eventsViewer=%v, events count=%d", v.loadingEvents, v.eventsViewer != nil, len(v.events))

	if v.loadingEvents {
		v.debugLogger.Log("renderEventsContent: showing loading state")
		return lipgloss.NewStyle().
			Foreground(styles.AccentColor).
			Render("Loading events...")
	}

	if v.eventsViewer == nil {
		v.debugLogger.Log("renderEventsContent: eventsViewer is nil, showing 'No events available'")
		return lipgloss.NewStyle().
			Foreground(styles.MutedColor).
			Render("No events available")
	}

	// If viewer is active, show full viewer UI
	if v.eventsViewer.IsActive() {
		v.debugLogger.Log("renderEventsContent: showing active viewer")
		return v.eventsViewer.View()
	}

	// Otherwise show preview with dive hint
	v.debugLogger.Log("renderEventsContent: showing preview")
	return v.eventsViewer.RenderPreview()
}

// renderLogsContent renders the logs
func (v *SingleTaskView) renderLogsContent() string {
	if v.loadingLogs {
		return lipgloss.NewStyle().
			Foreground(styles.AccentColor).
			Render("Loading logs...")
	}

	if len(v.logs) == 0 {
		return lipgloss.NewStyle().
			Foreground(styles.MutedColor).
			Render("No logs available")
	}

	// Calculate available height (subtract padding)
	availableHeight := v.Height - 20 - 2
	if availableHeight < 1 {
		availableHeight = 1
	}

	// Limit logs to display (most recent first)
	logsToShow := v.logs
	if len(logsToShow) > availableHeight {
		// Show most recent logs
		logsToShow = logsToShow[len(logsToShow)-availableHeight:]
	}

	var b strings.Builder

	timestampStyle := lipgloss.NewStyle().Foreground(styles.MutedColor)

	for _, logLine := range logsToShow {
		// Format timestamp
		timestamp := logLine.CreatedAt.Format("15:04:05.000")

		// Determine log level color
		var levelStyle lipgloss.Style
		if logLine.Level != nil {
			switch *logLine.Level {
			case rest.V1LogLineLevelERROR:
				levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Bold(true)
			case rest.V1LogLineLevelWARN:
				levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#eab308")).Bold(true)
			case rest.V1LogLineLevelDEBUG:
				levelStyle = lipgloss.NewStyle().Foreground(styles.MutedColor)
			default: // INFO
				levelStyle = lipgloss.NewStyle().Foreground(styles.PrimaryColor)
			}
		} else {
			levelStyle = lipgloss.NewStyle().Foreground(styles.PrimaryColor)
		}

		// Format level
		level := "INFO"
		if logLine.Level != nil {
			level = string(*logLine.Level)
		}

		// Build log line
		b.WriteString(timestampStyle.Render("[" + timestamp + "]"))
		b.WriteString(" ")
		b.WriteString(levelStyle.Render(level))
		b.WriteString(" ")
		b.WriteString(logLine.Message)
		b.WriteString("\n")
	}

	return b.String()
}

// renderDebugView renders the debug log overlay
func (v *SingleTaskView) renderDebugView() string {
	// Build view-specific context info
	extraInfo := fmt.Sprintf("Task ID: %s", v.taskID)
	if v.task != nil {
		extraInfo += fmt.Sprintf(" | Task: %s | Status: %s", v.task.DisplayName, v.task.Status)
	}

	return RenderDebugView(v.debugLogger, v.Width, v.Height, extraInfo)
}

// fetchTask fetches the task details from the API
func (v *SingleTaskView) fetchTask() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		v.debugLogger.Log("Fetching task: %s", v.taskID)

		// Parse task ID as UUID
		taskUUID, err := uuid.Parse(v.taskID)
		if err != nil {
			v.debugLogger.Log("Failed to parse task UUID: %v", err)
			return singleTaskMsg{
				task: nil,
				err:  fmt.Errorf("invalid task ID: %w", err),
			}
		}

		// Call the API to get task details
		v.debugLogger.Log("API call: V1TaskGetWithResponse(task=%s)", taskUUID.String())
		response, err := v.Ctx.Client.API().V1TaskGetWithResponse(
			ctx,
			taskUUID,
			&rest.V1TaskGetParams{},
		)

		if err != nil {
			v.debugLogger.Log("API error: %v", err)
			return singleTaskMsg{
				task: nil,
				err:  fmt.Errorf("failed to fetch task: %w", err),
			}
		}

		if response.JSON200 == nil {
			v.debugLogger.Log("Unexpected API response: status=%d", response.StatusCode())
			return singleTaskMsg{
				task: nil,
				err:  fmt.Errorf("unexpected response from API: status %d", response.StatusCode()),
			}
		}

		v.debugLogger.Log("API response: task=%s, status=%s", response.JSON200.DisplayName, response.JSON200.Status)

		return singleTaskMsg{
			task: response.JSON200,
			err:  nil,
		}
	}
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// fetchEvents fetches the events for the task from the API
func (v *SingleTaskView) fetchEvents() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		v.debugLogger.Log("Fetching events for task: %s", v.taskID)

		// Parse task ID as UUID
		taskUUID, err := uuid.Parse(v.taskID)
		if err != nil {
			v.debugLogger.Log("Failed to parse task UUID: %v", err)
			return eventsMsg{
				events: nil,
				err:    fmt.Errorf("invalid task ID: %w", err),
			}
		}

		v.debugLogger.Log("Parsed task UUID successfully: %s", taskUUID.String())

		// Call the API to get task events
		// Note: Limit and Offset are required by the API (not optional)
		limit := int64(1000)
		offset := int64(0)
		v.debugLogger.Log("API call: V1TaskEventListWithResponse(task=%s, limit=%d, offset=%d)", taskUUID.String(), limit, offset)
		v.debugLogger.Log("Expected API URL: /api/v1/stable/tasks/%s/task-events?limit=%d&offset=%d", taskUUID.String(), limit, offset)

		// Convert google UUID to openapi_types.UUID (they're the same underlying type)
		response, err := v.Ctx.Client.API().V1TaskEventListWithResponse(
			ctx,
			taskUUID, // uuid.UUID is compatible with openapi_types.UUID
			&rest.V1TaskEventListParams{
				Limit:  &limit,
				Offset: &offset,
			},
		)

		if err != nil {
			v.debugLogger.Log("API error: %v", err)
			return eventsMsg{
				events: nil,
				err:    fmt.Errorf("failed to fetch events: %w", err),
			}
		}

		v.debugLogger.Log("API response status: %d", response.StatusCode())
		v.debugLogger.Log("API response body (first 500 chars): %s", string(response.Body[:minInt(500, len(response.Body))]))

		if response.JSON200 == nil {
			v.debugLogger.Log("Unexpected API response: status=%d, JSON200 is nil", response.StatusCode())
			v.debugLogger.Log("Full response body: %s", string(response.Body))
			return eventsMsg{
				events: nil,
				err:    fmt.Errorf("unexpected response from API: status %d", response.StatusCode()),
			}
		}

		v.debugLogger.Log("API response: JSON200=%v", response.JSON200 != nil)
		v.debugLogger.Log("API response: Rows=%v", response.JSON200.Rows != nil)
		v.debugLogger.Log("API response: Pagination=%+v", response.JSON200.Pagination)

		events := []rest.V1TaskEvent{}
		if response.JSON200.Rows != nil {
			events = *response.JSON200.Rows
			v.debugLogger.Log("API response: extracted %d events from Rows", len(events))
		} else {
			v.debugLogger.Log("API response: Rows is nil, no events")
		}

		v.debugLogger.Log("API response: returning %d events", len(events))

		return eventsMsg{
			events: events,
			err:    nil,
		}
	}
}

// fetchLogs fetches the log lines for the task from the API
func (v *SingleTaskView) fetchLogs() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		v.debugLogger.Log("Fetching logs for task: %s", v.taskID)

		// Parse task ID as UUID
		taskUUID, err := uuid.Parse(v.taskID)
		if err != nil {
			v.debugLogger.Log("Failed to parse task UUID: %v", err)
			return logsMsg{
				logs: nil,
				err:  fmt.Errorf("invalid task ID: %w", err),
			}
		}

		// Call the API to get log lines
		v.debugLogger.Log("API call: V1LogLineListWithResponse(task=%s)", taskUUID.String())
		response, err := v.Ctx.Client.API().V1LogLineListWithResponse(
			ctx,
			taskUUID,
			&rest.V1LogLineListParams{},
		)

		if err != nil {
			v.debugLogger.Log("API error: %v", err)
			return logsMsg{
				logs: nil,
				err:  fmt.Errorf("failed to fetch logs: %w", err),
			}
		}

		if response.JSON200 == nil {
			v.debugLogger.Log("Unexpected API response: status=%d", response.StatusCode())
			return logsMsg{
				logs: nil,
				err:  fmt.Errorf("unexpected response from API: status %d", response.StatusCode()),
			}
		}

		logs := []rest.V1LogLine{}
		if response.JSON200.Rows != nil {
			logs = *response.JSON200.Rows
		}

		v.debugLogger.Log("API response: %d log lines", len(logs))

		return logsMsg{
			logs: logs,
			err:  nil,
		}
	}
}
