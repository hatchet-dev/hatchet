package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/enums"
)

// StatusStyle defines the styling for a status badge
type StatusStyle struct {
	Text       string
	Foreground lipgloss.AdaptiveColor
	Background lipgloss.AdaptiveColor
}

// GetV1TaskStatusStyle returns the appropriate style for a V1TaskStatus
// Matches frontend createV1RunStatusVariant from run-statuses.tsx
func GetV1TaskStatusStyle(status rest.V1TaskStatus) StatusStyle {
	switch status {
	case enums.V1ReadableStatusOlapCOMPLETED:
		return StatusStyle{
			Text:       "Succeeded",
			Foreground: styles.StatusSuccessColor,
			Background: styles.StatusSuccessBg,
		}
	case enums.V1ReadableStatusOlapFAILED:
		return StatusStyle{
			Text:       "Failed",
			Foreground: styles.StatusFailedColor,
			Background: styles.StatusFailedBg,
		}
	case enums.V1ReadableStatusOlapCANCELLED:
		return StatusStyle{
			Text:       "Cancelled",
			Foreground: styles.StatusCancelledColor,
			Background: styles.StatusCancelledBg,
		}
	case enums.V1ReadableStatusOlapRUNNING:
		return StatusStyle{
			Text:       "Running",
			Foreground: styles.StatusInProgressColor,
			Background: styles.StatusInProgressBg,
		}
	case enums.V1ReadableStatusOlapQUEUED:
		return StatusStyle{
			Text:       "Queued",
			Foreground: styles.StatusQueuedColor,
			Background: styles.StatusQueuedBg,
		}
	default:
		return StatusStyle{
			Text:       "Unknown",
			Foreground: styles.MutedColor,
			Background: lipgloss.AdaptiveColor{Light: "#00000000", Dark: "#00000000"},
		}
	}
}

// FormatV1TaskStatusForTable returns plain text with a status indicator icon
// Use this for table cells since bubbles table doesn't support per-cell colors
func FormatV1TaskStatusForTable(status rest.V1TaskStatus) string {
	switch status {
	case enums.V1ReadableStatusOlapCOMPLETED:
		return "✓ Succeeded"
	case enums.V1ReadableStatusOlapFAILED:
		return "✗ Failed"
	case enums.V1ReadableStatusOlapCANCELLED:
		return "○ Cancelled"
	case enums.V1ReadableStatusOlapRUNNING:
		return "● Running"
	case enums.V1ReadableStatusOlapQUEUED:
		return "◦ Queued"
	default:
		return "? Unknown"
	}
}

// RenderV1TaskStatus renders a V1TaskStatus as a styled badge
// Matches the frontend Badge component styling
func RenderV1TaskStatus(status rest.V1TaskStatus) string {
	style := GetV1TaskStatusStyle(status)

	// For terminal, we only use foreground color (background doesn't render well in tables)
	badgeStyle := lipgloss.NewStyle().
		Foreground(style.Foreground).
		Bold(false)

	// Capitalize first letter
	text := strings.ToUpper(style.Text[0:1]) + strings.ToLower(style.Text[1:])

	return badgeStyle.Render(text)
}

// RenderError renders an error message in error style with text wrapping
func RenderError(message string, width int) string {
	errorStyle := lipgloss.NewStyle().
		Foreground(styles.ErrorColor).
		Padding(0, 1).
		Width(width - 4) // Account for padding and margins

	return errorStyle.Render(message)
}
