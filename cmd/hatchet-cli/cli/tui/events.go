package tui

import (
	"context"
	"encoding/json"
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

type EventsView struct {
	lastFetch   time.Time
	table       *TableWithStyleFunc
	debugLogger *DebugLogger
	viewer      *ContentViewer
	events      []rest.V1Event
	BaseModel
	loading   bool
	showDebug bool
}

type tenantEventsMsg struct {
	err       error
	debugInfo string
	events    []rest.V1Event
}

type tenantEventTickMsg time.Time

func NewEventsView(ctx ViewContext) *EventsView {
	v := &EventsView{
		BaseModel:   BaseModel{Ctx: ctx},
		loading:     false,
		debugLogger: NewDebugLogger(5000),
		showDebug:   false,
	}

	columns := []table.Column{
		{Title: "Key", Width: 30},
		{Title: "Seen At", Width: 18},
		{Title: "Runs", Width: 24},
		{Title: "ID", Width: 36},
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

	t.SetStyleFunc(func(row, col int) lipgloss.Style {
		return lipgloss.NewStyle()
	})

	v.table = t
	return v
}

func (v *EventsView) Init() tea.Cmd {
	return tea.Batch(v.fetchEvents(), tenantEventTick())
}

func (v *EventsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmd tea.Cmd

	if v.viewer != nil && v.viewer.IsActive() {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			v.SetSize(msg.Width, msg.Height)
			return v, nil
		case tea.KeyMsg, tea.MouseMsg:
			cmd = v.viewer.Update(msg)
			if !v.viewer.IsActive() {
				v.viewer = nil
			}
			return v, cmd
		}
	}

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
		case "enter":
			return v, v.openSelectedEvent()
		case "r":
			v.loading = true
			return v, v.fetchEvents()
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

	case tenantEventTickMsg:
		return v, tea.Batch(v.fetchEvents(), tenantEventTick())

	case tenantEventsMsg:
		v.loading = false
		if msg.err != nil {
			v.HandleError(msg.err)
			v.debugLogger.Log("Error fetching events: %v", msg.err)
		} else {
			v.events = msg.events
			v.updateTableRows()
			v.lastFetch = time.Now()
			v.ClearError()
			v.debugLogger.Log("Fetched %d events", len(msg.events))
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
				if v.table.Cursor() < len(v.events)-1 {
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

func (v *EventsView) View() string {
	if v.Width == 0 {
		return "Initializing..."
	}

	if v.viewer != nil && v.viewer.IsActive() {
		header := RenderHeaderWithViewIndicator("Events", v.Ctx.ProfileName, v.Width)
		return header + "\n\n" + v.viewer.View()
	}

	if v.showDebug {
		return RenderDebugView(v.debugLogger, v.Width, v.Height, "")
	}

	header := RenderHeaderWithViewIndicator("Events", v.Ctx.ProfileName, v.Width)

	statsStyle := lipgloss.NewStyle().Foreground(styles.MutedColor).Padding(0, 1)
	stats := statsStyle.Render(fmt.Sprintf("Total: %d", len(v.events)))

	loadingText := ""
	if v.loading {
		loadingStyle := lipgloss.NewStyle().Foreground(styles.AccentColor).Padding(0, 1)
		loadingText = loadingStyle.Render("Loading...")
	}

	controlItems := []string{
		"↑/↓: Navigate",
		"enter: View Payload",
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

func (v *EventsView) SetSize(width, height int) {
	v.BaseModel.SetSize(width, height)
	if height > 12 {
		v.table.SetHeight(height - 12)
	}
	if v.viewer != nil {
		v.viewer.SetSize(width-4, height-8)
	}
}

func (v *EventsView) openSelectedEvent() tea.Cmd {
	cursor := v.table.Cursor()
	if cursor < 0 || cursor >= len(v.events) {
		return nil
	}

	event := v.events[cursor]
	content := map[string]interface{}{
		"id":                 event.Metadata.Id,
		"key":                event.Key,
		"payload":            event.Payload,
		"additionalMetadata": event.AdditionalMetadata,
	}

	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		v.HandleError(fmt.Errorf("failed to marshal event: %w", err))
		return nil
	}

	v.viewer = NewContentViewer(string(data), v.Width-4, v.Height-8)
	v.viewer.Activate()
	return nil
}

func (v *EventsView) fetchEvents() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		tenantUUID, err := uuid.Parse(v.Ctx.Client.TenantId())
		if err != nil {
			return tenantEventsMsg{err: fmt.Errorf("invalid tenant ID: %w", err)}
		}

		limit := int64(200)
		offset := int64(0)
		response, err := v.Ctx.Client.API().V1EventListWithResponse(ctx, tenantUUID, &rest.V1EventListParams{
			Limit:  &limit,
			Offset: &offset,
		})
		if err != nil {
			return tenantEventsMsg{
				err:       fmt.Errorf("failed to fetch events: %w", err),
				debugInfo: "Error: " + err.Error(),
			}
		}
		if response.JSON200 == nil {
			return tenantEventsMsg{
				err:       fmt.Errorf("unexpected response from API: status %d", response.StatusCode()),
				debugInfo: fmt.Sprintf("Status: %d", response.StatusCode()),
			}
		}

		events := []rest.V1Event{}
		if response.JSON200.Rows != nil {
			events = *response.JSON200.Rows
		}

		return tenantEventsMsg{
			events:    events,
			debugInfo: fmt.Sprintf("Fetched %d events", len(events)),
		}
	}
}

func (v *EventsView) updateTableRows() {
	rows := make([]table.Row, len(v.events))

	for i, e := range v.events {
		seenAt := e.Metadata.CreatedAt
		if e.SeenAt != nil {
			seenAt = *e.SeenAt
		}

		s := e.WorkflowRunSummary
		runs := fmt.Sprintf("✓%d ✗%d ▶%d ⧗%d ⊘%d", s.Succeeded, s.Failed, s.Running, s.Queued, s.Cancelled)

		rows[i] = table.Row{
			e.Key,
			formatRelativeTime(seenAt),
			runs,
			e.Metadata.Id,
		}
	}

	v.table.SetRows(rows)
}

func tenantEventTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tenantEventTickMsg(t)
	})
}
