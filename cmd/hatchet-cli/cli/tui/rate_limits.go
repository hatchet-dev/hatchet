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

// RateLimitsView displays a list of rate limits in a table
type RateLimitsView struct {
	lastFetch   time.Time
	table       *TableWithStyleFunc
	debugLogger *DebugLogger
	rateLimits  []rest.RateLimit
	BaseModel
	loading   bool
	showDebug bool
}

// rateLimitsMsg contains the fetched rate limits
type rateLimitsMsg struct {
	err        error
	debugInfo  string
	rateLimits []rest.RateLimit
}

// rateLimitTickMsg is sent periodically to refresh the data
type rateLimitTickMsg time.Time

// NewRateLimitsView creates a new rate limits list view
func NewRateLimitsView(ctx ViewContext) *RateLimitsView {
	v := &RateLimitsView{
		BaseModel:   BaseModel{Ctx: ctx},
		loading:     false,
		debugLogger: NewDebugLogger(5000),
		showDebug:   false,
	}

	columns := []table.Column{
		{Title: "Key", Width: 35},
		{Title: "Value", Width: 10},
		{Title: "Limit", Width: 10},
		{Title: "Usage %", Width: 10},
		{Title: "Window", Width: 12},
		{Title: "Last Refill", Width: 18},
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

	// Style: Value is remaining capacity. Red when exhausted (0), yellow when <=10% remaining.
	t.SetStyleFunc(func(row, col int) lipgloss.Style {
		if row < len(v.rateLimits) {
			rl := v.rateLimits[row]
			if rl.LimitValue > 0 {
				if rl.Value == 0 {
					return lipgloss.NewStyle().Foreground(styles.StatusFailedColor)
				}
				remaining := float64(rl.Value) / float64(rl.LimitValue)
				if remaining <= 0.1 {
					return lipgloss.NewStyle().Foreground(styles.StatusInProgressColor)
				}
			}
		}
		return lipgloss.NewStyle()
	})

	v.table = t
	return v
}

// Init initializes the view
func (v *RateLimitsView) Init() tea.Cmd {
	return tea.Batch(v.fetchRateLimits(), rateLimitTick())
}

// Update handles messages and updates the view state
func (v *RateLimitsView) Update(msg tea.Msg) (View, tea.Cmd) {
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
			return v, v.fetchRateLimits()
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

	case rateLimitTickMsg:
		return v, tea.Batch(v.fetchRateLimits(), rateLimitTick())

	case rateLimitsMsg:
		v.loading = false
		if msg.err != nil {
			v.HandleError(msg.err)
			v.debugLogger.Log("Error fetching rate limits: %v", msg.err)
		} else {
			v.rateLimits = msg.rateLimits
			v.updateTableRows()
			v.lastFetch = time.Now()
			v.ClearError()
			v.debugLogger.Log("Fetched %d rate limits", len(msg.rateLimits))
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
				if v.table.Cursor() < len(v.rateLimits)-1 {
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
func (v *RateLimitsView) View() string {
	if v.Width == 0 {
		return "Initializing..."
	}

	if v.showDebug {
		return RenderDebugView(v.debugLogger, v.Width, v.Height, "")
	}

	header := RenderHeaderWithViewIndicator("Rate Limits", v.Ctx.ProfileName, v.Width)

	statsStyle := lipgloss.NewStyle().Foreground(styles.MutedColor).Padding(0, 1)
	stats := statsStyle.Render(fmt.Sprintf("Total: %d", len(v.rateLimits)))

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
func (v *RateLimitsView) SetSize(width, height int) {
	v.BaseModel.SetSize(width, height)
	if height > 12 {
		v.table.SetHeight(height - 12)
	}
}

// fetchRateLimits fetches rate limits from the API
func (v *RateLimitsView) fetchRateLimits() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		tenantUUID, err := uuid.Parse(v.Ctx.Client.TenantId())
		if err != nil {
			return rateLimitsMsg{err: fmt.Errorf("invalid tenant ID: %w", err)}
		}

		limit := int64(100)
		response, err := v.Ctx.Client.API().RateLimitListWithResponse(ctx, tenantUUID, &rest.RateLimitListParams{
			Limit: &limit,
		})
		if err != nil {
			return rateLimitsMsg{
				err:       fmt.Errorf("failed to fetch rate limits: %w", err),
				debugInfo: "Error: " + err.Error(),
			}
		}
		if response.JSON200 == nil {
			return rateLimitsMsg{
				err:       fmt.Errorf("unexpected response from API: status %d", response.StatusCode()),
				debugInfo: fmt.Sprintf("Status: %d", response.StatusCode()),
			}
		}

		rateLimits := []rest.RateLimit{}
		if response.JSON200.Rows != nil {
			rateLimits = *response.JSON200.Rows
		}

		return rateLimitsMsg{
			rateLimits: rateLimits,
			debugInfo:  fmt.Sprintf("Fetched %d rate limits", len(rateLimits)),
		}
	}
}

// updateTableRows updates the table rows based on current rate limits
func (v *RateLimitsView) updateTableRows() {
	rows := make([]table.Row, len(v.rateLimits))

	for i, rl := range v.rateLimits {
		usagePct := "0%"
		if rl.LimitValue > 0 {
			pct := float64(rl.Value) / float64(rl.LimitValue) * 100
			usagePct = fmt.Sprintf("%.0f%%", pct)
		}

		lastRefill := rl.LastRefill.Format("01/02 15:04:05")

		rows[i] = table.Row{
			rl.Key,
			fmt.Sprintf("%d", rl.Value),
			fmt.Sprintf("%d", rl.LimitValue),
			usagePct,
			rl.Window,
			lastRefill,
		}
	}

	v.table.SetRows(rows)
}

// rateLimitTick returns a command that sends a tick message after a delay
func rateLimitTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return rateLimitTickMsg(t)
	})
}
