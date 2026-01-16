package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// WorkerDetailsView displays details about a specific worker
type WorkerDetailsView struct {
	lastFetch   time.Time
	worker      *rest.Worker
	debugLogger *DebugLogger
	workerID    string
	BaseModel
	loading   bool
	showDebug bool
}

// workerDetailsMsg contains the fetched worker details
type workerDetailsMsg struct {
	worker    *rest.Worker
	err       error
	debugInfo string
}

// workerDetailsTickMsg is sent periodically to refresh the data
type workerDetailsTickMsg time.Time

// NewWorkerDetailsView creates a new worker details view
func NewWorkerDetailsView(ctx ViewContext, workerID string) *WorkerDetailsView {
	v := &WorkerDetailsView{
		BaseModel: BaseModel{
			Ctx: ctx,
		},
		workerID:    workerID,
		loading:     false,
		debugLogger: NewDebugLogger(5000),
		showDebug:   false,
	}

	v.debugLogger.Log("WorkerDetailsView initialized for worker %s", workerID)

	return v
}

// Init initializes the view
func (v *WorkerDetailsView) Init() tea.Cmd {
	// Start fetching worker details immediately
	return tea.Batch(v.fetchWorkerDetails(), workerDetailsTick())
}

// Update handles messages and updates the view state
func (v *WorkerDetailsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.SetSize(msg.Width, msg.Height)
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
			// Navigate back to workers list
			v.debugLogger.Log("Navigating back to workers list")
			return v, NewNavigateBackMsg()
		case "r":
			// Refresh the data
			v.debugLogger.Log("Manual refresh triggered")
			v.loading = true
			return v, v.fetchWorkerDetails()
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
		}

	case workerDetailsTickMsg:
		// Auto-refresh every 5 seconds
		return v, tea.Batch(v.fetchWorkerDetails(), workerDetailsTick())

	case workerDetailsMsg:
		v.loading = false
		if msg.err != nil {
			v.HandleError(msg.err)
			v.debugLogger.Log("Error fetching worker details: %v", msg.err)
		} else {
			v.worker = msg.worker
			v.lastFetch = time.Now()
			v.ClearError()
			v.debugLogger.Log("Successfully fetched worker details")
		}
		// Log debug info if available
		if msg.debugInfo != "" {
			v.debugLogger.Log("API: %s", msg.debugInfo)
		}
		return v, nil
	}

	return v, cmd
}

// View renders the view to a string
func (v *WorkerDetailsView) View() string {
	if v.Width == 0 {
		return "Initializing..."
	}

	// If debug view is enabled, show debug overlay
	if v.showDebug {
		return v.renderDebugView()
	}

	// Header with logo and back navigation
	header := RenderHeaderWithViewIndicator("Worker Details", v.Ctx.ProfileName, v.Width)

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n\n")

	// Show loading indicator if loading and no worker data yet
	if v.loading && v.worker == nil {
		loadingStyle := lipgloss.NewStyle().
			Foreground(styles.AccentColor).
			Padding(0, 1)
		b.WriteString(loadingStyle.Render("Loading worker details..."))
		b.WriteString("\n")
	} else if v.worker != nil {
		// Render worker details
		b.WriteString(v.renderWorkerInfo())
	}

	// Error display
	if v.Err != nil {
		b.WriteString("\n")
		b.WriteString(RenderError(fmt.Sprintf("Error: %v", v.Err), v.Width))
		b.WriteString("\n")
	}

	// Last updated timestamp
	if !v.lastFetch.IsZero() {
		lastFetchStyle := lipgloss.NewStyle().
			Foreground(styles.MutedColor).
			Padding(0, 1)
		b.WriteString("\n")
		b.WriteString(lastFetchStyle.Render(fmt.Sprintf("Last updated: %s", v.lastFetch.Format("15:04:05"))))
		b.WriteString("\n")
	}

	// Footer with controls
	controlItems := []string{
		"r: Refresh",
		"d: Debug",
		"esc/backspace: Back",
		"q: Quit",
	}
	controls := RenderFooter(controlItems, v.Width)
	b.WriteString("\n")
	b.WriteString(controls)

	return b.String()
}

// renderWorkerInfo renders the worker information
func (v *WorkerDetailsView) renderWorkerInfo() string {
	if v.worker == nil {
		return ""
	}

	var b strings.Builder

	// Title: Worker Name + Status
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.AccentColor).
		Padding(0, 1)

	workerName := v.worker.Name
	if v.worker.WebhookUrl != nil && *v.worker.WebhookUrl != "" {
		workerName = *v.worker.WebhookUrl
	}

	b.WriteString(titleStyle.Render(workerName))
	b.WriteString("  ")

	// Status badge
	statusBadge := v.renderStatusBadge()
	b.WriteString(statusBadge)
	b.WriteString("\n\n")

	// Worker Information Section
	sectionStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		Padding(0, 1)

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.PrimaryColor)

	// First Connected
	b.WriteString(sectionStyle.Render(labelStyle.Render("First Connected: ") + formatRelativeTime(v.worker.Metadata.CreatedAt)))
	b.WriteString("\n")

	// Last Listener Established
	if v.worker.LastListenerEstablished != nil {
		b.WriteString(sectionStyle.Render(labelStyle.Render("Last Listener Established: ") + formatRelativeTime(*v.worker.LastListenerEstablished)))
		b.WriteString("\n")
	}

	// Last Heartbeat
	lastHeartbeat := "never"
	if v.worker.LastHeartbeatAt != nil {
		lastHeartbeat = formatRelativeTime(*v.worker.LastHeartbeatAt)
	}
	b.WriteString(sectionStyle.Render(labelStyle.Render("Last Heartbeat: ") + lastHeartbeat))
	b.WriteString("\n\n")

	// Available Run Slots
	slotsStr := "N/A"
	if v.worker.AvailableRuns != nil && v.worker.MaxRuns != nil {
		slotsStr = fmt.Sprintf("%d / %d", *v.worker.AvailableRuns, *v.worker.MaxRuns)
	}
	b.WriteString(sectionStyle.Render(labelStyle.Render("Available Run Slots: ") + slotsStr))
	b.WriteString("\n\n")

	// Runtime Info Section
	if v.worker.RuntimeInfo != nil && v.hasRuntimeInfo() {
		separatorStyle := lipgloss.NewStyle().
			Foreground(styles.AccentColor).
			Bold(true).
			Padding(0, 1)
		b.WriteString(separatorStyle.Render("Runtime Information"))
		b.WriteString("\n")

		if v.worker.RuntimeInfo.SdkVersion != nil && *v.worker.RuntimeInfo.SdkVersion != "" {
			b.WriteString(sectionStyle.Render(labelStyle.Render("Hatchet SDK: ") + *v.worker.RuntimeInfo.SdkVersion))
			b.WriteString("\n")
		}

		if v.worker.RuntimeInfo.Language != nil && v.worker.RuntimeInfo.LanguageVersion != nil && *v.worker.RuntimeInfo.LanguageVersion != "" {
			langStr := capitalizeSDKRuntime(string(*v.worker.RuntimeInfo.Language))
			b.WriteString(sectionStyle.Render(labelStyle.Render("Runtime: ") + fmt.Sprintf("%s %s", langStr, *v.worker.RuntimeInfo.LanguageVersion)))
			b.WriteString("\n")
		}

		if v.worker.RuntimeInfo.Os != nil && *v.worker.RuntimeInfo.Os != "" {
			b.WriteString(sectionStyle.Render(labelStyle.Render("OS: ") + *v.worker.RuntimeInfo.Os))
			b.WriteString("\n")
		}

		if v.worker.RuntimeInfo.RuntimeExtra != nil && *v.worker.RuntimeInfo.RuntimeExtra != "" {
			b.WriteString(sectionStyle.Render(labelStyle.Render("Runtime Extra: ") + *v.worker.RuntimeInfo.RuntimeExtra))
			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	// Registered Workflows Section
	if v.worker.RegisteredWorkflows != nil && len(*v.worker.RegisteredWorkflows) > 0 {
		separatorStyle := lipgloss.NewStyle().
			Foreground(styles.AccentColor).
			Bold(true).
			Padding(0, 1)
		b.WriteString(separatorStyle.Render(fmt.Sprintf("Registered Workflows (%d)", len(*v.worker.RegisteredWorkflows))))
		b.WriteString("\n")

		workflowStyle := lipgloss.NewStyle().
			Foreground(styles.MutedColor).
			Padding(0, 2)

		for _, workflow := range *v.worker.RegisteredWorkflows {
			b.WriteString(workflowStyle.Render("• " + workflow.Name))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Worker Labels Section
	if v.worker.Labels != nil && len(*v.worker.Labels) > 0 {
		separatorStyle := lipgloss.NewStyle().
			Foreground(styles.AccentColor).
			Bold(true).
			Padding(0, 1)
		b.WriteString(separatorStyle.Render(fmt.Sprintf("Worker Labels (%d)", len(*v.worker.Labels))))
		b.WriteString("\n")

		labelItemStyle := lipgloss.NewStyle().
			Foreground(styles.MutedColor).
			Padding(0, 2)

		for _, label := range *v.worker.Labels {
			labelVal := ""
			if label.Value != nil {
				labelVal = *label.Value
			}
			b.WriteString(labelItemStyle.Render(fmt.Sprintf("• %s: %s", label.Key, labelVal)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderStatusBadge renders a status badge for the worker
func (v *WorkerDetailsView) renderStatusBadge() string {
	if v.worker == nil || v.worker.Status == nil {
		return ""
	}

	var statusColor lipgloss.AdaptiveColor
	var statusText string

	switch *v.worker.Status {
	case "ACTIVE":
		statusColor = styles.StatusSuccessColor
		statusText = "Active"
	case "PAUSED":
		statusColor = styles.StatusInProgressColor
		statusText = "Paused"
	case "INACTIVE":
		statusColor = styles.StatusFailedColor
		statusText = "Inactive"
	default:
		statusColor = styles.MutedColor
		statusText = "Unknown"
	}

	badgeStyle := lipgloss.NewStyle().
		Foreground(statusColor).
		Bold(true).
		Padding(0, 1)

	return badgeStyle.Render(fmt.Sprintf("[%s]", statusText))
}

// hasRuntimeInfo checks if the worker has any runtime info to display
func (v *WorkerDetailsView) hasRuntimeInfo() bool {
	if v.worker.RuntimeInfo == nil {
		return false
	}

	return (v.worker.RuntimeInfo.SdkVersion != nil && *v.worker.RuntimeInfo.SdkVersion != "") ||
		(v.worker.RuntimeInfo.LanguageVersion != nil && *v.worker.RuntimeInfo.LanguageVersion != "") ||
		(v.worker.RuntimeInfo.Os != nil && *v.worker.RuntimeInfo.Os != "") ||
		(v.worker.RuntimeInfo.RuntimeExtra != nil && *v.worker.RuntimeInfo.RuntimeExtra != "")
}

// fetchWorkerDetails fetches worker details from the API
func (v *WorkerDetailsView) fetchWorkerDetails() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Parse worker ID as UUID
		workerUUID, err := uuid.Parse(v.workerID)
		if err != nil {
			return workerDetailsMsg{
				worker: nil,
				err:    fmt.Errorf("invalid worker ID: %w", err),
			}
		}

		// Debug: log request parameters
		debugReq := fmt.Sprintf("Request: worker=%s", workerUUID.String())

		// Call the API to get worker details
		response, err := v.Ctx.Client.API().WorkerGetWithResponse(
			ctx,
			workerUUID,
		)

		if err != nil {
			return workerDetailsMsg{
				worker:    nil,
				err:       fmt.Errorf("failed to fetch worker details: %w", err),
				debugInfo: debugReq + " | Error: " + err.Error(),
			}
		}

		if response.JSON200 == nil {
			// Debug: log the response body
			bodyStr := ""
			if response.Body != nil {
				bodyStr = string(response.Body)
			}
			return workerDetailsMsg{
				worker:    nil,
				err:       fmt.Errorf("unexpected response from API: status %d, body: %s", response.StatusCode(), bodyStr),
				debugInfo: fmt.Sprintf("Status: %d", response.StatusCode()),
			}
		}

		// Debug: combine request and response info
		debugInfo := debugReq + " | Response: success"

		return workerDetailsMsg{
			worker:    response.JSON200,
			err:       nil,
			debugInfo: debugInfo,
		}
	}
}

// workerDetailsTick returns a command that sends a tick message after a delay
func workerDetailsTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return workerDetailsTickMsg(t)
	})
}

// renderDebugView renders the debug log overlay using the shared component
func (v *WorkerDetailsView) renderDebugView() string {
	return RenderDebugView(v.debugLogger, v.Width, v.Height, "")
}

// SetSize updates the view dimensions
func (v *WorkerDetailsView) SetSize(width, height int) {
	v.BaseModel.SetSize(width, height)
}
