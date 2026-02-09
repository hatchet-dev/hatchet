package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

// RenderLogo returns the Hatchet logo as styled text
func RenderLogo() string {
	logoStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.AccentColor)

	return logoStyle.Render("HATCHET TUI")
}

// RenderHeaderWithLogo creates a header with the logo on the right side
// Returns the rendered header with proper spacing and borders
func RenderHeaderWithLogo(title string, width int) string {
	// Get the text logo
	logo := RenderLogo()

	// Calculate logo width
	logoWidth := lipgloss.Width(logo)

	// Create the title text
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.AccentColor).
		Padding(0, 1)

	titleText := titleStyle.Render(title)

	// Calculate available width for title (exclude padding, logo, and spacing)
	availableWidth := width - logoWidth - 8

	// Create left side (title) with proper width
	leftStyle := lipgloss.NewStyle().
		Width(availableWidth)

	leftSide := leftStyle.Render(titleText)

	// Join title and logo horizontally with some spacing
	content := lipgloss.JoinHorizontal(lipgloss.Center, leftSide, "  ", logo)

	// Wrap in border
	headerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(styles.AccentColor).
		Width(width-4).
		Padding(0, 1)

	return headerStyle.Render(content)
}
