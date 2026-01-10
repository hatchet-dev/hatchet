package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// ViewContext contains the shared context passed to all views
type ViewContext struct {
	// Profile name for display
	ProfileName string

	// Hatchet client for API calls
	Client client.Client

	// Terminal dimensions
	Width  int
	Height int
}

// View represents a TUI view component
type View interface {
	// Init initializes the view and returns any initial commands
	Init() tea.Cmd

	// Update handles messages and updates the view state
	Update(msg tea.Msg) (View, tea.Cmd)

	// View renders the view to a string
	View() string

	// SetSize updates the view dimensions
	SetSize(width, height int)
}

// BaseModel contains common fields for all views
type BaseModel struct {
	Ctx    ViewContext
	Width  int
	Height int
	Err    error
}

// SetSize updates the base model dimensions
func (m *BaseModel) SetSize(width, height int) {
	m.Width = width
	m.Height = height
	m.Ctx.Width = width
	m.Ctx.Height = height
}

// HandleError sets an error on the base model
func (m *BaseModel) HandleError(err error) {
	m.Err = err
}

// ClearError clears any error on the base model
func (m *BaseModel) ClearError() {
	m.Err = nil
}

// NavigateToRunMsg is sent when navigating to a workflow run details view (deprecated, use NavigateToRunWithDetectionMsg)
type NavigateToRunMsg struct {
	WorkflowRunID string
}

// NavigateToRunWithDetectionMsg requests detection of run type (task vs dag)
type NavigateToRunWithDetectionMsg struct {
	RunID string
}

// RunTypeDetectedMsg contains the detected run type and data
type RunTypeDetectedMsg struct {
	Type     string // "task" or "dag"
	TaskData *rest.V1TaskSummary
	DAGData  *rest.V1WorkflowRunDetails
	Error    error
}

// NavigateBackMsg is sent when navigating back to the tasks view
type NavigateBackMsg struct{}

// NewNavigateToRunMsg creates a navigation message to a workflow run (deprecated)
func NewNavigateToRunMsg(workflowRunID string) tea.Cmd {
	return func() tea.Msg {
		return NavigateToRunMsg{WorkflowRunID: workflowRunID}
	}
}

// NewNavigateToRunWithDetectionMsg creates a navigation message with run type detection
func NewNavigateToRunWithDetectionMsg(runID string) tea.Cmd {
	return func() tea.Msg {
		return NavigateToRunWithDetectionMsg{RunID: runID}
	}
}

// NewNavigateBackMsg creates a navigation message to go back
func NewNavigateBackMsg() tea.Cmd {
	return func() tea.Msg {
		return NavigateBackMsg{}
	}
}

// RenderHeader renders a consistent header with logo for all views
func RenderHeader(title string, profileName string, width int) string {
	fullTitle := fmt.Sprintf("%s - Profile: %s", title, profileName)
	return RenderHeaderWithLogo(fullTitle, width)
}

// RenderInstructions renders instruction text in a consistent style
func RenderInstructions(instructions string, width int) string {
	instructionsStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		Padding(0, 1).
		Width(width - 4)
	return instructionsStyle.Render(instructions)
}

// RenderFooter renders a consistent footer with controls/help text
func RenderFooter(controls []string, width int) string {
	footerStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(styles.AccentColor).
		Width(width-4).
		Padding(0, 1)

	footerText := strings.Join(controls, "  •  ")
	return footerStyle.Render(footerText)
}

// RenderDebugView renders a debug log overlay with file writing support
func RenderDebugView(logger *DebugLogger, width, height int, extraInfo string) string {
	logs := logger.GetLogs()

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.AccentColor).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(styles.AccentColor).
		Width(width-4).
		Padding(0, 1)

	headerParts := []string{
		fmt.Sprintf("Debug Logs - %d/%d entries", logger.Size(), logger.Capacity()),
	}

	// Add status message or file prompt to header
	if statusMsg := logger.GetStatusMessage(); statusMsg != "" {
		msgStyle := lipgloss.NewStyle().Foreground(styles.AccentColor)
		headerParts = append(headerParts, msgStyle.Render(statusMsg))
	} else if logger.IsPromptingFile() {
		if logger.IsConfirmingOverwrite() {
			headerParts = append(headerParts, fmt.Sprintf("⚠ File '%s' exists. Overwrite? (y/n)", logger.GetFileInput()))
		} else {
			headerParts = append(headerParts, fmt.Sprintf("💾 Write to file: %s_", logger.GetFileInput()))
		}
	}

	header := headerStyle.Render(strings.Join(headerParts, " │ "))

	// Extra info section (optional - for view-specific context)
	var extraInfoText string
	if extraInfo != "" {
		infoStyle := lipgloss.NewStyle().
			Foreground(styles.AccentColor).
			Padding(0, 1).
			Width(width - 4)
		extraInfoText = infoStyle.Render(extraInfo)
	}

	// Log entries
	logStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(width - 4)

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	if extraInfoText != "" {
		b.WriteString(extraInfoText)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Calculate how many logs we can show
	reservedLines := 8 // header, extra info, footer, spacing
	if extraInfo != "" {
		reservedLines += 2 // Extra space for info
	}
	maxLines := height - reservedLines
	if maxLines < 1 {
		maxLines = 1
	}

	// Show most recent logs first
	startIdx := 0
	if len(logs) > maxLines {
		startIdx = len(logs) - maxLines
	}

	for i := startIdx; i < len(logs); i++ {
		log := logs[i]
		timestamp := log.Timestamp.Format("15:04:05.000")
		logLine := fmt.Sprintf("[%s] %s", timestamp, log.Message)
		b.WriteString(logStyle.Render(logLine))
		b.WriteString("\n")
	}

	// Footer with controls
	footerStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(styles.AccentColor).
		Width(width-4).
		Padding(0, 1)

	var controlItems []string
	if logger.IsPromptingFile() {
		if logger.IsConfirmingOverwrite() {
			controlItems = []string{"y: Confirm", "n/esc: Cancel"}
		} else {
			controlItems = []string{"Type filename", "enter: Confirm", "esc: Cancel"}
		}
	} else {
		controlItems = []string{"w: Write to file", "c: Clear Logs", "d: Close Debug", "q: Quit"}
	}

	controls := footerStyle.Render(strings.Join(controlItems, "  •  "))
	b.WriteString("\n")
	b.WriteString(controls)

	return b.String()
}

// HandleDebugKeyboard handles keyboard input for debug views with file writing
// Returns true if the key was handled, along with any command to execute
func HandleDebugKeyboard(logger *DebugLogger, key string) (bool, tea.Cmd) {
	// If prompting for file, handle file input
	if logger.IsPromptingFile() {
		if logger.IsConfirmingOverwrite() {
			// Handling overwrite confirmation
			switch key {
			case "y", "Y":
				return true, logger.ConfirmOverwrite()
			case "n", "N", "esc":
				logger.CancelFilePrompt()
				return true, nil
			}
		} else {
			// Handling filename input
			switch key {
			case "enter":
				if logger.GetFileInput() != "" {
					return true, logger.CheckAndWriteFile()
				}
				return true, nil
			case "esc":
				logger.CancelFilePrompt()
				return true, nil
			default:
				logger.HandleFileInput(key)
				return true, nil
			}
		}
	}

	// Not in file prompt mode, return false to let view handle other keys
	return false, nil
}
