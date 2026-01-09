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
