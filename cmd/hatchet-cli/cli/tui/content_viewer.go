package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

// ContentViewer is a component for viewing long content with scrolling and file writing
type ContentViewer struct {
	content          string
	fileInput        string
	statusMessage    string
	lines            []string
	scrollOffset     int
	width            int
	height           int
	active           bool
	promptingFile    bool
	confirmOverwrite bool
}

// contentViewerFileWrittenMsg signals that a file was written
type contentViewerFileWrittenMsg struct{}

// contentViewerFileCheckMsg signals that file existence check completed
type contentViewerFileCheckMsg struct{}

// NewContentViewer creates a new content viewer
func NewContentViewer(content string, width, height int) *ContentViewer {
	lines := strings.Split(content, "\n")

	return &ContentViewer{
		content:          content,
		lines:            lines,
		scrollOffset:     0,
		width:            width,
		height:           height,
		active:           false,
		promptingFile:    false,
		fileInput:        "",
		confirmOverwrite: false,
		statusMessage:    "",
	}
}

// SetSize updates the viewer dimensions
func (v *ContentViewer) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetContent updates the content
func (v *ContentViewer) SetContent(content string) {
	v.content = content
	v.lines = strings.Split(content, "\n")
	v.scrollOffset = 0
}

// IsActive returns whether the viewer is in dive mode
func (v *ContentViewer) IsActive() bool {
	return v.active
}

// Activate enters dive mode
func (v *ContentViewer) Activate() {
	v.active = true
}

// Deactivate exits dive mode
func (v *ContentViewer) Deactivate() {
	v.active = false
}

// HandleMouse handles mouse events even when not active (for preview mode scrolling)
func (v *ContentViewer) HandleMouse(msg tea.MouseMsg) {
	if msg.Action == tea.MouseActionPress {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			v.ScrollUp()
		case tea.MouseButtonWheelDown:
			v.ScrollDown()
		}
	}
}

// Update handles messages when viewer is active
func (v *ContentViewer) Update(msg tea.Msg) tea.Cmd {
	if !v.active {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If prompting for filename
		if v.promptingFile {
			if v.confirmOverwrite {
				// Handling overwrite confirmation
				switch msg.String() {
				case "y", "Y":
					return v.writeToFile()
				case "n", "N", "esc":
					v.confirmOverwrite = false
					v.promptingFile = false
					v.fileInput = ""
					v.statusMessage = "Write cancelled"
				}
			} else {
				// Handling filename input
				switch msg.String() {
				case "enter":
					if v.fileInput != "" {
						return v.checkAndWriteFile()
					}
				case "esc":
					v.promptingFile = false
					v.fileInput = ""
					v.statusMessage = ""
				case "backspace":
					if len(v.fileInput) > 0 {
						v.fileInput = v.fileInput[:len(v.fileInput)-1]
					}
				default:
					// Add character to filename
					if len(msg.String()) == 1 {
						v.fileInput += msg.String()
					}
				}
			}
			return nil
		}

		// Normal navigation
		switch msg.String() {
		case "up", "k":
			v.ScrollUp()
		case "down", "j":
			v.ScrollDown()
		case "pgup":
			v.ScrollPageUp()
		case "pgdown":
			v.ScrollPageDown()
		case "g":
			v.ScrollToTop()
		case "G":
			v.ScrollToBottom()
		case "w":
			v.promptingFile = true
			v.fileInput = ""
			v.statusMessage = ""
		case "esc", "q":
			v.Deactivate()
		}

	case tea.MouseMsg:
		// Handle mouse wheel scrolling
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				v.ScrollUp()
			case tea.MouseButtonWheelDown:
				v.ScrollDown()
			}
		}
	}

	return nil
}

// ScrollUp scrolls up by one line
func (v *ContentViewer) ScrollUp() {
	if v.scrollOffset > 0 {
		v.scrollOffset--
	}
}

// ScrollDown scrolls down by one line
func (v *ContentViewer) ScrollDown() {
	// Reserve lines for status (1) and controls (1)
	reservedLines := 2
	contentHeight := v.height - reservedLines
	if contentHeight < 1 {
		contentHeight = 1
	}
	maxOffset := len(v.lines) - contentHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if v.scrollOffset < maxOffset {
		v.scrollOffset++
	}
}

// ScrollPageUp scrolls up by one page
func (v *ContentViewer) ScrollPageUp() {
	// Reserve lines for status (1) and controls (1)
	reservedLines := 2
	contentHeight := v.height - reservedLines
	if contentHeight < 1 {
		contentHeight = 1
	}
	v.scrollOffset -= contentHeight
	if v.scrollOffset < 0 {
		v.scrollOffset = 0
	}
}

// ScrollPageDown scrolls down by one page
func (v *ContentViewer) ScrollPageDown() {
	// Reserve lines for status (1) and controls (1)
	reservedLines := 2
	contentHeight := v.height - reservedLines
	if contentHeight < 1 {
		contentHeight = 1
	}
	maxOffset := len(v.lines) - contentHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	v.scrollOffset += contentHeight
	if v.scrollOffset > maxOffset {
		v.scrollOffset = maxOffset
	}
}

// ScrollToTop scrolls to the top
func (v *ContentViewer) ScrollToTop() {
	v.scrollOffset = 0
}

// ScrollToBottom scrolls to the bottom
func (v *ContentViewer) ScrollToBottom() {
	// Reserve lines for status (1) and controls (1)
	reservedLines := 2
	contentHeight := v.height - reservedLines
	if contentHeight < 1 {
		contentHeight = 1
	}
	maxOffset := len(v.lines) - contentHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	v.scrollOffset = maxOffset
}

// View renders the content viewer
func (v *ContentViewer) View() string {
	if len(v.lines) == 0 {
		return lipgloss.NewStyle().
			Foreground(styles.MutedColor).
			Render("No content available")
	}

	// If active, render with controls
	if v.active {
		return v.renderActiveDive()
	}

	// Preview mode - just show truncated content
	return v.renderPreviewContent()
}

// renderPreviewContent renders content in preview mode
func (v *ContentViewer) renderPreviewContent() string {
	// Calculate visible lines for preview
	maxLines := v.height
	if maxLines < 1 {
		maxLines = 1
	}

	var b strings.Builder
	truncated := false

	if len(v.lines) > maxLines {
		// Show preview lines
		for i := 0; i < maxLines-1; i++ {
			b.WriteString(v.lines[i])
			b.WriteString("\n")
		}
		truncated = true
	} else {
		// Show all lines
		for _, line := range v.lines {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Show truncation indicator with dive hint
	if truncated {
		hintStyle := lipgloss.NewStyle().
			Foreground(styles.MutedColor).
			Italic(true)
		hint := fmt.Sprintf("... (%d more lines) • Press Enter to dive in and scroll",
			len(v.lines)-maxLines+1)
		b.WriteString(hintStyle.Render(hint))
	}

	return b.String()
}

// checkAndWriteFile checks if file exists and writes or prompts for confirmation
func (v *ContentViewer) checkAndWriteFile() tea.Cmd {
	return func() tea.Msg {
		// Check if file exists
		if _, err := os.Stat(v.fileInput); err == nil {
			// File exists, prompt for overwrite
			v.confirmOverwrite = true
			return contentViewerFileCheckMsg{} // Trigger re-render to show confirmation prompt
		}

		// File doesn't exist, write directly
		return v.writeToFile()()
	}
}

// writeToFile writes the content to the specified file
func (v *ContentViewer) writeToFile() tea.Cmd {
	return func() tea.Msg {
		err := os.WriteFile(v.fileInput, []byte(v.content), 0600)
		if err != nil {
			v.statusMessage = fmt.Sprintf("Error writing file: %v", err)
		} else {
			v.statusMessage = fmt.Sprintf("Successfully wrote to %s", v.fileInput)
		}
		v.promptingFile = false
		v.confirmOverwrite = false
		v.fileInput = ""
		return contentViewerFileWrittenMsg{}
	}
}

// renderActiveDive renders content in active dive mode
func (v *ContentViewer) renderActiveDive() string {
	totalLines := len(v.lines)

	// Reserve lines for status (1) and controls (1)
	reservedLines := 2
	contentHeight := v.height - reservedLines
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Build status line
	statusStyle := lipgloss.NewStyle().
		Foreground(styles.AccentColor).
		Bold(true)

	var statusParts []string
	statusParts = append(statusParts, "▸ DIVE MODE")

	// Scroll indicators
	if totalLines > contentHeight {
		canScrollUp := v.scrollOffset > 0
		canScrollDown := v.scrollOffset+contentHeight < totalLines

		endLine := v.scrollOffset + contentHeight
		if endLine > totalLines {
			endLine = totalLines
		}

		scrollIndicator := fmt.Sprintf("Lines %d-%d of %d",
			v.scrollOffset+1,
			endLine,
			totalLines)
		if canScrollUp {
			scrollIndicator += " ↑"
		}
		if canScrollDown {
			scrollIndicator += " ↓"
		}
		statusParts = append(statusParts, scrollIndicator)
	}

	// Show status message or file prompt
	switch {
	case v.statusMessage != "":
		msgStyle := lipgloss.NewStyle().Foreground(styles.AccentColor)
		statusParts = append(statusParts, msgStyle.Render(v.statusMessage))
	case v.promptingFile:
		if v.confirmOverwrite {
			statusParts = append(statusParts, fmt.Sprintf("⚠ File '%s' exists. Overwrite? (y/n)", v.fileInput))
		} else {
			statusParts = append(statusParts, fmt.Sprintf("💾 Write to file: %s_", v.fileInput))
		}
	default:
		statusParts = append(statusParts, "w: Write to file")
	}

	status := statusStyle.Render(strings.Join(statusParts, " │ "))

	// Build controls line
	controlsStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor)

	var controls []string
	if v.promptingFile {
		if v.confirmOverwrite {
			controls = []string{"y: Confirm", "n/esc: Cancel"}
		} else {
			controls = []string{"Type filename", "enter: Confirm", "esc: Cancel"}
		}
	} else {
		controls = []string{"↑↓/jk: Scroll", "g/G: Top/Bottom", "w: Write", "esc: Exit"}
	}

	controlsText := controlsStyle.Render(strings.Join(controls, "  •  "))

	// Calculate visible content lines with bounds checking
	startLine := v.scrollOffset

	// Ensure startLine is within valid bounds
	if startLine < 0 {
		startLine = 0
	}
	if startLine > totalLines {
		startLine = totalLines
	}

	endLine := startLine + contentHeight
	if endLine > totalLines {
		endLine = totalLines
	}

	// Ensure we never have invalid slice bounds
	if startLine > endLine {
		startLine = endLine
	}

	var visibleLines []string
	if startLine < endLine && endLine <= len(v.lines) {
		visibleLines = v.lines[startLine:endLine]
	}

	// Build output
	var b strings.Builder
	b.WriteString(status)
	b.WriteString("\n")

	// Render content lines
	for _, line := range visibleLines {
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Fill remaining space with empty lines if needed
	renderedLines := len(visibleLines)
	for i := renderedLines; i < contentHeight; i++ {
		b.WriteString("\n")
	}

	b.WriteString(controlsText)

	return b.String()
}

// RenderPreview renders a preview with dive hint (public method for external use)
func (v *ContentViewer) RenderPreview() string {
	return v.renderPreviewContent()
}
