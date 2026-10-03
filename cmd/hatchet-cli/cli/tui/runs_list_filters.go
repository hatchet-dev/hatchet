package tui

import (
	"context"
	"fmt"
	"sync"
	"time"

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

// BuildRunsListFiltersForm builds a huh.Form for editing filters
// This form is meant to be embedded directly in the main tea.Program
// Returns the form and a pointer to the status slice that will be modified
// The workflow list is searched server-side as the user types, like the
// frontend does, instead of loading every workflow up front.
func BuildRunsListFiltersForm(filters *RunsListFilters, workflows []WorkflowOption, client rest.ClientWithResponsesInterface, tenantID string) (*huh.Form, *[]rest.V1TaskStatus) {
	search := ""

	// Names for workflow IDs we've seen, so selected workflows keep their
	// label even when the current search doesn't return them
	var namesMu sync.Mutex
	knownNames := map[string]string{}
	for _, wf := range workflows {
		knownNames[wf.ID] = wf.DisplayName
	}

	workflowOptions := func() []huh.Option[string] {
		rows := []rest.Workflow{}
		if tenantUUID, err := uuid.Parse(tenantID); err == nil {
			if fetched, err := SearchWorkflows(context.Background(), client, tenantUUID, search); err == nil {
				rows = fetched
			}
		}

		namesMu.Lock()
		defer namesMu.Unlock()
		for _, wf := range rows {
			knownNames[wf.Metadata.Id] = wf.Name
		}

		// Selected workflows always stay in the list, otherwise huh drops
		// them from the value when a search narrows the options
		options := []huh.Option[string]{}
		seen := map[string]bool{}
		for _, id := range filters.WorkflowIDs {
			seen[id] = true
			name := knownNames[id]
			if name == "" {
				name = id
			}
			options = append(options, huh.NewOption(name, id))
		}
		for _, wf := range rows {
			if !seen[wf.Metadata.Id] {
				options = append(options, huh.NewOption(wf.Name, wf.Metadata.Id))
			}
		}
		return options
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
		// Workflow search + multiselect - separate group
		huh.NewGroup(
			huh.NewInput().
				Title("Workflows").
				Description("Type to search all workflows | Enter/Tab for list").
				Placeholder("Search workflows...").
				Value(&search),
			huh.NewMultiSelect[string]().
				Description("x/space to toggle | Enter to confirm").
				OptionsFunc(workflowOptions, &search).
				Value(&filters.WorkflowIDs).
				Filterable(false).
				Height(10), // Limit visible options
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
		WithShowHelp(false).
		WithShowErrors(false)

	return form, statusSlice
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

// SearchWorkflows fetches one page of workflows matching the search string,
// using the same server-side search and page size as the frontend
func SearchWorkflows(ctx context.Context, client rest.ClientWithResponsesInterface, tenantUUID uuid.UUID, name string) ([]rest.Workflow, error) {
	limit := 200
	params := &rest.WorkflowListParams{Limit: &limit}
	if name != "" {
		params.Name = &name
	}

	resp, err := client.WorkflowListWithResponse(ctx, tenantUUID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflows: %w", err)
	}

	if resp.JSON200 == nil || resp.JSON200.Rows == nil {
		return nil, fmt.Errorf("unexpected response from API (status %d)", resp.StatusCode())
	}

	return *resp.JSON200.Rows, nil
}

// FetchWorkflows fetches available workflows for filtering
func FetchWorkflows(ctx context.Context, client rest.ClientWithResponsesInterface, tenantID string) ([]WorkflowOption, error) {
	// Parse tenant ID as UUID
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	rows, err := SearchWorkflows(ctx, client, tenantUUID, "")
	if err != nil {
		return nil, err
	}

	workflows := make([]WorkflowOption, 0, len(rows))
	for _, wf := range rows {
		workflows = append(workflows, WorkflowOption{
			ID:          wf.Metadata.Id,
			DisplayName: wf.Name,
		})
	}

	return workflows, nil
}
