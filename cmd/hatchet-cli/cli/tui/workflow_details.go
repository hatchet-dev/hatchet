package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// WorkflowDetailsView displays details about a specific workflow and its recent runs
type WorkflowDetailsView struct {
	lastFetch   time.Time
	table       *TableWithStyleFunc
	workflow    *rest.Workflow
	debugLogger *DebugLogger
	workflowID  string
	tasks       []rest.V1TaskSummary
	BaseModel
	currentOffset int64
	pageSize      int64
	totalCount    int
	loading       bool
	showDebug     bool
	hasMore       bool
}

// workflowDetailsMsg contains the fetched workflow details
type workflowDetailsMsg struct {
	workflow  *rest.Workflow
	err       error
	debugInfo string
}

// workflowRunsMsg contains the fetched runs for this workflow
type workflowRunsMsg struct {
	err        error
	debugInfo  string
	tasks      []rest.V1TaskSummary
	totalCount int
	hasMore    bool
}

// workflowDetailsTickMsg is sent periodically to refresh the data
type workflowDetailsTickMsg time.Time

// NewWorkflowDetailsView creates a new workflow details view
func NewWorkflowDetailsView(ctx ViewContext, workflowID string) *WorkflowDetailsView {
	v := &WorkflowDetailsView{
		BaseModel: BaseModel{
			Ctx: ctx,
		},
		workflowID:    workflowID,
		loading:       false,
		debugLogger:   NewDebugLogger(5000),
		showDebug:     false,
		currentOffset: 0,
		pageSize:      50,
		hasMore:       false,
		totalCount:    0,
	}

	v.debugLogger.Log("WorkflowDetailsView initialized for workflow %s", workflowID)

	// Create columns for runs table (matching runs list view exactly)
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
	s.Cell = lipgloss.NewStyle()
	t.SetStyles(s)

	// Set StyleFunc for per-cell styling (column 1 is status)
	t.SetStyleFunc(func(row, col int) lipgloss.Style {
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
func (v *WorkflowDetailsView) Init() tea.Cmd {
	// Start fetching workflow details and runs immediately
	return tea.Batch(v.fetchWorkflowDetails(), v.fetchWorkflowRuns(), workflowDetailsTick())
}

// Update handles messages and updates the view state
func (v *WorkflowDetailsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.SetSize(msg.Width, msg.Height)
		v.table.SetHeight(msg.Height - 16)
		return v, nil

	case tea.KeyMsg:
		// If debug view is showing, try handling debug-specific keys first
		if v.showDebug {
			if handled, cmd := HandleDebugKeyboard(v.debugLogger, msg.String()); handled {
				return v, cmd
			}
		}

		switch msg.String() {
		case "esc", "backspace":
			// Navigate back to workflows list
			v.debugLogger.Log("Navigating back to workflows list")
			return v, NewNavigateBackMsg()
		case "r":
			// Refresh the data
			v.debugLogger.Log("Manual refresh triggered")
			v.currentOffset = 0
			v.loading = true
			return v, tea.Batch(v.fetchWorkflowDetails(), v.fetchWorkflowRuns())
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
		case "right":
			// Next page
			if v.hasMore && !v.loading {
				v.currentOffset += v.pageSize
				v.debugLogger.Log("Loading next page, offset=%d", v.currentOffset)
				v.loading = true
				return v, v.fetchWorkflowRuns()
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
				return v, v.fetchWorkflowRuns()
			}
			return v, nil
		case "enter":
			// Navigate to selected run details with detection
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

	case workflowDetailsTickMsg:
		// Auto-refresh every 5 seconds
		return v, tea.Batch(v.fetchWorkflowDetails(), v.fetchWorkflowRuns(), workflowDetailsTick())

	case workflowDetailsMsg:
		v.loading = false
		if msg.err != nil {
			v.HandleError(msg.err)
			v.debugLogger.Log("Error fetching workflow details: %v", msg.err)
		} else {
			v.workflow = msg.workflow
			v.ClearError()
			v.debugLogger.Log("Successfully fetched workflow details")
		}
		if msg.debugInfo != "" {
			v.debugLogger.Log("API: %s", msg.debugInfo)
		}
		return v, nil

	case workflowRunsMsg:
		v.loading = false
		if msg.err != nil {
			v.HandleError(msg.err)
			v.debugLogger.Log("Error fetching workflow runs: %v", msg.err)
		} else {
			v.tasks = msg.tasks
			v.hasMore = msg.hasMore
			v.totalCount = msg.totalCount
			v.updateTableRows()
			v.lastFetch = time.Now()
			v.ClearError()
			v.debugLogger.Log("Successfully fetched %d runs (offset=%d, hasMore=%v, total=%d)",
				len(msg.tasks), v.currentOffset, v.hasMore, v.totalCount)
		}
		if msg.debugInfo != "" {
			v.debugLogger.Log("Runs API: %s", msg.debugInfo)
		}
		return v, nil
	}

	// Handle mouse events for table scrolling
	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		if mouseMsg.Action == tea.MouseActionPress {
			switch mouseMsg.Button {
			case tea.MouseButtonWheelUp:
				if v.table.Cursor() > 0 {
					upMsg := tea.KeyMsg{Type: tea.KeyUp}
					_, cmd = v.table.Update(upMsg)
					return v, cmd
				}
			case tea.MouseButtonWheelDown:
				if v.table.Cursor() < len(v.tasks)-1 {
					downMsg := tea.KeyMsg{Type: tea.KeyDown}
					_, cmd = v.table.Update(downMsg)
					return v, cmd
				}
			}
		}
	}

	// Update the table model
	_, cmd = v.table.Update(msg)
	return v, cmd
}

// View renders the view to a string
func (v *WorkflowDetailsView) View() string {
	if v.Width == 0 {
		return "Initializing..."
	}

	// If debug view is enabled, show debug overlay
	if v.showDebug {
		return v.renderDebugView()
	}

	var b strings.Builder

	// Header with workflow name
	title := "Workflow Details"
	if v.workflow != nil {
		title = fmt.Sprintf("Workflow Details: %s", v.workflow.Name)
	}
	header := RenderHeader(title, v.Ctx.ProfileName, v.Width)
	b.WriteString(header)
	b.WriteString("\n\n")

	// Workflow info section
	if v.workflow != nil {
		b.WriteString(v.renderWorkflowInfo())
		b.WriteString("\n")
	}

	// Runs section header
	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.AccentColor).
		Padding(0, 1)
	b.WriteString(sectionStyle.Render("Recent Runs"))
	b.WriteString("\n\n")

	// Loading indicator
	if v.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(styles.AccentColor).
			Padding(0, 1)
		b.WriteString(loadingStyle.Render("Loading..."))
		b.WriteString("\n")
	}

	// Pagination info
	currentPage := (v.currentOffset / v.pageSize) + 1
	paginationText := ""
	if len(v.tasks) > 0 {
		switch {
		case v.totalCount > 0:
			totalPages := (int64(v.totalCount) + v.pageSize - 1) / v.pageSize
			paginationText = fmt.Sprintf("Page %d/%d", currentPage, totalPages)
		case v.hasMore:
			paginationText = fmt.Sprintf("Page %d (more available)", currentPage)
		case currentPage > 1 || v.currentOffset > 0:
			paginationText = fmt.Sprintf("Page %d", currentPage)
		}
	}

	if paginationText != "" {
		paginationStyle := lipgloss.NewStyle().
			Foreground(styles.MutedColor).
			Padding(0, 1)
		b.WriteString(paginationStyle.Render(paginationText))
		b.WriteString("\n")
	}

	// Table
	b.WriteString(v.table.View())
	b.WriteString("\n\n")

	// Error display
	if v.Err != nil {
		b.WriteString(RenderError(fmt.Sprintf("Error: %v", v.Err), v.Width))
		b.WriteString("\n")
	}

	// Last fetch timestamp
	if !v.lastFetch.IsZero() {
		lastFetchStyle := lipgloss.NewStyle().
			Foreground(styles.MutedColor).
			Padding(0, 1)
		b.WriteString(lastFetchStyle.Render(fmt.Sprintf("Last updated: %s", v.lastFetch.Format("15:04:05"))))
		b.WriteString("\n")
	}

	// Footer with controls
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
	controlItems = append(controlItems, "r: Refresh", "d: Debug", "h: Help", "shift+p: Profile", "esc: Back", "q: Quit")
	controls := RenderFooter(controlItems, v.Width)
	b.WriteString(controls)

	return b.String()
}

// SetSize updates the view dimensions
func (v *WorkflowDetailsView) SetSize(width, height int) {
	v.BaseModel.SetSize(width, height)
	if height > 16 {
		v.table.SetHeight(height - 16)
	}
}

// renderWorkflowInfo renders the workflow information section
func (v *WorkflowDetailsView) renderWorkflowInfo() string {
	if v.workflow == nil {
		return ""
	}

	var b strings.Builder

	infoStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(v.Width - 4)

	// Status and version on one line
	var statusParts []string

	// Status
	status := "Active"
	statusColor := styles.StatusSuccessColor
	if v.workflow.IsPaused != nil && *v.workflow.IsPaused {
		status = "Paused"
		statusColor = styles.StatusInProgressColor
	}
	statusStyle := lipgloss.NewStyle().Foreground(statusColor).Bold(true)
	statusParts = append(statusParts, fmt.Sprintf("Status: %s", statusStyle.Render(status)))

	// Updated timestamp
	if v.workflow.Versions != nil && len(*v.workflow.Versions) > 0 {
		updatedAt := (*v.workflow.Versions)[0].Metadata.UpdatedAt
		updatedStyle := lipgloss.NewStyle().Foreground(styles.MutedColor)
		statusParts = append(statusParts, fmt.Sprintf("Updated: %s", updatedStyle.Render(formatRelativeTime(updatedAt))))
	}

	b.WriteString(infoStyle.Render(strings.Join(statusParts, "  |  ")))
	b.WriteString("\n")

	// Tags (if any)
	if v.workflow.Tags != nil && len(*v.workflow.Tags) > 0 {
		var tagStrings []string
		for _, tag := range *v.workflow.Tags {
			tagStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(tag.Color)).
				Bold(true)
			tagStrings = append(tagStrings, tagStyle.Render(tag.Name))
		}
		tagsLine := fmt.Sprintf("Tags: %s", strings.Join(tagStrings, ", "))
		b.WriteString(infoStyle.Render(tagsLine))
		b.WriteString("\n")
	}

	// Description (if any)
	if v.workflow.Description != nil && *v.workflow.Description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(styles.MutedColor).
			Padding(0, 1).
			Width(v.Width - 4)
		b.WriteString(descStyle.Render(*v.workflow.Description))
		b.WriteString("\n")
	}

	return b.String()
}

// fetchWorkflowDetails fetches workflow details from the API
func (v *WorkflowDetailsView) fetchWorkflowDetails() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Parse workflow ID as UUID
		workflowUUID, err := uuid.Parse(v.workflowID)
		if err != nil {
			return workflowDetailsMsg{
				workflow: nil,
				err:      fmt.Errorf("invalid workflow ID: %w", err),
			}
		}

		debugReq := fmt.Sprintf("Request: workflow=%s", workflowUUID.String())

		// Call the API to get workflow details
		response, err := v.Ctx.Client.API().WorkflowGetWithResponse(
			ctx,
			workflowUUID,
		)

		if err != nil {
			return workflowDetailsMsg{
				workflow:  nil,
				err:       fmt.Errorf("failed to fetch workflow: %w", err),
				debugInfo: debugReq + " | Error: " + err.Error(),
			}
		}

		if response.JSON200 == nil {
			bodyStr := ""
			if response.Body != nil {
				bodyStr = string(response.Body)
			}
			return workflowDetailsMsg{
				workflow:  nil,
				err:       fmt.Errorf("unexpected response from API: status %d, body: %s", response.StatusCode(), bodyStr),
				debugInfo: fmt.Sprintf("Status: %d", response.StatusCode()),
			}
		}

		debugInfo := debugReq + fmt.Sprintf(" | Response: workflow=%s", response.JSON200.Name)

		return workflowDetailsMsg{
			workflow:  response.JSON200,
			err:       nil,
			debugInfo: debugInfo,
		}
	}
}

// fetchWorkflowRuns fetches recent runs for this workflow from the API
func (v *WorkflowDetailsView) fetchWorkflowRuns() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Parse tenant ID and workflow ID as UUID
		tenantUUID, err := uuid.Parse(v.Ctx.Client.TenantId())
		if err != nil {
			return workflowRunsMsg{
				tasks: nil,
				err:   fmt.Errorf("invalid tenant ID: %w", err),
			}
		}

		workflowUUID, err := uuid.Parse(v.workflowID)
		if err != nil {
			return workflowRunsMsg{
				tasks: nil,
				err:   fmt.Errorf("invalid workflow ID: %w", err),
			}
		}

		// Build params to filter by this workflow
		// Default to last 7 days of runs
		since := time.Now().Add(-7 * 24 * time.Hour)
		workflowIDs := []openapi_types.UUID{workflowUUID}
		params := &rest.V1WorkflowRunListParams{
			Offset:      int64Ptr(v.currentOffset),
			Limit:       int64Ptr(v.pageSize),
			Since:       since,
			OnlyTasks:   false,
			WorkflowIds: &workflowIDs,
		}

		debugReq := fmt.Sprintf("Request: tenant=%s, workflow=%s, offset=%d, limit=%d, since=%s, workflowIds=%v",
			tenantUUID.String(), workflowUUID.String(), v.currentOffset, v.pageSize, since.Format("2006-01-02 15:04:05"), workflowIDs)

		// Call the API to list workflow runs for this workflow
		response, err := v.Ctx.Client.API().V1WorkflowRunListWithResponse(
			ctx,
			tenantUUID,
			params,
		)

		if err != nil {
			return workflowRunsMsg{
				tasks:     nil,
				err:       fmt.Errorf("failed to fetch runs: %w", err),
				debugInfo: debugReq + " | Error: " + err.Error(),
			}
		}

		if response.JSON200 == nil {
			bodyStr := ""
			if response.Body != nil {
				bodyStr = string(response.Body)
			}
			return workflowRunsMsg{
				tasks:     nil,
				err:       fmt.Errorf("unexpected response from API: status %d, body: %s", response.StatusCode(), bodyStr),
				debugInfo: fmt.Sprintf("Status: %d", response.StatusCode()),
			}
		}

		tasks := response.JSON200.Rows

		// Calculate pagination info
		hasMore := false
		totalCount := 0
		if response.JSON200.Pagination.NextPage != nil {
			hasMore = true
		}
		if response.JSON200.Pagination.NumPages != nil {
			numPages := *response.JSON200.Pagination.NumPages
			totalCount = int(numPages * v.pageSize)
		}

		debugInfo := debugReq + fmt.Sprintf(" | Response: rows=%d, hasMore=%v, total=%d",
			len(tasks), hasMore, totalCount)

		return workflowRunsMsg{
			tasks:      tasks,
			err:        nil,
			debugInfo:  debugInfo,
			hasMore:    hasMore,
			totalCount: totalCount,
		}
	}
}

// updateTableRows updates the table rows based on current tasks
func (v *WorkflowDetailsView) updateTableRows() {
	rows := make([]table.Row, len(v.tasks))

	for i, task := range v.tasks {
		// Task Name
		taskName := task.DisplayName

		// Status - use plain text, StyleFunc will apply colors
		statusStyle := GetV1TaskStatusStyle(task.Status)
		status := statusStyle.Text

		// Workflow (matches runs_list.go structure)
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

// workflowDetailsTick returns a command that sends a tick message after a delay
func workflowDetailsTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return workflowDetailsTickMsg(t)
	})
}

// renderDebugView renders the debug log overlay using the shared component
func (v *WorkflowDetailsView) renderDebugView() string {
	return RenderDebugView(v.debugLogger, v.Width, v.Height, "")
}
