package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// RunsListView displays a list of runs in a table
type RunsListView struct {
	BaseModel
	table          *TableWithStyleFunc
	tasks          []rest.V1TaskSummary
	metrics        rest.V1TaskRunMetrics // Metrics matching current filters
	workflows      []WorkflowOption
	filters        *RunsListFilters
	loading        bool
	lastFetch      time.Time
	debugLogger    *DebugLogger
	showDebug      bool                 // Whether to show debug overlay
	showingFilter  bool                 // Whether filter modal is open
	filterForm     *huh.Form            // The filter form when modal is open
	newFilters     *RunsListFilters     // Temp storage for filter edits
	filterStatuses *[]rest.V1TaskStatus // Status slice being edited by multiselect
}

// tasksMsg contains the fetched tasks
type tasksMsg struct {
	tasks     []rest.V1TaskSummary
	err       error
	debugInfo string
}

// metricsMsg contains the fetched metrics
type metricsMsg struct {
	metrics   rest.V1TaskRunMetrics
	err       error
	debugInfo string
}

// tickMsg is sent periodically to refresh the data
type tickMsg time.Time

// NewRunsListView creates a new runs list view
func NewRunsListView(ctx ViewContext) *RunsListView {
	v := &RunsListView{
		BaseModel: BaseModel{
			Ctx: ctx,
		},
		filters:     NewDefaultRunsListFilters(),
		loading:     false,
		debugLogger: NewDebugLogger(5000), // 5000 log entries max
		showDebug:   false,
	}

	v.debugLogger.Log("TasksView initialized, filters=%+v", v.filters)

	// Create columns matching the frontend
	columns := []table.Column{
		{Title: "Task Name", Width: 30},
		{Title: "Status", Width: 12},
		{Title: "Workflow", Width: 25},
		{Title: "Created At", Width: 16},
		{Title: "Started At", Width: 16},
		{Title: "Duration", Width: 12},
	}

	// Create the table with StyleFunc support
	t := NewTableWithStyleFunc(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	// Apply Hatchet theme colors to the table
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.AccentColor).
		BorderBottom(true).
		Bold(true).
		Foreground(styles.AccentColor)
	s.Selected = s.Selected.
		Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#0A1029"}).
		Background(styles.Blue).
		Bold(true)
	s.Cell = lipgloss.NewStyle() // Empty style to preserve our custom cell styling
	t.SetStyles(s)

	// Set StyleFunc for per-cell styling (column 1 is status)
	t.SetStyleFunc(func(row, col int) lipgloss.Style {
		// Column 1 is the status column
		if col == 1 && row < len(v.tasks) {
			statusStyle := GetV1TaskStatusStyle(v.tasks[row].Status)
			return lipgloss.NewStyle().Foreground(statusStyle.Foreground)
		}
		return lipgloss.NewStyle()
	})

	v.table = t

	return v
}

// Init initializes the view
func (v *RunsListView) Init() tea.Cmd {
	// Start fetching data and metrics immediately
	return tea.Batch(v.fetchTasks(), v.fetchMetrics(), tick())
}

// Update handles messages and updates the view state
func (v *RunsListView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmd tea.Cmd

	// If showing filter modal, delegate ALL messages to the form first
	if v.showingFilter && v.filterForm != nil {
		form, cmd := v.filterForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			v.filterForm = f

			// Check if form is complete
			if v.filterForm.State == huh.StateCompleted {
				// Sync status slice back to status map
				v.newFilters.Statuses = make(map[rest.V1TaskStatus]bool)
				for _, status := range *v.filterStatuses {
					v.newFilters.Statuses[status] = true
				}

				// Apply filters
				v.showingFilter = false
				v.filters = v.newFilters

				// Update time range based on window
				if v.filters.TimeWindow != "custom" {
					v.filters.Since = GetTimeRangeFromWindow(v.filters.TimeWindow)
					v.filters.Until = nil
				}

				v.loading = true
				return v, tea.Batch(cmd, v.fetchTasks(), v.fetchMetrics())
			}

			// Check for ESC to cancel
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				if keyMsg.String() == "esc" {
					v.showingFilter = false
					v.filterForm = nil
					return v, nil
				}
			}
		}
		return v, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.SetSize(msg.Width, msg.Height)
		v.table.SetHeight(msg.Height - 12)
		return v, nil

	case tea.KeyMsg:

		switch msg.String() {
		case "r":
			// Refresh the data
			v.debugLogger.Log("Manual refresh triggered")
			v.loading = true
			return v, tea.Batch(v.fetchTasks(), v.fetchMetrics())
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
		case "f":
			// Open filters modal inline
			v.debugLogger.Log("Opening filters")
			return v, v.initFiltersForm()
		case "enter":
			// Navigate to selected task's workflow run details with detection
			if len(v.tasks) > 0 {
				selectedIdx := v.table.Cursor()
				if selectedIdx >= 0 && selectedIdx < len(v.tasks) {
					task := v.tasks[selectedIdx]
					runID := task.Metadata.Id
					v.debugLogger.Log("Navigating to run with detection: %s", runID)
					return v, NewNavigateToRunWithDetectionMsg(runID)
				}
			}
			return v, nil
		}

	case tickMsg:
		// Don't auto-refresh if filter modal is open
		if v.showingFilter {
			return v, tick()
		}
		// Auto-refresh every 5 seconds
		return v, tea.Batch(v.fetchTasks(), v.fetchMetrics(), tick())

	case tasksMsg:
		v.loading = false
		if msg.err != nil {
			v.HandleError(msg.err)
			v.debugLogger.Log("Error fetching tasks: %v", msg.err)
		} else {
			v.tasks = msg.tasks
			v.updateTableRows()
			v.lastFetch = time.Now()
			v.ClearError()
			v.debugLogger.Log("Successfully fetched %d tasks", len(msg.tasks))
		}
		// Log debug info if available
		if msg.debugInfo != "" {
			v.debugLogger.Log("API: %s", msg.debugInfo)
		}
		return v, nil

	case metricsMsg:
		if msg.err != nil {
			v.debugLogger.Log("Error fetching metrics: %v", msg.err)
		} else {
			v.metrics = msg.metrics
			v.debugLogger.Log("Metrics updated: %d status counts", len(msg.metrics))
		}
		if msg.debugInfo != "" {
			v.debugLogger.Log("Metrics API: %s", msg.debugInfo)
		}
		return v, nil

	case filtersReadyMsg:
		// Workflows are ready, build the form
		v.workflows = msg.workflows
		v.showingFilter = true

		// Create a copy of current filters for editing
		v.newFilters = &RunsListFilters{
			WorkflowIDs: append([]string{}, v.filters.WorkflowIDs...), // Copy slice
			Statuses:    make(map[rest.V1TaskStatus]bool),
			TimeWindow:  v.filters.TimeWindow,
			Since:       v.filters.Since,
			Until:       v.filters.Until,
		}
		for k, enabled := range v.filters.Statuses {
			v.newFilters.Statuses[k] = enabled
		}

		// Build the form using the helper from tasks_filters.go
		v.filterForm, v.filterStatuses = BuildRunsListFiltersForm(v.newFilters, v.workflows)
		return v, v.filterForm.Init()

	case filtersClosedMsg:
		v.showingFilter = false
		if !msg.cancelled {
			// Update filters and refresh
			v.filters = msg.newFilters
			v.loading = true
			return v, tea.Batch(v.fetchTasks(), v.fetchMetrics())
		}
		return v, nil
	}

	// Don't update table if filter modal is open
	if v.showingFilter {
		return v, nil
	}

	// Update the embedded table model
	updatedModel, cmd := v.table.Update(msg)
	v.table.Model = &updatedModel
	return v, cmd
}

// View renders the view to a string
func (v *RunsListView) View() string {
	if v.Width == 0 {
		return "Initializing..."
	}

	// If filter modal is open, show the form with header and instructions
	if v.showingFilter && v.filterForm != nil {
		return v.renderFilterModal()
	}

	// If debug view is enabled, show debug overlay
	if v.showDebug {
		return v.renderDebugView()
	}

	// Header with logo (using reusable component)
	header := RenderHeader("Hatchet Tasks", v.Ctx.ProfileName, v.Width)

	// Active filters display
	filterSummary := v.renderActiveFilters()

	// Stats bar - use metrics API counts instead of counting from current page
	statsStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		Padding(0, 1)

	// Get counts from metrics (matching filters, not just current page)
	completed := 0
	running := 0
	queued := 0
	failed := 0
	cancelled := 0
	total := 0

	for _, metric := range v.metrics {
		switch metric.Status {
		case rest.V1TaskStatusCOMPLETED:
			completed = metric.Count
		case rest.V1TaskStatusRUNNING:
			running = metric.Count
		case rest.V1TaskStatusQUEUED:
			queued = metric.Count
		case rest.V1TaskStatusFAILED:
			failed = metric.Count
		case rest.V1TaskStatusCANCELLED:
			cancelled = metric.Count
		}
		total += metric.Count
	}

	stats := statsStyle.Render(fmt.Sprintf(
		"Total: %d  |  Completed: %d  |  Running: %d  |  Queued: %d  |  Failed: %d  |  Cancelled: %d",
		total, completed, running, queued, failed, cancelled,
	))

	// Loading indicator
	loadingText := ""
	if v.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(styles.AccentColor).
			Padding(0, 1)
		loadingText = loadingStyle.Render("Loading...")
	}

	// Footer with controls (using reusable component)
	controls := RenderFooter([]string{
		"↑/↓: Navigate",
		"enter: View Details",
		"f: Filters",
		"r: Refresh",
		"d: Debug",
		"q: Quit",
	}, v.Width)

	// Build the full view
	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n\n")
	if filterSummary != "" {
		b.WriteString(filterSummary)
		b.WriteString("\n")
	}
	b.WriteString(stats)
	if loadingText != "" {
		b.WriteString("  ")
		b.WriteString(loadingText)
	}
	b.WriteString("\n\n")
	b.WriteString(v.table.View())
	b.WriteString("\n\n")

	if v.Err != nil {
		b.WriteString(RenderError(fmt.Sprintf("Error: %v", v.Err)))
		b.WriteString("\n")
	}

	if !v.lastFetch.IsZero() {
		lastFetchStyle := lipgloss.NewStyle().
			Foreground(styles.MutedColor).
			Padding(0, 1)
		b.WriteString(lastFetchStyle.Render(fmt.Sprintf("Last updated: %s", v.lastFetch.Format("15:04:05"))))
		b.WriteString("\n")
	}

	b.WriteString(controls)

	return b.String()
}

// SetSize updates the view dimensions
func (v *RunsListView) SetSize(width, height int) {
	v.BaseModel.SetSize(width, height)
	if height > 12 {
		v.table.SetHeight(height - 12)
	}
}

// renderActiveFilters renders a summary of the currently active filters
func (v *RunsListView) renderActiveFilters() string {
	parts := []string{}

	// Workflows filter
	if len(v.filters.WorkflowIDs) > 0 {
		workflowNames := []string{}
		for _, wfID := range v.filters.WorkflowIDs {
			// Find the workflow name from cached workflows
			name := wfID // Default to ID
			for _, wf := range v.workflows {
				if wf.ID == wfID {
					name = wf.DisplayName
					break
				}
			}
			workflowNames = append(workflowNames, name)
		}
		if len(workflowNames) > 3 {
			parts = append(parts, fmt.Sprintf("Workflows: %s +%d more",
				strings.Join(workflowNames[:3], ", "),
				len(workflowNames)-3))
		} else {
			parts = append(parts, fmt.Sprintf("Workflows: %s", strings.Join(workflowNames, ", ")))
		}
	}

	// Status filter - only show if not all statuses are selected
	activeStatuses := []string{}
	for status, enabled := range v.filters.Statuses {
		if enabled {
			activeStatuses = append(activeStatuses, string(status))
		}
	}
	if len(activeStatuses) > 0 && len(activeStatuses) < 5 {
		parts = append(parts, fmt.Sprintf("Status: %s", strings.Join(activeStatuses, ", ")))
	}

	// Time window filter
	timeDesc := ""
	switch v.filters.TimeWindow {
	case "1h":
		timeDesc = "Last Hour"
	case "6h":
		timeDesc = "Last 6 Hours"
	case "1d":
		timeDesc = "Last 24 Hours"
	case "7d":
		timeDesc = "Last 7 Days"
	default:
		timeDesc = v.filters.TimeWindow
	}
	parts = append(parts, fmt.Sprintf("Time: %s", timeDesc))

	if len(parts) == 0 {
		return ""
	}

	filterStyle := lipgloss.NewStyle().
		Foreground(styles.AccentColor).
		Padding(0, 1)

	return filterStyle.Render("Active Filters: " + strings.Join(parts, " • "))
}

// renderFilterModal renders the filter form with header and instructions
func (v *RunsListView) renderFilterModal() string {
	var b strings.Builder

	// Header (using reusable component)
	header := RenderHeader("Filter Tasks", v.Ctx.ProfileName, v.Width)
	b.WriteString(header)
	b.WriteString("\n\n")

	// Instructions (using reusable component)
	instructions := RenderInstructions(
		"Navigate: ↑/↓ or Tab/Shift+Tab  •  Select: Space (multiselect) or Enter (single)  •  Search: Type to filter workflows",
		v.Width,
	)
	b.WriteString(instructions)
	b.WriteString("\n\n")

	// The form
	b.WriteString(v.filterForm.View())
	b.WriteString("\n")

	// Footer (using reusable component)
	footer := RenderFooter([]string{
		"Enter: Apply filters",
		"Esc: Cancel",
	}, v.Width)
	b.WriteString(footer)

	return b.String()
}

// fetchTasks fetches tasks from the API
func (v *RunsListView) fetchTasks() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Parse tenant ID as UUID
		tenantUUID, err := uuid.Parse(v.Ctx.Client.TenantId())
		if err != nil {
			return tasksMsg{
				tasks: nil,
				err:   fmt.Errorf("invalid tenant ID: %w", err),
			}
		}

		// Get active statuses from filters
		statuses := v.filters.GetActiveStatuses()

		params := &rest.V1WorkflowRunListParams{
			Offset:    int64Ptr(0),
			Limit:     int64Ptr(50),
			Since:     v.filters.Since,
			OnlyTasks: false,
			Statuses:  &statuses,
		}

		// Add workflow filters if specified
		if len(v.filters.WorkflowIDs) > 0 {
			workflowUUIDs := []openapi_types.UUID{}
			for _, wfID := range v.filters.WorkflowIDs {
				if workflowUUID, err := uuid.Parse(wfID); err == nil {
					workflowUUIDs = append(workflowUUIDs, workflowUUID)
				}
			}
			if len(workflowUUIDs) > 0 {
				params.WorkflowIds = &workflowUUIDs
			}
		}

		// Add until filter if specified
		if v.filters.Until != nil {
			params.Until = v.filters.Until
		}

		// Debug: log request parameters
		debugReq := fmt.Sprintf("Request: tenant=%s, since=%s, until=%v, workflows=%d, limit=50, statuses=%d",
			tenantUUID.String(), v.filters.Since.Format("2006-01-02 15:04:05"),
			v.filters.Until, len(v.filters.WorkflowIDs), len(statuses))

		// Call the API to list workflow runs
		// Matching the frontend query: queries.v1WorkflowRuns.list()
		response, err := v.Ctx.Client.API().V1WorkflowRunListWithResponse(
			ctx,
			tenantUUID,
			params,
		)

		if err != nil {
			return tasksMsg{
				tasks:     nil,
				err:       fmt.Errorf("failed to fetch tasks: %w", err),
				debugInfo: debugReq + " | Error: " + err.Error(),
			}
		}

		if response.JSON200 == nil {
			// Debug: log the response body
			bodyStr := ""
			if response.Body != nil {
				bodyStr = string(response.Body)
			}
			return tasksMsg{
				tasks:     nil,
				err:       fmt.Errorf("unexpected response from API: status %d, body: %s", response.StatusCode(), bodyStr),
				debugInfo: fmt.Sprintf("Status: %d", response.StatusCode()),
			}
		}

		tasks := response.JSON200.Rows

		// Debug: combine request and response info
		debugInfo := debugReq + " | Response: total_rows=" + fmt.Sprintf("%d", len(tasks))
		if response.JSON200.Pagination.NumPages != nil {
			debugInfo += fmt.Sprintf(", num_pages=%v", *response.JSON200.Pagination.NumPages)
		}
		if response.JSON200.Pagination.CurrentPage != nil {
			debugInfo += fmt.Sprintf(", current_page=%v", *response.JSON200.Pagination.CurrentPage)
		}

		return tasksMsg{
			tasks:     tasks,
			err:       nil,
			debugInfo: debugInfo,
		}
	}
}

// fetchMetrics fetches status metrics from the API based on current filters
func (v *RunsListView) fetchMetrics() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Parse tenant ID as UUID
		tenantUUID, err := uuid.Parse(v.Ctx.Client.TenantId())
		if err != nil {
			return metricsMsg{
				metrics: nil,
				err:     fmt.Errorf("invalid tenant ID: %w", err),
			}
		}

		// Build params matching the current filters (but without status filter)
		params := &rest.V1TaskListStatusMetricsParams{
			Since: v.filters.Since,
			Until: v.filters.Until,
		}

		// Add workflow filters if specified
		if len(v.filters.WorkflowIDs) > 0 {
			workflowUUIDs := []openapi_types.UUID{}
			for _, wfID := range v.filters.WorkflowIDs {
				if workflowUUID, err := uuid.Parse(wfID); err == nil {
					workflowUUIDs = append(workflowUUIDs, workflowUUID)
				}
			}
			if len(workflowUUIDs) > 0 {
				params.WorkflowIds = &workflowUUIDs
			}
		}

		// Debug: log request parameters
		debugReq := fmt.Sprintf("Metrics Request: tenant=%s, since=%s, until=%v, workflows=%d",
			tenantUUID.String(), v.filters.Since.Format("2006-01-02 15:04:05"),
			v.filters.Until, len(v.filters.WorkflowIDs))

		// Call the metrics API
		response, err := v.Ctx.Client.API().V1TaskListStatusMetricsWithResponse(
			ctx,
			tenantUUID,
			params,
		)

		if err != nil {
			return metricsMsg{
				metrics:   nil,
				err:       fmt.Errorf("failed to fetch metrics: %w", err),
				debugInfo: debugReq + " | Error: " + err.Error(),
			}
		}

		if response.JSON200 == nil {
			bodyStr := ""
			if response.Body != nil {
				bodyStr = string(response.Body)
			}
			return metricsMsg{
				metrics:   nil,
				err:       fmt.Errorf("unexpected response from metrics API: status %d, body: %s", response.StatusCode(), bodyStr),
				debugInfo: fmt.Sprintf("Status: %d", response.StatusCode()),
			}
		}

		metrics := *response.JSON200

		// Debug: response info
		debugInfo := debugReq + " | Response: " + fmt.Sprintf("%d status counts", len(metrics))

		return metricsMsg{
			metrics:   metrics,
			err:       nil,
			debugInfo: debugInfo,
		}
	}
}

// updateTableRows updates the table rows based on current tasks
func (v *RunsListView) updateTableRows() {
	rows := make([]table.Row, len(v.tasks))

	for i, task := range v.tasks {
		// Task Name
		taskName := task.DisplayName

		// Status - use plain text, StyleFunc will apply colors
		statusStyle := GetV1TaskStatusStyle(task.Status)
		status := statusStyle.Text

		// Workflow
		workflow := "N/A"
		if task.WorkflowName != nil {
			workflow = *task.WorkflowName
		}

		// Created At
		createdAt := formatRelativeTime(task.TaskInsertedAt)

		// Started At
		startedAt := "N/A"
		if task.StartedAt != nil {
			startedAt = formatRelativeTime(*task.StartedAt)
		}

		// Duration
		duration := formatTaskDuration(&task)

		rows[i] = table.Row{
			taskName,
			status,
			workflow,
			createdAt,
			startedAt,
			duration,
		}
	}

	v.table.SetRows(rows)
}

// formatTaskDuration formats the duration of a task
func formatTaskDuration(task *rest.V1TaskSummary) string {
	if task.StartedAt == nil {
		return "-"
	}

	var duration time.Duration
	switch {
	case task.FinishedAt != nil:
		duration = task.FinishedAt.Sub(*task.StartedAt)
	case task.Status == rest.V1TaskStatusRUNNING:
		duration = time.Since(*task.StartedAt)
	default:
		return "-"
	}

	return formatDuration(int(duration.Milliseconds()))
}

// formatDuration formats milliseconds into a human-readable duration
func formatDuration(ms int) string {
	duration := time.Duration(ms) * time.Millisecond

	if duration < time.Second {
		return fmt.Sprintf("%dms", ms)
	}

	seconds := duration.Seconds()
	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	}

	minutes := int(seconds / 60)
	secs := int(seconds) % 60
	return fmt.Sprintf("%dm%ds", minutes, secs)
}

// formatRelativeTime formats a time as a relative duration (e.g., "5m ago")
func formatRelativeTime(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return fmt.Sprintf("%ds ago", int(duration.Seconds()))
	}

	if duration < time.Hour {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	}

	if duration < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	}

	return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
}

// tick returns a command that sends a tick message after a delay
func tick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// renderDebugView renders the debug log overlay
func (v *RunsListView) renderDebugView() string {
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

	// Current filters info
	filtersStyle := lipgloss.NewStyle().
		Foreground(styles.AccentColor).
		Padding(0, 1).
		Width(v.Width - 4)

	activeStatuses := []string{}
	for status, enabled := range v.filters.Statuses {
		if enabled {
			activeStatuses = append(activeStatuses, string(status))
		}
	}

	filtersInfo := fmt.Sprintf(
		"Active Filters: Workflows=%d (%v), Statuses=%d (%v), Since=%s, Until=%v",
		len(v.filters.WorkflowIDs),
		v.filters.WorkflowIDs,
		len(activeStatuses),
		activeStatuses,
		v.filters.Since.Format("2006-01-02 15:04:05"),
		v.filters.Until,
	)
	filtersText := filtersStyle.Render(filtersInfo)

	// Log entries
	logStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(v.Width - 4)

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(filtersText)
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

// filtersClosedMsg is sent when the filter modal is closed
type filtersClosedMsg struct {
	newFilters *RunsListFilters
	cancelled  bool
}

// initFiltersForm initializes the filter form inline
func (v *RunsListView) initFiltersForm() tea.Cmd {
	// Fetch workflows if not cached
	if len(v.workflows) == 0 {
		return func() tea.Msg {
			ctx := context.Background()
			workflows, err := FetchWorkflows(ctx, v.Ctx.Client.API(), v.Ctx.Client.TenantId())
			if err != nil {
				v.debugLogger.Log("Error fetching workflows: %v", err)
				return filtersReadyMsg{workflows: []WorkflowOption{}}
			}
			v.debugLogger.Log("Fetched %d workflows", len(workflows))
			return filtersReadyMsg{workflows: workflows}
		}
	}

	// Workflows already cached, build form immediately
	return func() tea.Msg {
		return filtersReadyMsg{workflows: v.workflows}
	}
}

// filtersReadyMsg is sent when workflows are ready and we can build the form
type filtersReadyMsg struct {
	workflows []WorkflowOption
}

// Helper functions for pointer conversions
func int64Ptr(i int64) *int64 {
	return &i
}
