package features

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/config/client"
)

// SchedulesClient provides methods for interacting with workflow schedules
// in the Hatchet platform.
type SchedulesClient interface {
	// Create creates a new scheduled workflow run.
	Create(ctx context.Context, workflowName string, trigger CreateScheduledRunTrigger) (*rest.ScheduledWorkflows, error)

	// Delete removes a scheduled workflow run.
	Delete(ctx context.Context, scheduledRunId string) error

	// List retrieves a collection of scheduled workflow runs based on the provided parameters.
	List(ctx context.Context, opts rest.WorkflowScheduledListParams) (*rest.ScheduledWorkflowsList, error)

	// Get retrieves a specific scheduled workflow run by its ID.
	Get(ctx context.Context, scheduledRunId string) (*rest.ScheduledWorkflows, error)
}

// CreateScheduledRunTrigger contains the configuration for creating a scheduled run trigger.
type CreateScheduledRunTrigger struct {
	// TriggerAt specifies when the workflow should be triggered.
	TriggerAt time.Time `json:"triggerAt"`

	// Input is the optional input data for the workflow.
	Input map[string]interface{} `json:"input,omitempty"`

	// AdditionalMetadata is optional metadata to associate with the scheduled run.
	AdditionalMetadata map[string]interface{} `json:"additionalMetadata,omitempty"`

	Priority *int32 `json:"priority,omitempty"`
}

// schedulesClientImpl implements the SchedulesClient interface.
type schedulesClientImpl struct {
	api       *rest.ClientWithResponses
	tenantId  uuid.UUID
	namespace *string
}

// NewSchedulesClient creates a new client for interacting with workflow schedules.
func NewSchedulesClient(
	api *rest.ClientWithResponses,
	tenantId *string,
	namespace *string,
) SchedulesClient {
	tenantIdUUID := uuid.MustParse(*tenantId)

	return &schedulesClientImpl{
		api:       api,
		tenantId:  tenantIdUUID,
		namespace: namespace,
	}
}

// Create creates a new scheduled workflow run.
func (s *schedulesClientImpl) Create(ctx context.Context, workflowName string, trigger CreateScheduledRunTrigger) (*rest.ScheduledWorkflows, error) {
	workflowName = client.ApplyNamespace(workflowName, s.namespace)

	request := rest.ScheduleWorkflowRunRequest{
		Input:              trigger.Input,
		AdditionalMetadata: trigger.AdditionalMetadata,
		TriggerAt:          trigger.TriggerAt,
		Priority:           trigger.Priority,
	}

	resp, err := s.api.ScheduledWorkflowRunCreateWithResponse(
		ctx,
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
func (s *schedulesClientImpl) Delete(ctx context.Context, scheduledRunId string) error {
	scheduledRunIdUUID, err := uuid.Parse(scheduledRunId)
	if err != nil {
		return err
	}

	_, err = s.api.WorkflowScheduledDeleteWithResponse(
		ctx,
		s.tenantId,
		scheduledRunIdUUID,
	)
	return err
}

// List retrieves a collection of scheduled workflow runs based on the provided parameters.
func (s *schedulesClientImpl) List(ctx context.Context, opts rest.WorkflowScheduledListParams) (*rest.ScheduledWorkflowsList, error) {
	resp, err := s.api.WorkflowScheduledListWithResponse(
		ctx,
		s.tenantId,
		&opts,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Get retrieves a specific scheduled workflow run by its ID.
func (s *schedulesClientImpl) Get(ctx context.Context, scheduledRunId string) (*rest.ScheduledWorkflows, error) {
	scheduledRunIdUUID, err := uuid.Parse(scheduledRunId)
	if err != nil {
		return nil, err
	}

	resp, err := s.api.WorkflowScheduledGetWithResponse(
		ctx,
		s.tenantId,
		scheduledRunIdUUID,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
