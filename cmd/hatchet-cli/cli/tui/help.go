package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

// HelpView displays a fullscreen help modal with ASCII art logo and command reference
type HelpView struct {
	viewport viewport.Model
	BaseModel
	ready bool
}

// NewHelpView creates a new help view
func NewHelpView(ctx ViewContext) *HelpView {
	return &HelpView{
		BaseModel: BaseModel{
			Ctx: ctx,
		},
	}
}

// Init initializes the view
func (v *HelpView) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the view state
func (v *HelpView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.SetSize(msg.Width, msg.Height)

		// Initialize viewport with proper dimensions
		if !v.ready {
			// Height: total height - header (3) - footer (3) - spacing (4)
			viewportHeight := msg.Height - 10
			if viewportHeight < 10 {
				viewportHeight = 10
			}

			v.viewport = viewport.New(msg.Width-4, viewportHeight)
			v.viewport.YPosition = 0 // Start at top
			v.viewport.SetContent(v.buildHelpContent())
			v.ready = true
		} else {
			// Update existing viewport size
			viewportHeight := msg.Height - 10
			if viewportHeight < 10 {
				viewportHeight = 10
			}
			v.viewport.Width = msg.Width - 4
			v.viewport.Height = viewportHeight
			v.viewport.SetContent(v.buildHelpContent())
		}

		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "h", "esc":
			// Close help view
			return v, NewNavigateBackMsg()
		case "down", "j":
			// Scroll down
			v.viewport, cmd = v.viewport.Update(msg)
			return v, cmd
		case "up", "k":
			// Scroll up
			v.viewport, cmd = v.viewport.Update(msg)
			return v, cmd
		case "pgdown", "space":
			// Page down
			v.viewport, cmd = v.viewport.Update(msg)
			return v, cmd
		case "pgup":
			// Page up
			v.viewport, cmd = v.viewport.Update(msg)
			return v, cmd
		case "home", "g":
			// Go to top
			v.viewport.GotoTop()
			return v, nil
		case "end", "G":
			// Go to bottom
			v.viewport.GotoBottom()
			return v, nil
		}
	}

	// Update viewport for mouse wheel events
	if v.ready {
		v.viewport, cmd = v.viewport.Update(msg)
		return v, cmd
	}

	return v, nil
}

// View renders the help view
func (v *HelpView) View() string {
	if v.Width == 0 {
		return "Initializing..."
	}

	if !v.ready {
		return "Loading help..."
	}

	var b strings.Builder

	// Header (using reusable component)
	header := RenderHeader("Help", v.Ctx.ProfileName, v.Width)
	b.WriteString(header)
	b.WriteString("\n\n")

	// Viewport with scrollable content
	b.WriteString(v.viewport.View())
	b.WriteString("\n")

	// Scroll indicator
	scrollInfo := ""
	if v.viewport.TotalLineCount() > v.viewport.Height {
		percentage := int(v.viewport.ScrollPercent() * 100)
		scrollInfo = lipgloss.NewStyle().
			Foreground(styles.AccentColor).
			Padding(0, 1).
			Render(lipgloss.JoinHorizontal(
				lipgloss.Left,
				strings.Repeat("─", (v.Width-12)*percentage/100),
				"●",
				strings.Repeat("─", (v.Width-12)*(100-percentage)/100),
			))
	}
	if scrollInfo != "" {
		b.WriteString(scrollInfo)
		b.WriteString("\n")
	}

	// Footer with scroll controls (using reusable component)
	footer := RenderFooter([]string{
		"↑/↓ j/k: Scroll",
		"g/G: Top/Bottom",
		"space/pgdn: Page Down",
		"h/esc: Close Help",
		"q: Quit",
	}, v.Width)
	b.WriteString(footer)

	return b.String()
}

// buildHelpContent builds the full help content including logo and sections
func (v *HelpView) buildHelpContent() string {
	var b strings.Builder

	// ASCII Art Logo - centered
	logo := v.renderLogo()
	b.WriteString(logo)
	b.WriteString("\n\n")

	// Help content - organized by category
	helpContent := v.renderHelpContent()
	b.WriteString(helpContent)

	return b.String()
}

// renderLogo renders the ASCII art Hatchet logo, centered
func (v *HelpView) renderLogo() string {
	logoLines := []string{
		"                                                                                                    ",
		"                                                                                                    ",
		"      ....... ...                                                                                  ",
		"    .......  ......         ...   ....    .....  .........  .......   ...   ...  ........ .........",
		"  .......   .........       ....  ....    .....  ......... .........  ....  .... ........ .........",
		"........    ..........      ....  ....   .......    ...    ....  .... ....  .... ....        ....  ",
		"......           ......     ..........   .......    ...    ...        .......... ........    ....  ",
		"...........    ........     ....  ....  ... .....   ...    ....  .... ....  .... ....        ....  ",
		"  ........    .......       ....  .... ..........   ...    .........  ....  .... ........    ....  ",
		"    ......  .......         ....   ... ....   ...   ...     .......   ...   .... ........    ....  ",
		"     .... .......                                                                                  ",
		"                                                                                                    ",
		"                                                                                                    ",
	}

	// Style the logo with cyan color and center it
	logoStyle := lipgloss.NewStyle().
		Foreground(styles.Cyan).
		Bold(true).
		Align(lipgloss.Center).
		Width(v.Width - 4)

	var logoBuilder strings.Builder
	for _, line := range logoLines {
		logoBuilder.WriteString(logoStyle.Render(line))
		logoBuilder.WriteString("\n")
	}

	return logoBuilder.String()
}

// SetSize updates the view dimensions and initializes the viewport
func (v *HelpView) SetSize(width, height int) {
	v.BaseModel.SetSize(width, height)

	// Initialize viewport with proper dimensions
	if !v.ready && width > 0 && height > 0 {
		// Height: total height - header (3) - footer (3) - spacing (4)
		viewportHeight := height - 10
		if viewportHeight < 10 {
			viewportHeight = 10
		}

		v.viewport = viewport.New(width-4, viewportHeight)
		v.viewport.YPosition = 0 // Start at top
		v.viewport.SetContent(v.buildHelpContent())
		v.ready = true
	} else if v.ready {
		// Update existing viewport size
		viewportHeight := height - 10
		if viewportHeight < 10 {
			viewportHeight = 10
		}
		v.viewport.Width = width - 4
		v.viewport.Height = viewportHeight
		v.viewport.SetContent(v.buildHelpContent())
	}
}

// renderHelpContent renders the help content organized by category
func (v *HelpView) renderHelpContent() string {
	var b strings.Builder

	// Content container style
	contentStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Width(v.Width - 8)

	// Section header style
	sectionStyle := lipgloss.NewStyle().
		Foreground(styles.HighlightColor).
		Bold(true).
		Padding(0, 2)

	// Command style
	commandStyle := lipgloss.NewStyle().
		Foreground(styles.AccentColor).
		Padding(0, 2)

	// Description style
	descStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		Padding(0, 2)

	// Welcome message
	welcomeStyle := lipgloss.NewStyle().
		Foreground(styles.PrimaryColor).
		Align(lipgloss.Center).
		Width(v.Width-4).
		Padding(0, 2)
	b.WriteString(welcomeStyle.Render("Welcome to Hatchet TUI - Your Terminal Interface for Hatchet"))
	b.WriteString("\n\n")

	// Navigation section
	b.WriteString(sectionStyle.Render("━━━ Navigation ━━━"))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("v or shift+tab") + " " + descStyle.Render("Switch between primary views (Runs, Workflows, Workers)")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("↑/↓, j/k") + "       " + descStyle.Render("Navigate through lists")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("enter") + "          " + descStyle.Render("Select item / View details")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("esc") + "            " + descStyle.Render("Go back / Cancel")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("mouse scroll") + "   " + descStyle.Render("Scroll through lists (wheel up/down)")))
	b.WriteString("\n\n")

	// View Controls section
	b.WriteString(sectionStyle.Render("━━━ View Controls ━━━"))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("r") + "              " + descStyle.Render("Refresh current view")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("f") + "              " + descStyle.Render("Open filter modal (Runs view)")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("→ (right)") + "     " + descStyle.Render("Next page (when available)")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("← (left)") + "      " + descStyle.Render("Previous page (when available)")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("d") + "              " + descStyle.Render("Toggle debug view")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("h") + "              " + descStyle.Render("Show this help screen")))
	b.WriteString("\n\n")

	// Debug Controls section
	b.WriteString(sectionStyle.Render("━━━ Debug Mode ━━━"))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("d") + "              " + descStyle.Render("Toggle debug view on/off")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("c") + "              " + descStyle.Render("Clear debug logs")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("w") + "              " + descStyle.Render("Write debug logs to file")))
	b.WriteString("\n\n")

	// Views section
	b.WriteString(sectionStyle.Render("━━━ Available Views ━━━"))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("Runs") + "           " + descStyle.Render("View and manage task runs with filtering")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("Workflows") + "      " + descStyle.Render("Browse workflows and view recent runs")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("Workers") + "        " + descStyle.Render("Monitor workers and their status")))
	b.WriteString("\n\n")

	// Filter Modal section
	b.WriteString(sectionStyle.Render("━━━ Filter Modal (Runs View) ━━━"))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("tab/shift+tab") + " " + descStyle.Render("Navigate between filter fields")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("space") + "         " + descStyle.Render("Select/deselect (multi-select fields)")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("type") + "          " + descStyle.Render("Search in workflow selector")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("enter") + "         " + descStyle.Render("Apply filters")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("esc") + "           " + descStyle.Render("Cancel without applying")))
	b.WriteString("\n\n")

	// General section
	b.WriteString(sectionStyle.Render("━━━ General ━━━"))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("shift+p") + "       " + descStyle.Render("Switch profile")))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(commandStyle.Render("q, ctrl+c") + "    " + descStyle.Render("Quit Hatchet TUI")))
	b.WriteString("\n\n")

	// Tips section
	b.WriteString(sectionStyle.Render("━━━ Tips ━━━"))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render("• Data refreshes automatically every 5 seconds"))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render("• Use debug mode (d) to see API call logs and troubleshoot issues"))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render("• Filters persist until you change them or quit the TUI"))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render("• Mouse support: scroll with wheel, click to select (in some terminals)"))
	b.WriteString("\n")

	return b.String()
}
