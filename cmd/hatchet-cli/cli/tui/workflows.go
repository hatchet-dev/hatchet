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

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// WorkflowsView displays a list of workflows in a table
type WorkflowsView struct {
	lastFetch   time.Time
	table       *TableWithStyleFunc
	debugLogger *DebugLogger
	searchForm  *huh.Form
	searchQuery string
	workflows   []rest.Workflow
	BaseModel
	totalPages    int
	currentOffset int
	pageSize      int
	showDebug     bool
	showingSearch bool
	hasMore       bool
	loading       bool
}

// workflowsMsg contains the fetched workflows
type workflowsMsg struct {
	err        error
	debugInfo  string
	workflows  []rest.Workflow
	totalPages int
	hasMore    bool
}

// workflowTickMsg is sent periodically to refresh the data
type workflowTickMsg time.Time

// NewWorkflowsView creates a new workflows list view
func NewWorkflowsView(ctx ViewContext) *WorkflowsView {
	v := &WorkflowsView{
		BaseModel: BaseModel{
			Ctx: ctx,
		},
		loading:       false,
		debugLogger:   NewDebugLogger(5000), // 5000 log entries max
		showDebug:     false,
		currentOffset: 0,
		pageSize:      50,
		hasMore:       false,
		totalPages:    0,
	}

	v.debugLogger.Log("WorkflowsView initialized")

	// Create columns:
	// - Name (hoverable, first column)
	// - Created At
	// - Status
	columns := []table.Column{
		{Title: "Name", Width: 40},
		{Title: "Created At", Width: 16},
		{Title: "Status", Width: 12},
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

	// Set StyleFunc for per-cell styling (column 2 is status)
	t.SetStyleFunc(func(row, col int) lipgloss.Style {
		// Column 2 is the status column
		if col == 2 && row < len(v.workflows) {
			isPaused := v.workflows[row].IsPaused != nil && *v.workflows[row].IsPaused
			if isPaused {
				// Yellow for paused (matching frontend "inProgress" variant)
				return lipgloss.NewStyle().Foreground(styles.StatusInProgressColor)
			}
			// Green for active (matching frontend "successful" variant)
			return lipgloss.NewStyle().Foreground(styles.StatusSuccessColor)
		}
		return lipgloss.NewStyle()
	})

	v.table = t

	return v
}

// Init initializes the view
func (v *WorkflowsView) Init() tea.Cmd {
	// Start fetching data immediately
	return tea.Batch(v.fetchWorkflows(), workflowTick())
}

// Update handles messages and updates the view state
func (v *WorkflowsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmd tea.Cmd

	// If showing search form, delegate ALL messages to the form first
	if v.showingSearch && v.searchForm != nil {
		form, cmd := v.searchForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			v.searchForm = f

			// Check if form is complete
			if v.searchForm.State == huh.StateCompleted {
				// Apply search
				v.showingSearch = false
				v.currentOffset = 0
				v.loading = true
				v.debugLogger.Log("Search applied: %s", v.searchQuery)
				return v, v.fetchWorkflows()
			}

			// Check for down arrow or ESC to cancel
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				if keyMsg.String() == "down" || keyMsg.String() == "esc" {
					v.showingSearch = false
					v.searchForm = nil
					v.debugLogger.Log("Search cancelled")
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
			return v, v.fetchWorkflows()
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
		case "/":
			// Open search modal
			v.debugLogger.Log("Opening search")
			v.showingSearch = true
			v.searchForm = huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Key("search").
						Title("Search workflows by name").
						Value(&v.searchQuery).
						Placeholder("Enter workflow name..."),
				),
			).WithTheme(styles.HatchetTheme())
			return v, v.searchForm.Init()
		case "x":
			// Clear search
			if v.searchQuery != "" {
				v.debugLogger.Log("Clearing search")
				v.searchQuery = ""
				v.currentOffset = 0
				v.loading = true
				return v, v.fetchWorkflows()
			}
			return v, nil
		case "right":
			// Next page
			if v.hasMore && !v.loading {
				v.currentOffset += v.pageSize
				v.debugLogger.Log("Loading next page, offset=%d", v.currentOffset)
				v.loading = true
				return v, v.fetchWorkflows()
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
				return v, v.fetchWorkflows()
			}
			return v, nil
		case "enter":
			// Navigate to selected workflow details
			if len(v.workflows) > 0 {
				selectedIdx := v.table.Cursor()
				if selectedIdx >= 0 && selectedIdx < len(v.workflows) {
					workflow := v.workflows[selectedIdx]
					workflowID := workflow.Metadata.Id
					v.debugLogger.Log("Navigating to workflow: %s", workflowID)
					return v, NewNavigateToWorkflowMsg(workflowID)
				}
			}
			return v, nil
		}

	case workflowTickMsg:
		// Don't auto-refresh if search modal is open
		if v.showingSearch {
			return v, workflowTick()
		}
		// Auto-refresh every 5 seconds
		return v, tea.Batch(v.fetchWorkflows(), workflowTick())

	case workflowsMsg:
		v.loading = false
		if msg.err != nil {
			v.HandleError(msg.err)
			v.debugLogger.Log("Error fetching workflows: %v", msg.err)
		} else {
			v.workflows = msg.workflows
			v.hasMore = msg.hasMore
			v.totalPages = msg.totalPages
			v.updateTableRows()
			v.lastFetch = time.Now()
			v.ClearError()
			v.debugLogger.Log("Successfully fetched %d workflows (offset=%d, hasMore=%v, totalPages=%d)",
				len(msg.workflows), v.currentOffset, v.hasMore, v.totalPages)
		}
		// Log debug info if available
		if msg.debugInfo != "" {
			v.debugLogger.Log("API: %s", msg.debugInfo)
		}
		return v, nil
	}

	// Handle mouse events for table scrolling
	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		// Handle mouse wheel scrolling using new API
		if mouseMsg.Action == tea.MouseActionPress {
			switch mouseMsg.Button {
			case tea.MouseButtonWheelUp:
				// Move cursor up
				if v.table.Cursor() > 0 {
					upMsg := tea.KeyMsg{Type: tea.KeyUp}
					_, cmd = v.table.Update(upMsg)
					return v, cmd
				}
			case tea.MouseButtonWheelDown:
				// Move cursor down
				if v.table.Cursor() < len(v.workflows)-1 {
					downMsg := tea.KeyMsg{Type: tea.KeyDown}
					_, cmd = v.table.Update(downMsg)
					return v, cmd
				}
			}
		}
	}

	// Don't update table if search modal is open
	if v.showingSearch {
		return v, nil
	}

	// Update the table model (handles keyboard and other events)
	_, cmd = v.table.Update(msg)
	return v, cmd
}

// View renders the view to a string
func (v *WorkflowsView) View() string {
	if v.Width == 0 {
		return "Initializing..."
	}

	// If debug view is enabled, show debug overlay
	if v.showDebug {
		return v.renderDebugView()
	}

	// If search form is showing, render it
	if v.showingSearch && v.searchForm != nil {
		return v.renderSearchModal()
	}

	// Header with logo and view indicator (using reusable component)
	header := RenderHeaderWithViewIndicator("Workflows", v.Ctx.ProfileName, v.Width)

	// Stats bar
	statsStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		Padding(0, 1)

	activeCount := 0
	pausedCount := 0
	for _, wf := range v.workflows {
		if wf.IsPaused != nil && *wf.IsPaused {
			pausedCount++
		} else {
			activeCount++
		}
	}

	statsText := fmt.Sprintf(
		"Total: %d  |  Active: %d  |  Paused: %d",
		len(v.workflows), activeCount, pausedCount,
	)
	if v.searchQuery != "" {
		statsText = fmt.Sprintf("%s  |  Search: '%s'", statsText, v.searchQuery)
	}
	stats := statsStyle.Render(statsText)

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

	// Only show pagination if there are actually workflows
	if len(v.workflows) > 0 {
		switch {
		case v.totalPages > 0:
			paginationText = fmt.Sprintf("Page %d/%d", currentPage, v.totalPages)
		case v.hasMore:
			paginationText = fmt.Sprintf("Page %d (more available)", currentPage)
		case currentPage > 1 || v.currentOffset > 0:
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
	controlItems = append(controlItems, "/: Search", "r: Refresh", "d: Debug", "h: Help", "v: Switch View", "shift+p: Profile", "q: Quit")
	if v.searchQuery != "" {
		controlItems = append(controlItems, "x: Clear Search")
	}
	controls := RenderFooter(controlItems, v.Width)

	// Build the full view
	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n\n")
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
func (v *WorkflowsView) SetSize(width, height int) {
	v.BaseModel.SetSize(width, height)
	if height > 12 {
		v.table.SetHeight(height - 12)
	}
}

// fetchWorkflows fetches workflows from the API
func (v *WorkflowsView) fetchWorkflows() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Parse tenant ID as UUID
		tenantUUID, err := uuid.Parse(v.Ctx.Client.TenantId())
		if err != nil {
			return workflowsMsg{
				workflows: nil,
				err:       fmt.Errorf("invalid tenant ID: %w", err),
			}
		}

		// Build params matching the frontend query
		params := &rest.WorkflowListParams{
			Offset: &v.currentOffset,
			Limit:  &v.pageSize,
		}

		// Add name filter if search query is set
		if v.searchQuery != "" {
			params.Name = &v.searchQuery
		}

		// Debug: log request parameters
		debugReq := fmt.Sprintf("Request: tenant=%s, offset=%d, limit=%d, name=%s",
			tenantUUID.String(), v.currentOffset, v.pageSize, v.searchQuery)

		// Call the API to list workflows
		// Matching the frontend query: queries.workflows.list()
		response, err := v.Ctx.Client.API().WorkflowListWithResponse(
			ctx,
			tenantUUID,
			params,
		)

		if err != nil {
			return workflowsMsg{
				workflows: nil,
				err:       fmt.Errorf("failed to fetch workflows: %w", err),
				debugInfo: debugReq + " | Error: " + err.Error(),
			}
		}

		if response.JSON200 == nil {
			// Debug: log the response body
			bodyStr := ""
			if response.Body != nil {
				bodyStr = string(response.Body)
			}
			return workflowsMsg{
				workflows: nil,
				err:       fmt.Errorf("unexpected response from API: status %d, body: %s", response.StatusCode(), bodyStr),
				debugInfo: fmt.Sprintf("Status: %d", response.StatusCode()),
			}
		}

		workflows := []rest.Workflow{}
		if response.JSON200.Rows != nil {
			workflows = *response.JSON200.Rows
		}

		// Calculate pagination info
		hasMore := false
		totalPages := 0

		if response.JSON200.Pagination != nil {
			if response.JSON200.Pagination.NextPage != nil {
				hasMore = true
			}
			if response.JSON200.Pagination.NumPages != nil {
				totalPages = int(*response.JSON200.Pagination.NumPages)
			}
		}

		// Debug: combine request and response info
		debugInfo := debugReq + fmt.Sprintf(" | Response: rows=%d, hasMore=%v, totalPages=%d",
			len(workflows), hasMore, totalPages)

		return workflowsMsg{
			workflows:  workflows,
			err:        nil,
			debugInfo:  debugInfo,
			hasMore:    hasMore,
			totalPages: totalPages,
		}
	}
}

// updateTableRows updates the table rows based on current workflows
func (v *WorkflowsView) updateTableRows() {
	rows := make([]table.Row, len(v.workflows))

	for i, workflow := range v.workflows {
		// Name
		name := workflow.Name

		// Created At
		createdAt := formatRelativeTime(workflow.Metadata.CreatedAt)

		// Status - use plain text, StyleFunc will apply colors
		status := "Active"
		if workflow.IsPaused != nil && *workflow.IsPaused {
			status = "Paused"
		}

		rows[i] = table.Row{
			name,
			createdAt,
			status,
		}
	}

	v.table.SetRows(rows)
}

// workflowTick returns a command that sends a tick message after a delay
func workflowTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return workflowTickMsg(t)
	})
}

// renderDebugView renders the debug log overlay using the shared component
func (v *WorkflowsView) renderDebugView() string {
	return RenderDebugView(v.debugLogger, v.Width, v.Height, "")
}

// renderSearchModal renders the search input form
func (v *WorkflowsView) renderSearchModal() string {
	var b strings.Builder

	// Header
	header := RenderHeader("Search Workflows", v.Ctx.ProfileName, v.Width)
	b.WriteString(header)
	b.WriteString("\n\n")

	// Instructions
	instructions := RenderInstructions(
		"Type to search  •  Enter: Apply  •  ↓/Esc: Cancel",
		v.Width,
	)
	b.WriteString(instructions)
	b.WriteString("\n\n")

	// The form
	b.WriteString(v.searchForm.View())
	b.WriteString("\n")

	// Footer with controls
	footer := RenderFooter([]string{
		"Enter: Apply Search",
		"↓/Esc: Cancel",
		"x: Clear Search (when not searching)",
	}, v.Width)
	b.WriteString(footer)

	return b.String()
}
