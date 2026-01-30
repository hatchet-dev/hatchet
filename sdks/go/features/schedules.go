package features

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/config/client"
)

// CreateScheduledRunTrigger contains the configuration for creating a scheduled run trigger.
type CreateScheduledRunTrigger struct {
	// TriggerAt specifies when the workflow should be triggered.
	TriggerAt time.Time `json:"triggerAt"`

	// Input is the optional input data for the workflow.
	Input map[string]interface{} `json:"input,omitempty"`

	// AdditionalMetadata is optional metadata to associate with the scheduled run.
	AdditionalMetadata map[string]interface{} `json:"additionalMetadata,omitempty"`

	Priority *RunPriority `json:"priority,omitempty"`
}

// SchedulesClient provides methods for interacting with workflow schedules
type SchedulesClient struct {
	api       *rest.ClientWithResponses
	tenantId  uuid.UUID
	namespace *string
}

// NewSchedulesClient creates a new SchedulesClient
func NewSchedulesClient(
	api *rest.ClientWithResponses,
	tenantId uuid.UUID,
	namespace *string,
) *SchedulesClient {
	tenantIdUUID := tenantId

	return &SchedulesClient{
		api:       api,
		tenantId:  tenantIdUUID,
		namespace: namespace,
	}
}

// Create creates a new scheduled workflow run.
func (s *SchedulesClient) Create(ctx context.Context, workflowName string, trigger CreateScheduledRunTrigger) (*rest.ScheduledWorkflows, error) {
	workflowName = client.ApplyNamespace(workflowName, s.namespace)

	var priority *int32

	if trigger.Priority != nil {
		priorityInt := int32(*trigger.Priority)
		priority = &priorityInt
	}

	request := rest.ScheduleWorkflowRunRequest{
		Input:              trigger.Input,
		AdditionalMetadata: trigger.AdditionalMetadata,
		TriggerAt:          trigger.TriggerAt,
		Priority:           priority,
	}

	resp, err := s.api.ScheduledWorkflowRunCreateWithResponse(
		ctx,
		s.tenantId,
		workflowName,
		request,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create scheduled workflow run")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Delete removes a scheduled workflow run.
func (s *SchedulesClient) Delete(ctx context.Context, scheduledRunId string) error {
	scheduledRunIdUUID, err := uuid.Parse(scheduledRunId)
	if err != nil {
		return errors.Wrap(err, "failed to parse scheduled run id")
	}

	resp, err := s.api.WorkflowScheduledDeleteWithResponse(
		ctx,
		s.tenantId,
		scheduledRunIdUUID,
	)
	if err != nil {
		return errors.Wrap(err, "failed to delete scheduled workflow run")
	}

	if err := validateStatusCodeResponse(resp.StatusCode(), resp.Body); err != nil {
		return err
	}

	return nil
}

// List retrieves a collection of scheduled workflow runs based on the provided parameters.
func (s *SchedulesClient) List(ctx context.Context, opts rest.WorkflowScheduledListParams) (*rest.ScheduledWorkflowsList, error) {
	resp, err := s.api.WorkflowScheduledListWithResponse(
		ctx,
		s.tenantId,
		&opts,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list scheduled workflow runs")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Get retrieves a specific scheduled workflow run by its ID.
func (s *SchedulesClient) Get(ctx context.Context, scheduledRunId string) (*rest.ScheduledWorkflows, error) {
	scheduledRunIdUUID, err := uuid.Parse(scheduledRunId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse scheduled run id")
	}

	resp, err := s.api.WorkflowScheduledGetWithResponse(
		ctx,
		s.tenantId,
		scheduledRunIdUUID,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get scheduled workflow run")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
