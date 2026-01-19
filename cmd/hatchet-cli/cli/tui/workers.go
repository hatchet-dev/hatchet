package tui

import (
	"context"
	"fmt"
	"slices"
	"sort"
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

// WorkersView displays a list of workers in a table
type WorkersView struct {
	lastFetch         time.Time
	table             *TableWithStyleFunc
	debugLogger       *DebugLogger
	filterForm        *huh.Form
	workers           []rest.Worker
	filteredWorkers   []rest.Worker
	selectedStatuses  []string
	tempStatusFilters []string
	BaseModel
	loading       bool
	showDebug     bool
	showingFilter bool
}

// workersMsg contains the fetched workers
type workersMsg struct {
	err       error
	debugInfo string
	workers   []rest.Worker
}

// workerTickMsg is sent periodically to refresh the data
type workerTickMsg time.Time

// NewWorkersView creates a new workers list view
func NewWorkersView(ctx ViewContext) *WorkersView {
	v := &WorkersView{
		BaseModel: BaseModel{
			Ctx: ctx,
		},
		loading:          false,
		debugLogger:      NewDebugLogger(5000), // 5000 log entries max
		showDebug:        false,
		selectedStatuses: []string{"ACTIVE", "PAUSED"}, // Show active and paused workers by default
	}

	v.debugLogger.Log("WorkersView initialized")

	// Create columns with Name first (highlighted), Status last:
	// Name, Started At, Slots, Last Seen, SDK Version, Status
	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Started At", Width: 16},
		{Title: "Slots", Width: 12},
		{Title: "Last Seen", Width: 16},
		{Title: "SDK Version", Width: 20},
		{Title: "Status", Width: 10},
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

	// Set StyleFunc for per-cell styling
	t.SetStyleFunc(func(row, col int) lipgloss.Style {
		// Column 5 is the status column (last column)
		if col == 5 && row < len(v.filteredWorkers) {
			status := v.filteredWorkers[row].Status
			if status == nil {
				return lipgloss.NewStyle()
			}

			switch *status {
			case "ACTIVE":
				return lipgloss.NewStyle().Foreground(styles.StatusSuccessColor)
			case "PAUSED":
				return lipgloss.NewStyle().Foreground(styles.StatusInProgressColor)
			case "INACTIVE":
				return lipgloss.NewStyle().Foreground(styles.StatusFailedColor)
			default:
				return lipgloss.NewStyle().Foreground(styles.MutedColor)
			}
		}
		return lipgloss.NewStyle()
	})

	v.table = t

	return v
}

// Init initializes the view
func (v *WorkersView) Init() tea.Cmd {
	// Start fetching data immediately
	return tea.Batch(v.fetchWorkers(), workerTick())
}

// Update handles messages and updates the view state
func (v *WorkersView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmd tea.Cmd

	// If showing filter form, delegate ALL messages to the form first
	if v.showingFilter && v.filterForm != nil {
		form, cmd := v.filterForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			v.filterForm = f

			// Check if form is complete
			if v.filterForm.State == huh.StateCompleted {
				// Apply filters from temp to selected
				v.selectedStatuses = v.tempStatusFilters

				v.showingFilter = false
				v.debugLogger.Log("Filters applied: %v", v.selectedStatuses)

				// Re-filter the table rows
				v.updateTableRows()
				return v, nil
			}

			// Check for ESC to cancel
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				if keyMsg.String() == "esc" {
					v.showingFilter = false
					v.filterForm = nil
					v.debugLogger.Log("Filter cancelled")
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
			// Refresh the data
			v.debugLogger.Log("Manual refresh triggered")
			v.loading = true
			return v, v.fetchWorkers()
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
			// Open filter modal
			v.debugLogger.Log("Opening filters")
			v.showingFilter = true

			// Initialize temp filter state from current filters
			v.tempStatusFilters = make([]string, len(v.selectedStatuses))
			copy(v.tempStatusFilters, v.selectedStatuses)

			// Create filter form with multi-select
			v.filterForm = huh.NewForm(
				huh.NewGroup(
					huh.NewMultiSelect[string]().
						Title("Worker Statuses").
						Description("x/space to toggle | Enter to confirm | Esc to cancel").
						Options(
							huh.NewOption("Active", "ACTIVE"),
							huh.NewOption("Paused", "PAUSED"),
							huh.NewOption("Inactive", "INACTIVE"),
						).
						Value(&v.tempStatusFilters).
						Filterable(false),
				),
			).WithTheme(styles.HatchetTheme()).
				WithShowHelp(false)

			return v, v.filterForm.Init()
		case "enter":
			// Navigate to selected worker details
			if len(v.filteredWorkers) > 0 {
				selectedIdx := v.table.Cursor()
				if selectedIdx >= 0 && selectedIdx < len(v.filteredWorkers) {
					worker := v.filteredWorkers[selectedIdx]
					workerID := worker.Metadata.Id
					v.debugLogger.Log("Navigating to worker: %s", workerID)
					return v, NewNavigateToWorkerMsg(workerID)
				}
			}
			return v, nil
		}

	case workerTickMsg:
		// Don't auto-refresh if filter modal is open
		if v.showingFilter {
			return v, workerTick()
		}
		// Auto-refresh every 5 seconds
		return v, tea.Batch(v.fetchWorkers(), workerTick())

	case workersMsg:
		v.loading = false
		if msg.err != nil {
			v.HandleError(msg.err)
			v.debugLogger.Log("Error fetching workers: %v", msg.err)
		} else {
			v.workers = msg.workers
			v.updateTableRows()
			v.lastFetch = time.Now()
			v.ClearError()
			v.debugLogger.Log("Successfully fetched %d workers", len(msg.workers))
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
				if v.table.Cursor() < len(v.workers)-1 {
					downMsg := tea.KeyMsg{Type: tea.KeyDown}
					_, cmd = v.table.Update(downMsg)
					return v, cmd
				}
			}
		}
	}

	// Don't update table if filter modal is open
	if v.showingFilter {
		return v, nil
	}

	// Update the table model (handles keyboard and other events)
	_, cmd = v.table.Update(msg)
	return v, cmd
}

// View renders the view to a string
func (v *WorkersView) View() string {
	if v.Width == 0 {
		return "Initializing..."
	}

	// If debug view is enabled, show debug overlay
	if v.showDebug {
		return v.renderDebugView()
	}

	// If filter form is showing, render it
	if v.showingFilter && v.filterForm != nil {
		return v.renderFilterModal()
	}

	// Header with logo and view indicator (using reusable component)
	header := RenderHeaderWithViewIndicator("Workers", v.Ctx.ProfileName, v.Width)

	// Stats bar
	statsStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		Padding(0, 1)

	activeCount := 0
	pausedCount := 0
	inactiveCount := 0
	for _, w := range v.workers {
		if w.Status != nil {
			switch *w.Status {
			case "ACTIVE":
				activeCount++
			case "PAUSED":
				pausedCount++
			case "INACTIVE":
				inactiveCount++
			}
		}
	}

	stats := statsStyle.Render(fmt.Sprintf(
		"Total: %d  |  Active: %d  |  Paused: %d  |  Inactive: %d",
		len(v.workers), activeCount, pausedCount, inactiveCount,
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
	controlItems := []string{
		"↑/↓: Navigate",
		"enter: View Details",
		"f: Filter",
		"r: Refresh",
		"d: Debug",
		"h: Help",
		"shift+tab: Switch View",
		"q: Quit",
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
func (v *WorkersView) SetSize(width, height int) {
	v.BaseModel.SetSize(width, height)
	if height > 12 {
		v.table.SetHeight(height - 12)
	}
}

// fetchWorkers fetches workers from the API
func (v *WorkersView) fetchWorkers() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Parse tenant ID as UUID
		tenantUUID, err := uuid.Parse(v.Ctx.Client.TenantId())
		if err != nil {
			return workersMsg{
				workers: nil,
				err:     fmt.Errorf("invalid tenant ID: %w", err),
			}
		}

		// Debug: log request parameters
		debugReq := fmt.Sprintf("Request: tenant=%s", tenantUUID.String())

		// Call the API to list workers
		// Matching the frontend query: queries.workers.list()
		response, err := v.Ctx.Client.API().WorkerListWithResponse(
			ctx,
			tenantUUID,
		)

		if err != nil {
			return workersMsg{
				workers:   nil,
				err:       fmt.Errorf("failed to fetch workers: %w", err),
				debugInfo: debugReq + " | Error: " + err.Error(),
			}
		}

		if response.JSON200 == nil {
			// Debug: log the response body
			bodyStr := ""
			if response.Body != nil {
				bodyStr = string(response.Body)
			}
			return workersMsg{
				workers:   nil,
				err:       fmt.Errorf("unexpected response from API: status %d, body: %s", response.StatusCode(), bodyStr),
				debugInfo: fmt.Sprintf("Status: %d", response.StatusCode()),
			}
		}

		workers := []rest.Worker{}
		if response.JSON200.Rows != nil {
			workers = *response.JSON200.Rows
		}

		// Debug: combine request and response info
		debugInfo := debugReq + fmt.Sprintf(" | Response: rows=%d", len(workers))

		return workersMsg{
			workers:   workers,
			err:       nil,
			debugInfo: debugInfo,
		}
	}
}

// updateTableRows updates the table rows based on current workers
func (v *WorkersView) updateTableRows() {
	// Filter workers based on selected status filters
	v.filteredWorkers = []rest.Worker{}
	for _, worker := range v.workers {
		if worker.Status != nil {
			// Convert WorkerStatus to string
			statusStr := fmt.Sprintf("%v", *worker.Status)
			// Check if this status is in the selected statuses
			if slices.Contains(v.selectedStatuses, statusStr) {
				v.filteredWorkers = append(v.filteredWorkers, worker)
			}
		}
	}

	// Sort workers by Started At (CreatedAt) descending (newest first)
	sort.Slice(v.filteredWorkers, func(i, j int) bool {
		return v.filteredWorkers[i].Metadata.CreatedAt.After(v.filteredWorkers[j].Metadata.CreatedAt)
	})

	rows := make([]table.Row, len(v.filteredWorkers))

	for i, worker := range v.filteredWorkers {
		// Status
		status := "Unknown"
		if worker.Status != nil {
			switch *worker.Status {
			case "ACTIVE":
				status = "Active"
			case "PAUSED":
				status = "Paused"
			case "INACTIVE":
				status = "Inactive"
			}
		}

		// Name - use webhookUrl if available, otherwise name
		name := worker.Name
		if worker.WebhookUrl != nil && *worker.WebhookUrl != "" {
			name = *worker.WebhookUrl
		}

		// Started At
		startedAt := formatRelativeTime(worker.Metadata.CreatedAt)

		// Slots
		slots := "N/A"
		if worker.AvailableRuns != nil && worker.MaxRuns != nil {
			slots = fmt.Sprintf("%d / %d", *worker.AvailableRuns, *worker.MaxRuns)
		}

		// Last Seen
		lastSeen := "Never"
		if worker.LastHeartbeatAt != nil {
			lastSeen = formatRelativeTime(*worker.LastHeartbeatAt)
		}

		// SDK Version - capitalize properly (e.g., "Python" instead of "PYTHON")
		// Always show both language and version in format: "Language Version"
		sdkVersion := "Unknown"
		if worker.RuntimeInfo != nil {
			if worker.RuntimeInfo.Language != nil {
				langStr := capitalizeSDKRuntime(string(*worker.RuntimeInfo.Language))
				versionStr := "(unknown)"
				if worker.RuntimeInfo.SdkVersion != nil && *worker.RuntimeInfo.SdkVersion != "" {
					versionStr = *worker.RuntimeInfo.SdkVersion
				}
				sdkVersion = fmt.Sprintf("%s %s", langStr, versionStr)
			}
		}

		rows[i] = table.Row{
			name,
			startedAt,
			slots,
			lastSeen,
			sdkVersion,
			status,
		}
	}

	v.table.SetRows(rows)
}

// capitalizeSDKRuntime capitalizes SDK runtime names properly
// e.g., "PYTHON" -> "Python", "GO" -> "Go", "TYPESCRIPT" -> "TypeScript"
func capitalizeSDKRuntime(runtime string) string {
	runtime = strings.ToLower(runtime)

	// Special cases
	switch runtime {
	case "typescript":
		return "TypeScript"
	case "javascript":
		return "JavaScript"
	default:
		// Capitalize first letter for standard languages (Python, Go, etc.)
		if len(runtime) > 0 {
			return strings.ToUpper(runtime[:1]) + runtime[1:]
		}
		return runtime
	}
}

// workerTick returns a command that sends a tick message after a delay
func workerTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return workerTickMsg(t)
	})
}

// renderDebugView renders the debug log overlay using the shared component
func (v *WorkersView) renderDebugView() string {
	return RenderDebugView(v.debugLogger, v.Width, v.Height, "")
}

// renderFilterModal renders the filter modal
func (v *WorkersView) renderFilterModal() string {
	var b strings.Builder

	// Header (using reusable component)
	header := RenderHeader("Filter Workers", v.Ctx.ProfileName, v.Width)
	b.WriteString(header)
	b.WriteString("\n\n")

	// Instructions (using reusable component)
	instructions := RenderInstructions(
		"↑/↓: Navigate  •  x/space: Toggle  •  Enter: Apply  •  Esc: Cancel",
		v.Width,
	)
	b.WriteString(instructions)
	b.WriteString("\n\n")

	// The form
	b.WriteString(v.filterForm.View())
	b.WriteString("\n")

	// Footer (using reusable component)
	footer := RenderFooter([]string{
		"↑/↓: Navigate",
		"x/space: Toggle",
		"Enter: Apply",
		"Esc: Cancel",
	}, v.Width)
	b.WriteString(footer)

	return b.String()
}
