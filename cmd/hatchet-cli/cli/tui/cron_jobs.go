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

// CronJobsView displays a list of cron jobs in a table
type CronJobsView struct {
	lastFetch   time.Time
	table       *TableWithStyleFunc
	debugLogger *DebugLogger
	cronJobs    []rest.CronWorkflows
	BaseModel
	loading   bool
	showDebug bool
}

// cronJobsMsg contains the fetched cron jobs
type cronJobsMsg struct {
	err       error
	debugInfo string
	cronJobs  []rest.CronWorkflows
}

// cronJobTickMsg is sent periodically to refresh the data
type cronJobTickMsg time.Time

// NewCronJobsView creates a new cron jobs list view
func NewCronJobsView(ctx ViewContext) *CronJobsView {
	v := &CronJobsView{
		BaseModel:   BaseModel{Ctx: ctx},
		loading:     false,
		debugLogger: NewDebugLogger(5000),
		showDebug:   false,
	}

	columns := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Expression", Width: 18},
		{Title: "Workflow", Width: 25},
		{Title: "Enabled", Width: 8},
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

	// Style: color the Enabled column (col 3) green for enabled, red for disabled
	t.SetStyleFunc(func(row, col int) lipgloss.Style {
		if col == 3 && row < len(v.cronJobs) {
			if v.cronJobs[row].Enabled {
				return lipgloss.NewStyle().Foreground(styles.StatusSuccessColor)
			}
			return lipgloss.NewStyle().Foreground(styles.StatusFailedColor)
		}
		return lipgloss.NewStyle()
	})

	v.table = t
	return v
}

// Init initializes the view
func (v *CronJobsView) Init() tea.Cmd {
	return tea.Batch(v.fetchCronJobs(), cronJobTick())
}

// Update handles messages and updates the view state
func (v *CronJobsView) Update(msg tea.Msg) (View, tea.Cmd) {
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
			return v, v.fetchCronJobs()
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
		}

	case cronJobTickMsg:
		return v, tea.Batch(v.fetchCronJobs(), cronJobTick())

	case cronJobsMsg:
		v.loading = false
		if msg.err != nil {
			v.HandleError(msg.err)
			v.debugLogger.Log("Error fetching cron jobs: %v", msg.err)
		} else {
			v.cronJobs = msg.cronJobs
			v.updateTableRows()
			v.lastFetch = time.Now()
			v.ClearError()
			v.debugLogger.Log("Fetched %d cron jobs", len(msg.cronJobs))
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
				if v.table.Cursor() < len(v.cronJobs)-1 {
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
func (v *CronJobsView) View() string {
	if v.Width == 0 {
		return "Initializing..."
	}

	if v.showDebug {
		return RenderDebugView(v.debugLogger, v.Width, v.Height, "")
	}

	header := RenderHeaderWithViewIndicator("Cron Jobs", v.Ctx.ProfileName, v.Width)

	statsStyle := lipgloss.NewStyle().Foreground(styles.MutedColor).Padding(0, 1)
	enabledCount := 0
	for _, cj := range v.cronJobs {
		if cj.Enabled {
			enabledCount++
		}
	}
	stats := statsStyle.Render(fmt.Sprintf("Total: %d  |  Enabled: %d", len(v.cronJobs), enabledCount))

	loadingText := ""
	if v.loading {
		loadingStyle := lipgloss.NewStyle().Foreground(styles.AccentColor).Padding(0, 1)
		loadingText = loadingStyle.Render("Loading...")
	}

	controlItems := []string{
		"↑/↓: Navigate",
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
func (v *CronJobsView) SetSize(width, height int) {
	v.BaseModel.SetSize(width, height)
	if height > 12 {
		v.table.SetHeight(height - 12)
	}
}

// fetchCronJobs fetches cron jobs from the API
func (v *CronJobsView) fetchCronJobs() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		tenantUUID, err := uuid.Parse(v.Ctx.Client.TenantId())
		if err != nil {
			return cronJobsMsg{err: fmt.Errorf("invalid tenant ID: %w", err)}
		}

		limit := int64(50)
		offset := int64(0)
		response, err := v.Ctx.Client.API().CronWorkflowListWithResponse(ctx, tenantUUID, &rest.CronWorkflowListParams{
			Limit:  &limit,
			Offset: &offset,
		})
		if err != nil {
			return cronJobsMsg{
				err:       fmt.Errorf("failed to fetch cron jobs: %w", err),
				debugInfo: "Error: " + err.Error(),
			}
		}
		if response.JSON200 == nil {
			return cronJobsMsg{
				err:       fmt.Errorf("unexpected response from API: status %d", response.StatusCode()),
				debugInfo: fmt.Sprintf("Status: %d", response.StatusCode()),
			}
		}

		cronJobs := []rest.CronWorkflows{}
		if response.JSON200.Rows != nil {
			cronJobs = *response.JSON200.Rows
		}

		return cronJobsMsg{
			cronJobs:  cronJobs,
			debugInfo: fmt.Sprintf("Fetched %d cron jobs", len(cronJobs)),
		}
	}
}

// updateTableRows updates the table rows based on current cron jobs
func (v *CronJobsView) updateTableRows() {
	rows := make([]table.Row, len(v.cronJobs))

	for i, cj := range v.cronJobs {
		name := "(no name)"
		if cj.Name != nil && *cj.Name != "" {
			name = *cj.Name
		}

		enabled := "✗"
		if cj.Enabled {
			enabled = "✓"
		}

		createdAt := formatRelativeTime(cj.Metadata.CreatedAt)

		rows[i] = table.Row{
			name,
			cj.Cron,
			cj.WorkflowName,
			enabled,
			createdAt,
		}
	}

	v.table.SetRows(rows)
}

// cronJobTick returns a command that sends a tick message after a delay
func cronJobTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return cronJobTickMsg(t)
	})
}
