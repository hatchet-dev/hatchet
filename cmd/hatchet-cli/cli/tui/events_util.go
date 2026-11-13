package tui

import (
	"sort"

	"github.com/charmbracelet/lipgloss"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/enums"
)

// FormatEventType maps a V1TaskEventType enum to a human-readable string
// Mapping taken from frontend: frontend/app/src/pages/main/v1/workflow-runs-v1/$run/v2components/events-columns.tsx
func FormatEventType(eventType rest.V1TaskEventType) string {
	switch eventType {
	case enums.V1EventTypeOlapASSIGNED:
		return "Assigned to worker"
	case enums.V1EventTypeOlapSTARTED:
		return "Started"
	case enums.V1EventTypeOlapFINISHED:
		return "Completed"
	case enums.V1EventTypeOlapFAILED:
		return "Failed"
	case enums.V1EventTypeOlapCANCELLED:
		return "Cancelled"
	case enums.V1EventTypeOlapRETRYING:
		return "Retrying"
	case enums.V1EventTypeOlapREQUEUEDNOWORKER:
		return "Requeuing (no worker available)"
	case enums.V1EventTypeOlapREQUEUEDRATELIMIT:
		return "Requeuing (rate limit)"
	case enums.V1EventTypeOlapSCHEDULINGTIMEDOUT:
		return "Scheduling timed out"
	case enums.V1EventTypeOlapTIMEOUTREFRESHED:
		return "Timeout refreshed"
	case enums.V1EventTypeOlapREASSIGNED:
		return "Reassigned"
	case enums.V1EventTypeOlapTIMEDOUT:
		return "Execution timed out"
	case enums.V1EventTypeOlapSLOTRELEASED:
		return "Slot released"
	case enums.V1EventTypeOlapRETRIEDBYUSER:
		return "Replayed by user"
	case enums.V1EventTypeOlapACKNOWLEDGED:
		return "Acknowledged by worker"
	case enums.V1EventTypeOlapCREATED:
		return "Created"
	case enums.V1EventTypeOlapRATELIMITERROR:
		return "Rate limit error"
	case enums.V1EventTypeOlapSENTTOWORKER:
		return "Sent to worker"
	case enums.V1EventTypeOlapQUEUED:
		return "Queued"
	case enums.V1EventTypeOlapSKIPPED:
		return "Skipped"
	case enums.V1EventTypeOlapCOULDNOTSENDTOWORKER:
		return "Could not send to worker"
	default:
		return "Unknown"
	}
}

// SortEventsByTimestamp sorts events by timestamp in descending order (newest first)
func SortEventsByTimestamp(events []rest.V1TaskEvent) []rest.V1TaskEvent {
	// Create a copy to avoid modifying the original slice
	sorted := make([]rest.V1TaskEvent, len(events))
	copy(sorted, events)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.After(sorted[j].Timestamp)
	})

	return sorted
}

// EventSeverity represents the severity level of a task event
type EventSeverity int

const (
	EventSeverityInfo EventSeverity = iota
	EventSeverityWarning
	EventSeverityCritical
)

// GetEventSeverity returns the severity level for a given event type
func GetEventSeverity(eventType rest.V1TaskEventType) EventSeverity {
	switch eventType {
	// CRITICAL: Red events
	case enums.V1EventTypeOlapFAILED,
		enums.V1EventTypeOlapRATELIMITERROR,
		enums.V1EventTypeOlapSCHEDULINGTIMEDOUT,
		enums.V1EventTypeOlapTIMEDOUT,
		enums.V1EventTypeOlapCANCELLED:
		return EventSeverityCritical

	// WARNING: Yellow events
	case enums.V1EventTypeOlapREASSIGNED,
		enums.V1EventTypeOlapREQUEUEDNOWORKER,
		enums.V1EventTypeOlapREQUEUEDRATELIMIT,
		enums.V1EventTypeOlapRETRIEDBYUSER,
		enums.V1EventTypeOlapRETRYING:
		return EventSeverityWarning

	// INFO: Green events (default)
	default:
		return EventSeverityInfo
	}
}

// RenderEventSeverityDot returns a colored dot indicator for the severity level
func RenderEventSeverityDot(severity EventSeverity) string {
	dot := "●"
	var style lipgloss.Style

	switch severity {
	case EventSeverityCritical:
		// Red
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
	case EventSeverityWarning:
		// Yellow
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#eab308"))
	case EventSeverityInfo:
		// Green
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	default:
		// Default gray
		style = lipgloss.NewStyle().Foreground(styles.MutedColor)
	}

	return style.Render(dot)
}
