package tui

import (
	"sort"

	"github.com/charmbracelet/lipgloss"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// FormatEventType maps a V1TaskEventType enum to a human-readable string
// Mapping taken from frontend: frontend/app/src/pages/main/v1/workflow-runs-v1/$run/v2components/events-columns.tsx
func FormatEventType(eventType rest.V1TaskEventType) string {
	switch eventType {
	case rest.V1TaskEventTypeASSIGNED:
		return "Assigned to worker"
	case rest.V1TaskEventTypeSTARTED:
		return "Started"
	case rest.V1TaskEventTypeFINISHED:
		return "Completed"
	case rest.V1TaskEventTypeFAILED:
		return "Failed"
	case rest.V1TaskEventTypeCANCELLED:
		return "Cancelled"
	case rest.V1TaskEventTypeRETRYING:
		return "Retrying"
	case rest.V1TaskEventTypeREQUEUEDNOWORKER:
		return "Requeuing (no worker available)"
	case rest.V1TaskEventTypeREQUEUEDRATELIMIT:
		return "Requeuing (rate limit)"
	case rest.V1TaskEventTypeSCHEDULINGTIMEDOUT:
		return "Scheduling timed out"
	case rest.V1TaskEventTypeTIMEOUTREFRESHED:
		return "Timeout refreshed"
	case rest.V1TaskEventTypeREASSIGNED:
		return "Reassigned"
	case rest.V1TaskEventTypeTIMEDOUT:
		return "Execution timed out"
	case rest.V1TaskEventTypeSLOTRELEASED:
		return "Slot released"
	case rest.V1TaskEventTypeRETRIEDBYUSER:
		return "Replayed by user"
	case rest.V1TaskEventTypeACKNOWLEDGED:
		return "Acknowledged by worker"
	case rest.V1TaskEventTypeCREATED:
		return "Created"
	case rest.V1TaskEventTypeRATELIMITERROR:
		return "Rate limit error"
	case rest.V1TaskEventTypeSENTTOWORKER:
		return "Sent to worker"
	case rest.V1TaskEventTypeQUEUED:
		return "Queued"
	case rest.V1TaskEventTypeSKIPPED:
		return "Skipped"
	case rest.V1TaskEventTypeCOULDNOTSENDTOWORKER:
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
	case rest.V1TaskEventTypeFAILED,
		rest.V1TaskEventTypeRATELIMITERROR,
		rest.V1TaskEventTypeSCHEDULINGTIMEDOUT,
		rest.V1TaskEventTypeTIMEDOUT,
		rest.V1TaskEventTypeCANCELLED:
		return EventSeverityCritical

	// WARNING: Yellow events
	case rest.V1TaskEventTypeREASSIGNED,
		rest.V1TaskEventTypeREQUEUEDNOWORKER,
		rest.V1TaskEventTypeREQUEUEDRATELIMIT,
		rest.V1TaskEventTypeRETRIEDBYUSER,
		rest.V1TaskEventTypeRETRYING:
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
