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

// WebhooksView displays a list of V1 webhooks in a table
type WebhooksView struct {
	lastFetch   time.Time
	table       *TableWithStyleFunc
	debugLogger *DebugLogger
	webhooks    []rest.V1Webhook
	BaseModel
	loading   bool
	showDebug bool
}

// webhooksMsg contains the fetched webhooks
type webhooksMsg struct {
	err       error
	debugInfo string
	webhooks  []rest.V1Webhook
}

// webhookTickMsg is sent periodically to refresh the data
type webhookTickMsg time.Time

// NewWebhooksView creates a new webhooks list view
func NewWebhooksView(ctx ViewContext) *WebhooksView {
	v := &WebhooksView{
		BaseModel:   BaseModel{Ctx: ctx},
		loading:     false,
		debugLogger: NewDebugLogger(5000),
		showDebug:   false,
	}

	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Source", Width: 20},
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

	t.SetStyleFunc(func(row, col int) lipgloss.Style {
		return lipgloss.NewStyle()
	})

	v.table = t
	return v
}

// Init initializes the view
func (v *WebhooksView) Init() tea.Cmd {
	return tea.Batch(v.fetchWebhooks(), webhookTick())
}

// Update handles messages and updates the view state
func (v *WebhooksView) Update(msg tea.Msg) (View, tea.Cmd) {
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
			return v, v.fetchWebhooks()
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

	case webhookTickMsg:
		return v, tea.Batch(v.fetchWebhooks(), webhookTick())

	case webhooksMsg:
		v.loading = false
		if msg.err != nil {
			v.HandleError(msg.err)
			v.debugLogger.Log("Error fetching webhooks: %v", msg.err)
		} else {
			v.webhooks = msg.webhooks
			v.updateTableRows()
			v.lastFetch = time.Now()
			v.ClearError()
			v.debugLogger.Log("Fetched %d webhooks", len(msg.webhooks))
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
				if v.table.Cursor() < len(v.webhooks)-1 {
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
func (v *WebhooksView) View() string {
	if v.Width == 0 {
		return "Initializing..."
	}

	if v.showDebug {
		return RenderDebugView(v.debugLogger, v.Width, v.Height, "")
	}

	header := RenderHeaderWithViewIndicator("Webhooks", v.Ctx.ProfileName, v.Width)

	statsStyle := lipgloss.NewStyle().Foreground(styles.MutedColor).Padding(0, 1)
	stats := statsStyle.Render(fmt.Sprintf("Total: %d", len(v.webhooks)))

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
func (v *WebhooksView) SetSize(width, height int) {
	v.BaseModel.SetSize(width, height)
	if height > 12 {
		v.table.SetHeight(height - 12)
	}
}

// fetchWebhooks fetches webhooks from the API
func (v *WebhooksView) fetchWebhooks() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		tenantUUID, err := uuid.Parse(v.Ctx.Client.TenantId())
		if err != nil {
			return webhooksMsg{err: fmt.Errorf("invalid tenant ID: %w", err)}
		}

		limit := int64(200)
		offset := int64(0)
		response, err := v.Ctx.Client.API().V1WebhookListWithResponse(ctx, tenantUUID, &rest.V1WebhookListParams{
			Limit:  &limit,
			Offset: &offset,
		})
		if err != nil {
			return webhooksMsg{
				err:       fmt.Errorf("failed to fetch webhooks: %w", err),
				debugInfo: "Error: " + err.Error(),
			}
		}
		if response.JSON200 == nil {
			return webhooksMsg{
				err:       fmt.Errorf("unexpected response from API: status %d", response.StatusCode()),
				debugInfo: fmt.Sprintf("Status: %d", response.StatusCode()),
			}
		}

		webhooks := []rest.V1Webhook{}
		if response.JSON200.Rows != nil {
			webhooks = *response.JSON200.Rows
		}

		return webhooksMsg{
			webhooks:  webhooks,
			debugInfo: fmt.Sprintf("Fetched %d webhooks", len(webhooks)),
		}
	}
}

// updateTableRows updates the table rows based on current webhooks
func (v *WebhooksView) updateTableRows() {
	rows := make([]table.Row, len(v.webhooks))

	for i, wh := range v.webhooks {
		createdAt := formatRelativeTime(wh.Metadata.CreatedAt)
		rows[i] = table.Row{
			wh.Name,
			string(wh.SourceName),
			createdAt,
		}
	}

	v.table.SetRows(rows)
}

// webhookTick returns a command that sends a tick message after a delay
func webhookTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return webhookTickMsg(t)
	})
}
