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

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// ScheduledRunsView displays a list of scheduled runs in a table
type ScheduledRunsView struct {
	lastFetch     time.Time
	table         *TableWithStyleFunc
	debugLogger   *DebugLogger
	scheduledRuns []rest.ScheduledWorkflows
	BaseModel
	loading   bool
	showDebug bool
}

// scheduledRunsMsg contains the fetched scheduled runs
type scheduledRunsMsg struct {
	err       error
	debugInfo string
	runs      []rest.ScheduledWorkflows
}

// scheduledRunTickMsg is sent periodically to refresh the data
type scheduledRunTickMsg time.Time

// NewScheduledRunsView creates a new scheduled runs list view
func NewScheduledRunsView(ctx ViewContext) *ScheduledRunsView {
	v := &ScheduledRunsView{
		BaseModel:   BaseModel{Ctx: ctx},
		loading:     false,
		debugLogger: NewDebugLogger(5000),
		showDebug:   false,
	}

	columns := []table.Column{
		{Title: "ID", Width: 10},
		{Title: "Status", Width: 14},
		{Title: "Trigger At", Width: 20},
		{Title: "Workflow", Width: 25},
		{Title: "Created At", Width: 18},
	}

	t := NewTableWithStyleFunc(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)

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

	// Style: color status column by run status
	t.SetStyleFunc(func(row, col int) lipgloss.Style {
		if col == 1 && row < len(v.scheduledRuns) {
			run := v.scheduledRuns[row]
			if run.WorkflowRunStatus != nil {
				switch string(*run.WorkflowRunStatus) {
				case "SUCCEEDED":
					return lipgloss.NewStyle().Foreground(styles.StatusSuccessColor)
				case "FAILED":
					return lipgloss.NewStyle().Foreground(styles.StatusFailedColor)
				case "RUNNING":
					return lipgloss.NewStyle().Foreground(styles.StatusInProgressColor)
				case "CANCELLED":
					return lipgloss.NewStyle().Foreground(styles.StatusCancelledColor)
				}
			}
		}
		return lipgloss.NewStyle()
	})

	v.table = t
	return v
}

// Init initializes the view
func (v *ScheduledRunsView) Init() tea.Cmd {
	return tea.Batch(v.fetchScheduledRuns(), scheduledRunTick())
}

// Update handles messages and updates the view state
func (v *ScheduledRunsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.SetSize(msg.Width, msg.Height)
		v.table.SetHeight(msg.Height - 12)
		return v, nil

	case tea.KeyMsg:
		if v.showDebug {
			if handled, debugCmd := HandleDebugKeyboard(v.debugLogger, msg.String()); handled {
				return v, debugCmd
			}
		}

		switch msg.String() {
		case "r":
			v.loading = true
			return v, v.fetchScheduledRuns()
		case "d":
			v.showDebug = !v.showDebug
			return v, nil
		case "c":
			if v.showDebug && !v.debugLogger.IsPromptingFile() {
				v.debugLogger.Clear()
			}
			return v, nil
		case "w":
			if v.showDebug && !v.debugLogger.IsPromptingFile() {
				v.debugLogger.StartFilePrompt()
			}
			return v, nil
		case "enter":
			// Navigate to run details if the scheduled run was triggered
			if len(v.scheduledRuns) > 0 {
				selectedIdx := v.table.Cursor()
				if selectedIdx >= 0 && selectedIdx < len(v.scheduledRuns) {
					run := v.scheduledRuns[selectedIdx]
					if run.WorkflowRunId != nil {
						runID := run.WorkflowRunId.String()
						v.debugLogger.Log("Navigating to run: %s", runID)
						return v, NewNavigateToRunWithDetectionMsg(runID)
					}
				}
			}
			return v, nil
		}

	case scheduledRunTickMsg:
		return v, tea.Batch(v.fetchScheduledRuns(), scheduledRunTick())

	case scheduledRunsMsg:
		v.loading = false
		if msg.err != nil {
			v.HandleError(msg.err)
			v.debugLogger.Log("Error fetching scheduled runs: %v", msg.err)
		} else {
			v.scheduledRuns = msg.runs
			v.updateTableRows()
			v.lastFetch = time.Now()
			v.ClearError()
			v.debugLogger.Log("Fetched %d scheduled runs", len(msg.runs))
		}
		if msg.debugInfo != "" {
			v.debugLogger.Log("API: %s", msg.debugInfo)
		}
		return v, nil
	}

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
				if v.table.Cursor() < len(v.scheduledRuns)-1 {
					downMsg := tea.KeyMsg{Type: tea.KeyDown}
					_, cmd = v.table.Update(downMsg)
					return v, cmd
				}
			}
		}
	}

	_, cmd = v.table.Update(msg)
	return v, cmd
}

// View renders the view to a string
func (v *ScheduledRunsView) View() string {
	if v.Width == 0 {
		return "Initializing..."
	}

	if v.showDebug {
		return RenderDebugView(v.debugLogger, v.Width, v.Height, "")
	}

	header := RenderHeaderWithViewIndicator("Scheduled Runs", v.Ctx.ProfileName, v.Width)

	statsStyle := lipgloss.NewStyle().Foreground(styles.MutedColor).Padding(0, 1)
	stats := statsStyle.Render(fmt.Sprintf("Total: %d", len(v.scheduledRuns)))

	loadingText := ""
	if v.loading {
		loadingStyle := lipgloss.NewStyle().Foreground(styles.AccentColor).Padding(0, 1)
		loadingText = loadingStyle.Render("Loading...")
	}

	controlItems := []string{
		"↑/↓: Navigate",
		"enter: View Run",
		"r: Refresh",
		"d: Debug",
		"h: Help",
		"shift+tab: Switch View",
		"q: Quit",
	}
	controls := RenderFooter(controlItems, v.Width)

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n\n")
	b.WriteString(stats)
	if loadingText != "" {
		b.WriteString("  ")
		b.WriteString(loadingText)
	}
	b.WriteString("\n\n")
	b.WriteString(v.table.View())
	b.WriteString("\n\n")

	if v.Err != nil {
		b.WriteString(RenderError(fmt.Sprintf("Error: %v", v.Err), v.Width))
		b.WriteString("\n")
	}

	if !v.lastFetch.IsZero() {
		lastFetchStyle := lipgloss.NewStyle().Foreground(styles.MutedColor).Padding(0, 1)
		b.WriteString(lastFetchStyle.Render(fmt.Sprintf("Last updated: %s", v.lastFetch.Format("15:04:05"))))
		b.WriteString("\n")
	}

	b.WriteString(controls)
	return b.String()
}

// SetSize updates the view dimensions
func (v *ScheduledRunsView) SetSize(width, height int) {
	v.BaseModel.SetSize(width, height)
	if height > 12 {
		v.table.SetHeight(height - 12)
	}
}

// fetchScheduledRuns fetches scheduled runs from the API
func (v *ScheduledRunsView) fetchScheduledRuns() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		tenantUUID, err := uuid.Parse(v.Ctx.Client.TenantId())
		if err != nil {
			return scheduledRunsMsg{err: fmt.Errorf("invalid tenant ID: %w", err)}
		}

		limit := int64(50)
		offset := int64(0)
		response, err := v.Ctx.Client.API().WorkflowScheduledListWithResponse(ctx, tenantUUID, &rest.WorkflowScheduledListParams{
			Limit:  &limit,
			Offset: &offset,
		})
		if err != nil {
			return scheduledRunsMsg{
				err:       fmt.Errorf("failed to fetch scheduled runs: %w", err),
				debugInfo: "Error: " + err.Error(),
			}
		}
		if response.JSON200 == nil {
			return scheduledRunsMsg{
				err:       fmt.Errorf("unexpected response from API: status %d", response.StatusCode()),
				debugInfo: fmt.Sprintf("Status: %d", response.StatusCode()),
			}
		}

		runs := []rest.ScheduledWorkflows{}
		if response.JSON200.Rows != nil {
			runs = *response.JSON200.Rows
		}

		return scheduledRunsMsg{
			runs:      runs,
			debugInfo: fmt.Sprintf("Fetched %d scheduled runs", len(runs)),
		}
	}
}

// updateTableRows updates the table rows based on current scheduled runs
func (v *ScheduledRunsView) updateTableRows() {
	rows := make([]table.Row, len(v.scheduledRuns))

	for i, run := range v.scheduledRuns {
		// Short ID (first 8 chars)
		shortID := run.Metadata.Id
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}

		// Status
		status := "Scheduled"
		if run.WorkflowRunStatus != nil {
			status = string(*run.WorkflowRunStatus)
		}

		// Trigger At
		triggerAt := run.TriggerAt.Format("01/02 15:04:05")

		// Created At
		createdAt := formatRelativeTime(run.Metadata.CreatedAt)

		rows[i] = table.Row{
			shortID,
			status,
			triggerAt,
			run.WorkflowName,
			createdAt,
		}
	}

	v.table.SetRows(rows)
}

// scheduledRunTick returns a command that sends a tick message after a delay
func scheduledRunTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return scheduledRunTickMsg(t)
	})
}
