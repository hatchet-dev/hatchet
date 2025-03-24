package features

import (
	"context"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// WorkflowsClient provides methods for interacting with workflows
// in the Hatchet platform.
type WorkflowsClient interface {
	// Get retrieves a workflow by its ID or name.
	Get(workflowId string, ctx ...context.Context) (*rest.Workflow, error)

	// List retrieves all workflows for the tenant with optional filtering parameters.
	List(opts *rest.WorkflowListParams, ctx ...context.Context) (*rest.WorkflowList, error)

	// Delete removes a workflow by its ID or name.
	Delete(workflowId string, ctx ...context.Context) (*rest.WorkflowDeleteResponse, error)

	// IsPaused checks if a workflow is paused.
	IsPaused(workflowId string, ctx ...context.Context) (bool, error)

	// Pause pauses a workflow.
	Pause(workflowId string, ctx ...context.Context) (*rest.Workflow, error)

	// Unpause unpauses a workflow.
	Unpause(workflowId string, ctx ...context.Context) (*rest.Workflow, error)
}

// workflowsClientImpl implements the WorkflowsClient interface.
type workflowsClientImpl struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

// NewWorkflowsClient creates a new client for interacting with workflows.
func NewWorkflowsClient(
	api *rest.ClientWithResponses,
	tenantId *string,
) WorkflowsClient {
	tenantIdUUID := uuid.MustParse(*tenantId)

	return &workflowsClientImpl{
		api:      api,
		tenantId: tenantIdUUID,
	}
}

// Get retrieves a workflow by its ID or name.
func (w *workflowsClientImpl) Get(workflowId string, ctx ...context.Context) (*rest.Workflow, error) {
	id := uuid.MustParse(workflowId)

	resp, err := w.api.WorkflowGetWithResponse(
		getContext(ctx...),
		id,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// List retrieves all workflows for the tenant with optional filtering parameters.
func (w *workflowsClientImpl) List(opts *rest.WorkflowListParams, ctx ...context.Context) (*rest.WorkflowList, error) {
	resp, err := w.api.WorkflowListWithResponse(
		getContext(ctx...),
		w.tenantId,
		opts,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Delete removes a workflow by its ID or name.
func (w *workflowsClientImpl) Delete(workflowId string, ctx ...context.Context) (*rest.WorkflowDeleteResponse, error) {
	id := uuid.MustParse(workflowId)

	resp, err := w.api.WorkflowDeleteWithResponse(
		getContext(ctx...),
		id,
	)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// IsPaused checks if a workflow is paused.
func (w *workflowsClientImpl) IsPaused(workflowId string, ctx ...context.Context) (bool, error) {
	workflow, err := w.Get(workflowId, ctx...)
	if err != nil {
		return false, err
	}

	if workflow.IsPaused == nil {
		return false, nil
	}

	return *workflow.IsPaused, nil
}

// Pause pauses a workflow.
func (w *workflowsClientImpl) Pause(workflowId string, ctx ...context.Context) (*rest.Workflow, error) {

	// TODO by name

	id := uuid.MustParse(workflowId)

	paused := true

	request := rest.WorkflowUpdateJSONRequestBody{
		IsPaused: &paused,
	}

	resp, err := w.api.WorkflowUpdateWithResponse(
		getContext(ctx...),
		id,
		request,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Unpause unpauses a workflow.
func (w *workflowsClientImpl) Unpause(workflowId string, ctx ...context.Context) (*rest.Workflow, error) {

	id := uuid.MustParse(workflowId)

	paused := false

	request := rest.WorkflowUpdateJSONRequestBody{
		IsPaused: &paused,
	}

	resp, err := w.api.WorkflowUpdateWithResponse(
		getContext(ctx...),
		id,
		request,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
