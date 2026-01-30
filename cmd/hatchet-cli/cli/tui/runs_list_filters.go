package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// RunsListFilters holds the current filter state
type RunsListFilters struct {
	Since       time.Time
	Statuses    map[rest.V1TaskStatus]bool
	Until       *time.Time
	TimeWindow  string
	WorkflowIDs []string
}

// NewDefaultRunsListFilters creates default filters matching frontend defaults
func NewDefaultRunsListFilters() *RunsListFilters {
	return &RunsListFilters{
		WorkflowIDs: []string{}, // Empty means all workflows
		Statuses: map[rest.V1TaskStatus]bool{
			rest.V1TaskStatusCOMPLETED: true,
			rest.V1TaskStatusFAILED:    true,
			rest.V1TaskStatusCANCELLED: true,
			rest.V1TaskStatusRUNNING:   true,
			rest.V1TaskStatusQUEUED:    true,
		},
		TimeWindow: "1d", // 24 hours default
		Since:      time.Now().Add(-24 * time.Hour),
		Until:      nil,
	}
}

// GetTimeRangeFromWindow converts time window string to time.Time
func GetTimeRangeFromWindow(window string) time.Time {
	switch window {
	case "1h":
		return time.Now().Add(-1 * time.Hour)
	case "6h":
		return time.Now().Add(-6 * time.Hour)
	case "1d":
		return time.Now().Add(-24 * time.Hour)
	case "7d":
		return time.Now().Add(-7 * 24 * time.Hour)
	default:
		return time.Now().Add(-24 * time.Hour)
	}
}

// GetActiveStatuses returns a slice of enabled statuses
func (f *RunsListFilters) GetActiveStatuses() []rest.V1TaskStatus {
	statuses := []rest.V1TaskStatus{}
	for status, enabled := range f.Statuses {
		if enabled {
			statuses = append(statuses, status)
		}
	}
	return statuses
}

// customFilterKeyMap creates a custom keymap where down arrow exits filter mode
func customFilterKeyMap() *huh.KeyMap {
	km := huh.NewDefaultKeyMap()
	// Change SetFilter from "enter/esc" to "down" - this exits filter mode
	km.MultiSelect.SetFilter = key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "exit filter"),
		key.WithDisabled(),
	)
	return km
}

// BuildRunsListFiltersForm builds a huh.Form for editing filters
// This form is meant to be embedded directly in the main tea.Program
// Returns the form and a pointer to the status slice that will be modified
func BuildRunsListFiltersForm(filters *RunsListFilters, workflows []WorkflowOption) (*huh.Form, *[]rest.V1TaskStatus) {
	// Build workflow options
	workflowOptions := []huh.Option[string]{}
	for _, wf := range workflows {
		workflowOptions = append(workflowOptions, huh.NewOption(wf.DisplayName, wf.ID))
	}

	// Build time window options
	timeWindowOptions := []huh.Option[string]{
		huh.NewOption("Last Hour", "1h"),
		huh.NewOption("Last 6 Hours", "6h"),
		huh.NewOption("Last 24 Hours", "1d"),
		huh.NewOption("Last 7 Days", "7d"),
	}

	// Create a status slice that the multiselect can modify
	statusSlice := currentFiltersToSlice(filters)

	form := huh.NewForm(
		// Workflow multiselect - separate group
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Workflows").
				Description("x/space to toggle | / to filter, ↓ to exit filter | Enter to confirm").
				Options(workflowOptions...).
				Value(&filters.WorkflowIDs).
				Filterable(true). // Enable search with /
				Height(10),       // Limit visible options
		),

		// Time window selector - separate group
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Time Range").
				Description("Tab for next field").
				Options(timeWindowOptions...).
				Value(&filters.TimeWindow),
		),

		// Status checkboxes - separate group
		huh.NewGroup(
			huh.NewMultiSelect[rest.V1TaskStatus]().
				Title("Statuses").
				Description("x/space to toggle | Enter to confirm").
				Options(
					huh.NewOption("Completed", rest.V1TaskStatusCOMPLETED),
					huh.NewOption("Failed", rest.V1TaskStatusFAILED),
					huh.NewOption("Cancelled", rest.V1TaskStatusCANCELLED),
					huh.NewOption("Running", rest.V1TaskStatusRUNNING),
					huh.NewOption("Queued", rest.V1TaskStatusQUEUED),
				).
				Value(statusSlice).
				Filterable(false),
		),
	).WithTheme(styles.HatchetTheme()).
		WithKeyMap(customFilterKeyMap()).
		WithShowHelp(false).
		WithShowErrors(false)

	return form, statusSlice
}

// RunFiltersFormProgram runs the filters form in a separate tea.Program
// This ensures it takes full control of the terminal without interference
func RunFiltersFormProgram(currentFilters *RunsListFilters, workflows []WorkflowOption) (*RunsListFilters, error) {
	model := &filterFormModel{
		currentFilters: currentFilters,
		workflows:      workflows,
		done:           false,
	}

	// Create a new program with input options to ensure it captures all input
	p := tea.NewProgram(
		model,
		tea.WithFilter(func(m tea.Model, msg tea.Msg) tea.Msg {
			// Only allow the filter form to receive messages
			return msg
		}),
	)

	finalModel, err := p.Run()
	if err != nil {
		return currentFilters, err
	}

	result := finalModel.(*filterFormModel)
	if result.cancelled {
		return currentFilters, fmt.Errorf("cancelled")
	}

	return result.newFilters, nil
}

// filterFormModel wraps the huh form to run in a tea.Program
type filterFormModel struct {
	form           *huh.Form
	currentFilters *RunsListFilters
	newFilters     *RunsListFilters
	workflows      []WorkflowOption
	done           bool
	cancelled      bool
}

func (m *filterFormModel) Init() tea.Cmd {
	// Build the form
	newFilters := &RunsListFilters{
		WorkflowIDs: append([]string{}, m.currentFilters.WorkflowIDs...),
		Statuses:    make(map[rest.V1TaskStatus]bool),
		TimeWindow:  m.currentFilters.TimeWindow,
		Since:       m.currentFilters.Since,
		Until:       m.currentFilters.Until,
	}

	// Copy status map
	for k, v := range m.currentFilters.Statuses {
		newFilters.Statuses[k] = v
	}

	m.newFilters = newFilters

	// Build workflow options
	workflowOptions := []huh.Option[string]{
		huh.NewOption("All Workflows", ""),
	}
	for _, wf := range m.workflows {
		workflowOptions = append(workflowOptions, huh.NewOption(wf.DisplayName, wf.ID))
	}

	// Build time window options
	timeWindowOptions := []huh.Option[string]{
		huh.NewOption("Last Hour", "1h"),
		huh.NewOption("Last 6 Hours", "6h"),
		huh.NewOption("Last 24 Hours", "1d"),
		huh.NewOption("Last 7 Days", "7d"),
	}

	m.form = huh.NewForm(
		// Workflow multiselect - separate group
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Workflows").
				Description("x/space to toggle | / to filter, ↓ to exit filter | Enter to confirm").
				Options(workflowOptions...).
				Value(&m.newFilters.WorkflowIDs).
				Filterable(true). // Enable search with /
				Height(10),       // Limit visible options
		),

		// Time window selector - separate group
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Time Range").
				Description("Press Tab for next field").
				Options(timeWindowOptions...).
				Value(&m.newFilters.TimeWindow),
		),

		// Status checkboxes - separate group
		huh.NewGroup(
			huh.NewMultiSelect[rest.V1TaskStatus]().
				Title("Statuses").
				Description("x/space to toggle | Enter to confirm").
				Options(
					huh.NewOption("Completed", rest.V1TaskStatusCOMPLETED),
					huh.NewOption("Failed", rest.V1TaskStatusFAILED),
					huh.NewOption("Cancelled", rest.V1TaskStatusCANCELLED),
					huh.NewOption("Running", rest.V1TaskStatusRUNNING),
					huh.NewOption("Queued", rest.V1TaskStatusQUEUED),
				).
				Value(currentFiltersToSlice(m.currentFilters)).
				Filterable(false),
		),
	).WithTheme(styles.HatchetTheme()).
		WithKeyMap(customFilterKeyMap()).
		WithShowHelp(false). // Disable help to reduce double-press confusion
		WithShowErrors(false)

	return m.form.Init()
}

func (m *filterFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc", "ctrl+c":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		}
	}

	// Update the form
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	// Check if form is complete
	if m.form.State == huh.StateCompleted {
		m.done = true

		// Update time range based on window
		if m.newFilters.TimeWindow != "custom" {
			m.newFilters.Since = GetTimeRangeFromWindow(m.newFilters.TimeWindow)
			m.newFilters.Until = nil
		}

		return m, tea.Quit
	}

	return m, cmd
}

func (m *filterFormModel) View() string {
	if m.done {
		return ""
	}
	return m.form.View()
}

// ShowFiltersForm displays a form to edit filters (deprecated, use RunFiltersFormProgram)
func ShowFiltersForm(currentFilters *RunsListFilters, workflows []WorkflowOption) (*RunsListFilters, error) {
	newFilters := &RunsListFilters{
		WorkflowIDs: append([]string{}, currentFilters.WorkflowIDs...),
		Statuses:    make(map[rest.V1TaskStatus]bool),
		TimeWindow:  currentFilters.TimeWindow,
		Since:       currentFilters.Since,
		Until:       currentFilters.Until,
	}

	// Copy status map
	for k, v := range currentFilters.Statuses {
		newFilters.Statuses[k] = v
	}

	// Build workflow options
	workflowOptions := []huh.Option[string]{
		huh.NewOption("All Workflows", ""),
	}
	for _, wf := range workflows {
		workflowOptions = append(workflowOptions, huh.NewOption(wf.DisplayName, wf.ID))
	}

	// Build time window options
	timeWindowOptions := []huh.Option[string]{
		huh.NewOption("Last Hour", "1h"),
		huh.NewOption("Last 6 Hours", "6h"),
		huh.NewOption("Last 24 Hours", "1d"),
		huh.NewOption("Last 7 Days", "7d"),
	}

	form := huh.NewForm(
		// Workflow multiselect - separate group
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Workflows").
				Description("x/space to toggle | / to filter, ↓ to exit filter | Enter to confirm").
				Options(workflowOptions...).
				Value(&newFilters.WorkflowIDs).
				Filterable(true). // Enable search with /
				Height(10),       // Limit visible options
		),

		// Time window selector - separate group
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Time Range").
				Options(timeWindowOptions...).
				Value(&newFilters.TimeWindow),
		),

		// Status checkboxes - separate group
		huh.NewGroup(
			huh.NewMultiSelect[rest.V1TaskStatus]().
				Title("Statuses").
				Options(
					huh.NewOption("Completed", rest.V1TaskStatusCOMPLETED),
					huh.NewOption("Failed", rest.V1TaskStatusFAILED),
					huh.NewOption("Cancelled", rest.V1TaskStatusCANCELLED),
					huh.NewOption("Running", rest.V1TaskStatusRUNNING),
					huh.NewOption("Queued", rest.V1TaskStatusQUEUED),
				).
				Value(currentFiltersToSlice(currentFilters)).
				Filterable(false),
		),
	).WithTheme(styles.HatchetTheme()).
		WithKeyMap(customFilterKeyMap())

	// Run the form directly - this will be called from a tea.Exec command
	// which suspends the parent program
	err := form.Run()
	if err != nil {
		return currentFilters, err
	}

	// Update time range based on window
	if newFilters.TimeWindow != "custom" {
		newFilters.Since = GetTimeRangeFromWindow(newFilters.TimeWindow)
		newFilters.Until = nil
	}

	return newFilters, nil
}

// currentFiltersToSlice converts status map to slice for multiselect
func currentFiltersToSlice(filters *RunsListFilters) *[]rest.V1TaskStatus {
	statuses := []rest.V1TaskStatus{}
	for status, enabled := range filters.Statuses {
		if enabled {
			statuses = append(statuses, status)
		}
	}
	return &statuses
}

// WorkflowOption represents a workflow for the selector
type WorkflowOption struct {
	ID          string
	DisplayName string
}

// FetchWorkflows fetches available workflows for filtering
func FetchWorkflows(ctx context.Context, client rest.ClientWithResponsesInterface, tenantID uuid.UUID) ([]WorkflowOption, error) {
	// Parse tenant ID as UUID
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	// Fetch workflows list
	resp, err := client.WorkflowListWithResponse(ctx, tenantUUID, &rest.WorkflowListParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflows: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response from API")
	}

	rows := *resp.JSON200.Rows
	workflows := make([]WorkflowOption, 0, len(rows))
	for _, wf := range rows {
		workflows = append(workflows, WorkflowOption{
			ID:          wf.Metadata.Id,
			DisplayName: wf.Name,
		})
	}

	return workflows, nil
}
