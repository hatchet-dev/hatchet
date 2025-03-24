package features

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// SchedulesClient provides methods for interacting with workflow schedules
// in the Hatchet platform.
type SchedulesClient interface {
	// Create creates a new scheduled workflow run.
	Create(workflowName string, trigger CreateScheduledRunTrigger, ctx ...context.Context) (*rest.ScheduledWorkflows, error)

	// Delete removes a scheduled workflow run.
	Delete(scheduledRunId string, ctx ...context.Context) error

	// List retrieves a collection of scheduled workflow runs based on the provided parameters.
	List(opts rest.WorkflowScheduledListParams, ctx ...context.Context) (*rest.ScheduledWorkflowsList, error)

	// Get retrieves a specific scheduled workflow run by its ID.
	Get(scheduledRunId string, ctx ...context.Context) (*rest.ScheduledWorkflows, error)
}

// CreateScheduledRunTrigger contains the configuration for creating a scheduled run trigger.
type CreateScheduledRunTrigger struct {
	// TriggerAt specifies when the workflow should be triggered.
	TriggerAt time.Time `json:"triggerAt"`

	// Input is the optional input data for the workflow.
	Input map[string]interface{} `json:"input,omitempty"`

	// AdditionalMetadata is optional metadata to associate with the scheduled run.
	AdditionalMetadata map[string]interface{} `json:"additionalMetadata,omitempty"`
}

// schedulesClientImpl implements the SchedulesClient interface.
type schedulesClientImpl struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

// NewSchedulesClient creates a new client for interacting with workflow schedules.
func NewSchedulesClient(
	api *rest.ClientWithResponses,
	tenantId *string,
) SchedulesClient {
	tenantIdUUID := uuid.MustParse(*tenantId)

	return &schedulesClientImpl{
		api:      api,
		tenantId: tenantIdUUID,
	}
}

// Create creates a new scheduled workflow run.
func (s *schedulesClientImpl) Create(workflowName string, trigger CreateScheduledRunTrigger, ctx ...context.Context) (*rest.ScheduledWorkflows, error) {

	request := rest.ScheduleWorkflowRunRequest{
		Input:              trigger.Input,
		AdditionalMetadata: trigger.AdditionalMetadata,
		TriggerAt:          trigger.TriggerAt,
	}

	resp, err := s.api.ScheduledWorkflowRunCreateWithResponse(
		getContext(ctx...),
		s.tenantId,
		workflowName,
		request,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Delete removes a scheduled workflow run.
func (s *schedulesClientImpl) Delete(scheduledRunId string, ctx ...context.Context) error {
	scheduledRunIdUUID, err := uuid.Parse(scheduledRunId)
	if err != nil {
		return err
	}

	_, err = s.api.WorkflowScheduledDeleteWithResponse(
		getContext(ctx...),
		s.tenantId,
		scheduledRunIdUUID,
	)
	return err
}

// List retrieves a collection of scheduled workflow runs based on the provided parameters.
func (s *schedulesClientImpl) List(opts rest.WorkflowScheduledListParams, ctx ...context.Context) (*rest.ScheduledWorkflowsList, error) {
	resp, err := s.api.WorkflowScheduledListWithResponse(
		getContext(ctx...),
		s.tenantId,
		&opts,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Get retrieves a specific scheduled workflow run by its ID.
func (s *schedulesClientImpl) Get(scheduledRunId string, ctx ...context.Context) (*rest.ScheduledWorkflows, error) {
	scheduledRunIdUUID, err := uuid.Parse(scheduledRunId)
	if err != nil {
		return nil, err
	}

	resp, err := s.api.WorkflowScheduledGetWithResponse(
		getContext(ctx...),
		s.tenantId,
		scheduledRunIdUUID,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
