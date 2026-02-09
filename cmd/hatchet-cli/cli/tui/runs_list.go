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
	lastFetch      time.Time
	filterStatuses *[]rest.V1TaskStatus
	newFilters     *RunsListFilters
	filters        *RunsListFilters
	debugLogger    *DebugLogger
	filterForm     *huh.Form
	table          *TableWithStyleFunc
	workflows      []WorkflowOption
	metrics        rest.V1TaskRunMetrics
	tasks          []rest.V1TaskSummary
	BaseModel
	currentOffset int64
	totalCount    int
	pageSize      int64
	loading       bool
	showingFilter bool
	hasMore       bool
	showDebug     bool
}

// tasksMsg contains the fetched tasks
type tasksMsg struct {
	err        error
	debugInfo  string
	tasks      []rest.V1TaskSummary
	totalCount int
	hasMore    bool
}

// metricsMsg contains the fetched metrics
type metricsMsg struct {
	err       error
	debugInfo string
	metrics   rest.V1TaskRunMetrics
}

// tickMsg is sent periodically to refresh the data
type tickMsg time.Time

// NewRunsListView creates a new runs list view
func NewRunsListView(ctx ViewContext) *RunsListView {
	v := &RunsListView{
		BaseModel: BaseModel{
			Ctx: ctx,
		},
		filters:       NewDefaultRunsListFilters(),
		loading:       false,
		debugLogger:   NewDebugLogger(5000), // 5000 log entries max
		showDebug:     false,
		currentOffset: 0,
		pageSize:      50,
		hasMore:       false,
		totalCount:    0,
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

				// Reset to first page when filters change
				v.currentOffset = 0
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
		// If debug view is showing, try handling debug-specific keys first
		if v.showDebug {
			if handled, cmd := HandleDebugKeyboard(v.debugLogger, msg.String()); handled {
				return v, cmd
			}
		}

		switch msg.String() {
		case "r":
			// Refresh the data and reset to first page
			v.debugLogger.Log("Manual refresh triggered")
			v.currentOffset = 0
			v.loading = true
			return v, tea.Batch(v.fetchTasks(), v.fetchMetrics())
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
		case "f":
			// Open filters modal inline
			v.debugLogger.Log("Opening filters")
			return v, v.initFiltersForm()
		case "right":
			// Next page
			if v.hasMore && !v.loading {
				v.currentOffset += v.pageSize
				v.debugLogger.Log("Loading next page, offset=%d", v.currentOffset)
				v.loading = true
				return v, v.fetchTasks()
			}
			return v, nil
		case "left":
			// Previous page
			if v.currentOffset > 0 && !v.loading {
				v.currentOffset -= v.pageSize
				if v.currentOffset < 0 {
					v.currentOffset = 0
				}
				v.debugLogger.Log("Loading previous page, offset=%d", v.currentOffset)
				v.loading = true
				return v, v.fetchTasks()
			}
			return v, nil
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
			v.hasMore = msg.hasMore
			v.totalCount = msg.totalCount
			v.updateTableRows()
			v.lastFetch = time.Now()
			v.ClearError()
			v.debugLogger.Log("Successfully fetched %d tasks (offset=%d, hasMore=%v, total=%d)",
				len(msg.tasks), v.currentOffset, v.hasMore, v.totalCount)
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

	// Handle mouse events for table scrolling
	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		// Don't log mouse events to avoid infinite scroll in debug view
		// v.debugLogger.Log("Mouse event: action=%v, button=%v", mouseMsg.Action, mouseMsg.Button)

		// Handle mouse wheel scrolling using new API
		if mouseMsg.Action == tea.MouseActionPress {
			switch mouseMsg.Button {
			case tea.MouseButtonWheelUp:
				// Move cursor up
				if v.table.Cursor() > 0 {
					// Create a KeyMsg for up arrow
					upMsg := tea.KeyMsg{Type: tea.KeyUp}
					_, cmd = v.table.Update(upMsg)
					return v, cmd
				}
			case tea.MouseButtonWheelDown:
				// Move cursor down
				if v.table.Cursor() < len(v.tasks)-1 {
					// Create a KeyMsg for down arrow
					downMsg := tea.KeyMsg{Type: tea.KeyDown}
					_, cmd = v.table.Update(downMsg)
					return v, cmd
				}
			}
		}
	}

	// Update the table model (handles keyboard and other events)
	_, cmd = v.table.Update(msg)
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

	// Header with logo and view indicator (using reusable component)
	header := RenderHeaderWithViewIndicator("Runs", v.Ctx.ProfileName, v.Width)

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

	// Pagination info
	currentPage := (v.currentOffset / v.pageSize) + 1
	paginationText := ""

	// Only show pagination if there are actually tasks
	if len(v.tasks) > 0 {
		switch {
		case v.totalCount > 0:
			totalPages := (int64(v.totalCount) + v.pageSize - 1) / v.pageSize
			paginationText = fmt.Sprintf("Page %d/%d", currentPage, totalPages)
			// Only show "more available" if we're not on the last page
			if v.hasMore && currentPage < totalPages {
				paginationText += " (more available)"
			}
		case v.hasMore:
			paginationText = fmt.Sprintf("Page %d (more available)", currentPage)
		case currentPage > 1 || v.currentOffset > 0:
			// Show page number if we're not on first page
			paginationText = fmt.Sprintf("Page %d", currentPage)
		}
	}

	paginationStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		Padding(0, 1)

	pagination := ""
	if paginationText != "" {
		pagination = paginationStyle.Render(paginationText)
	}

	// Footer with controls (using reusable component)
	controlItems := []string{
		"↑/↓: Navigate",
		"enter: View Details",
	}
	if v.currentOffset > 0 {
		controlItems = append(controlItems, "←: Prev Page")
	}
	if v.hasMore {
		controlItems = append(controlItems, "→: Next Page")
	}
	controlItems = append(controlItems, "f: Filters", "r: Refresh", "d: Debug", "h: Help", "v: Switch View", "shift+p: Profile", "q: Quit")
	controls := RenderFooter(controlItems, v.Width)

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
	if pagination != "" {
		b.WriteString("  ")
		b.WriteString(pagination)
	}
	b.WriteString("\n\n")
	b.WriteString(v.table.View())
	b.WriteString("\n\n")

	if v.Err != nil {
		b.WriteString(RenderError(fmt.Sprintf("Error: %v", v.Err), v.Width))
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
			Offset:    int64Ptr(v.currentOffset),
			Limit:     int64Ptr(v.pageSize),
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
		debugReq := fmt.Sprintf("Request: tenant=%s, offset=%d, limit=%d, since=%s, until=%v, workflows=%d, statuses=%d",
			tenantUUID.String(), v.currentOffset, v.pageSize, v.filters.Since.Format("2006-01-02 15:04:05"),
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

		// Calculate pagination info
		hasMore := false
		totalCount := 0
		currentPage := int64(0)
		numPages := int64(0)

		if response.JSON200.Pagination.NextPage != nil {
			hasMore = true
		}
		if response.JSON200.Pagination.CurrentPage != nil {
			currentPage = *response.JSON200.Pagination.CurrentPage
		}
		if response.JSON200.Pagination.NumPages != nil {
			numPages = *response.JSON200.Pagination.NumPages
			// Estimate total count based on pages
			totalCount = int(numPages * v.pageSize)
		}

		// Debug: combine request and response info
		debugInfo := debugReq + fmt.Sprintf(" | Response: rows=%d, hasMore=%v, totalCount=%d, page=%d/%d",
			len(tasks), hasMore, totalCount, currentPage, numPages)

		return tasksMsg{
			tasks:      tasks,
			err:        nil,
			debugInfo:  debugInfo,
			hasMore:    hasMore,
			totalCount: totalCount,
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

// renderDebugView renders the debug log overlay using the shared component
func (v *RunsListView) renderDebugView() string {
	// Build view-specific context info
	activeStatuses := []string{}
	for status, enabled := range v.filters.Statuses {
		if enabled {
			activeStatuses = append(activeStatuses, string(status))
		}
	}

	return RenderDebugView(v.debugLogger, v.Width, v.Height, "")
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
