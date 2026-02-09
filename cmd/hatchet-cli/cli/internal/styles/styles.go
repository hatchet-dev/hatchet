package styles

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Primary colors
	Navy     = lipgloss.Color("#0A1029") // hsl(228 61% 10%)
	Cyan     = lipgloss.Color("#B8D9FF") // hsl(212 100% 86%)
	NavyDark = lipgloss.Color("#02081D") // hsl(227 88% 6%)
	CyanDark = lipgloss.Color("#A5C5E9") // hsl(212 60% 78%)
	Blue     = lipgloss.Color("#3392FF") // hsl(212 100% 60%)
	Magenta  = lipgloss.Color("#BC46DD") // hsl(287 69% 57%)
	Yellow   = lipgloss.Color("#B8D41C") // hsl(69 77% 47%)

	// Adaptive colors for light/dark terminals
	PrimaryColor   = lipgloss.AdaptiveColor{Light: "#0A1029", Dark: "#B8D9FF"}
	AccentColor    = lipgloss.AdaptiveColor{Light: "#3392FF", Dark: "#3392FF"}
	SuccessColor   = lipgloss.AdaptiveColor{Light: "#3392FF", Dark: "#3392FF"}
	HighlightColor = lipgloss.AdaptiveColor{Light: "#BC46DD", Dark: "#BC46DD"}
	MutedColor     = lipgloss.AdaptiveColor{Light: "#A5C5E9", Dark: "#A5C5E9"}

	// Status colors - matching frontend badge variants
	// Successful: green-300/green-800 with green-500/20 background
	StatusSuccessColor = lipgloss.AdaptiveColor{Light: "#166534", Dark: "#86efac"}
	StatusSuccessBg    = lipgloss.AdaptiveColor{Light: "#22c55e33", Dark: "#22c55e33"} // green-500/20

	// Failed: red-300/red-800 with red-500/20 background
	StatusFailedColor = lipgloss.AdaptiveColor{Light: "#991b1b", Dark: "#fca5a5"}
	StatusFailedBg    = lipgloss.AdaptiveColor{Light: "#ef444433", Dark: "#ef444433"} // red-500/20

	// In Progress: yellow-300/yellow-800 with yellow-500/20 background
	StatusInProgressColor = lipgloss.AdaptiveColor{Light: "#854d0e", Dark: "#fde047"}
	StatusInProgressBg    = lipgloss.AdaptiveColor{Light: "#eab30833", Dark: "#eab30833"} // yellow-500/20

	// Queued: slate-300/slate-800 with slate-500/20 background
	StatusQueuedColor = lipgloss.AdaptiveColor{Light: "#1e293b", Dark: "#cbd5e1"}
	StatusQueuedBg    = lipgloss.AdaptiveColor{Light: "#64748b33", Dark: "#64748b33"} // slate-500/20

	// Cancelled: orange-300/orange-800 with orange-500/20 background
	StatusCancelledColor = lipgloss.AdaptiveColor{Light: "#9a3412", Dark: "#fdba74"}
	StatusCancelledBg    = lipgloss.AdaptiveColor{Light: "#f9731633", Dark: "#f9731633"} // orange-500/20

	// Error color for general errors
	ErrorColor = lipgloss.AdaptiveColor{Light: "#dc2626", Dark: "#ff5555"}
)

// Common styles
var (
	// Base styles
	Bold   = lipgloss.NewStyle().Bold(true)
	Italic = lipgloss.NewStyle().Italic(true)

	// Headings
	H1 = lipgloss.NewStyle().
		Bold(true).
		Foreground(PrimaryColor).
		MarginTop(1).
		MarginBottom(1)

	H2 = lipgloss.NewStyle().
		Bold(true).
		Foreground(PrimaryColor).
		MarginBottom(1)

	// Emphasis styles
	Primary = lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true)

	Accent = lipgloss.NewStyle().
		Foreground(AccentColor).
		Bold(true)

	Success = lipgloss.NewStyle().
		Foreground(SuccessColor).
		Bold(true)

	Highlight = lipgloss.NewStyle().
			Foreground(HighlightColor).
			Bold(true)

	Muted = lipgloss.NewStyle().
		Foreground(MutedColor)

	// Code/mono style
	Code = lipgloss.NewStyle().
		Foreground(AccentColor).
		Background(lipgloss.AdaptiveColor{Light: "#F5F5F5", Dark: "#1a1a1a"}).
		Padding(0, 1)

	// Box styles
	Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(AccentColor).
		Padding(1, 2).
		MarginTop(1).
		MarginBottom(1)

	InfoBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(AccentColor).
		Padding(0, 1).
		MarginTop(1)

	SuccessBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(SuccessColor).
			Padding(0, 1).
			MarginTop(1)

	// List item style
	ListItem = lipgloss.NewStyle().
			PaddingLeft(2)
)

// Helper functions for common patterns

// Title renders a large title with optional icon
func Title(text string) string {
	return H1.Render("✦ " + text)
}

// Section renders a section header
func Section(text string) string {
	return H2.Render(text)
}

// KeyValue renders a key-value pair
func KeyValue(key, value string) string {
	keyStyle := Muted
	valueStyle := Primary
	return fmt.Sprintf("%s %s", keyStyle.Render(key+":"), valueStyle.Render(value))
}

// URL renders a URL in accent color
func URL(url string) string {
	return Accent.Render(url)
}

// Success renders a success message with checkmark
func SuccessMessage(message string) string {
	icon := Success.Render("✓")
	text := Primary.Render(message)
	return fmt.Sprintf("%s %s", icon, text)
}

// Info renders an info message with icon
func InfoMessage(message string) string {
	icon := Accent.Render("ℹ")
	text := Primary.Render(message)
	return fmt.Sprintf("%s %s", icon, text)
}

// List renders a bulleted list
func List(items []string) string {
	var lines []string
	bullet := Accent.Render("•")
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%s %s", bullet, item))
	}
	return strings.Join(lines, "\n")
}

// HatchetTheme creates a custom huh theme using Hatchet's color scheme
func HatchetTheme() *huh.Theme {
	t := huh.ThemeBase()

	// Focused field styles (active input)
	t.Focused.Base = t.Focused.Base.BorderForeground(AccentColor)
	t.Focused.Title = t.Focused.Title.Foreground(AccentColor).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(MutedColor)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(lipgloss.Color("#ff5555"))
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(lipgloss.Color("#ff5555"))

	// Select field styles
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(AccentColor)
	t.Focused.Option = t.Focused.Option.Foreground(PrimaryColor)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(AccentColor).Bold(true)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(SuccessColor).SetString("✓ ")
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(MutedColor).SetString("○ ")
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(PrimaryColor)

	// Button styles
	t.Focused.FocusedButton = t.Focused.FocusedButton.
		Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#0A1029"}).
		Background(AccentColor).
		Bold(true).
		Padding(0, 3)

	t.Focused.BlurredButton = t.Focused.BlurredButton.
		Foreground(MutedColor).
		Background(lipgloss.AdaptiveColor{Light: "#f0f0f0", Dark: "#1a1a1a"}).
		Padding(0, 3)

	// Text input styles
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(AccentColor)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(MutedColor)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(AccentColor)

	// Blurred field styles (inactive input)
	t.Blurred.Base = t.Blurred.Base.BorderForeground(MutedColor)
	t.Blurred.Title = t.Blurred.Title.Foreground(MutedColor)
	t.Blurred.Description = t.Blurred.Description.Foreground(MutedColor)
	t.Blurred.SelectSelector = t.Blurred.SelectSelector.Foreground(MutedColor)
	t.Blurred.TextInput.Prompt = t.Blurred.TextInput.Prompt.Foreground(MutedColor)
	t.Blurred.TextInput.Placeholder = t.Blurred.TextInput.Placeholder.Foreground(MutedColor)

	return t
}
