package features

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// RunsClient provides methods for interacting with workflow runs
// in the Hatchet platform.
type RunsClient interface {
	// Get retrieves a workflow run by its ID.
	Get(runId string, ctx ...context.Context) (*rest.V1WorkflowRunGetResponse, error)

	// GetDetails retrieves detailed information about a workflow run by its ID.
	GetDetails(runId string, ctx ...context.Context) (*rest.WorkflowRunGetShapeResponse, error)

	// List retrieves a collection of workflow runs based on the provided parameters.
	List(opts rest.V1WorkflowRunListParams, ctx ...context.Context) (*rest.V1WorkflowRunListResponse, error)

	// Replay requests a task to be replayed within a workflow run.
	Replay(opts rest.V1ReplayTaskRequest, ctx ...context.Context) (*rest.V1TaskReplayResponse, error)

	// Cancel requests cancellation of a specific task within a workflow run.
	Cancel(opts rest.V1CancelTaskRequest, ctx ...context.Context) (*rest.V1TaskCancelResponse, error)
}

// runsClientImpl implements the RunsClient interface.
type runsClientImpl struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

// NewRunsClient creates a new client for interacting with workflow runs.
func NewRunsClient(
	api *rest.ClientWithResponses,
	tenantId *string,
) RunsClient {
	tenantIdUUID := uuid.MustParse(*tenantId)

	return &runsClientImpl{
		api:      api,
		tenantId: tenantIdUUID,
	}
}

// Get retrieves a workflow run by its ID.
func (r *runsClientImpl) Get(runId string, ctx ...context.Context) (*rest.V1WorkflowRunGetResponse, error) {
	return r.api.V1WorkflowRunGetWithResponse(
		getContext(ctx...),
		uuid.MustParse(runId),
	)
}

// GetDetails retrieves detailed information about a workflow run by its ID.
func (r *runsClientImpl) GetDetails(runId string, ctx ...context.Context) (*rest.WorkflowRunGetShapeResponse, error) {
	return r.api.WorkflowRunGetShapeWithResponse(
		getContext(ctx...),
		r.tenantId,
		uuid.MustParse(runId),
	)
}

// List retrieves a collection of workflow runs based on the provided parameters.
func (r *runsClientImpl) List(opts rest.V1WorkflowRunListParams, ctx ...context.Context) (*rest.V1WorkflowRunListResponse, error) {
	return r.api.V1WorkflowRunListWithResponse(
		getContext(ctx...),
		r.tenantId,
		&opts,
	)
}

// Replay requests a task to be replayed within a workflow run.
func (r *runsClientImpl) Replay(opts rest.V1ReplayTaskRequest, ctx ...context.Context) (*rest.V1TaskReplayResponse, error) {
	json, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}

	return r.api.V1TaskReplayWithBodyWithResponse(
		getContext(ctx...),
		r.tenantId,
		"application/json",
		bytes.NewReader(json),
	)
}

// Cancel requests cancellation of a specific task within a workflow run.
func (r *runsClientImpl) Cancel(opts rest.V1CancelTaskRequest, ctx ...context.Context) (*rest.V1TaskCancelResponse, error) {
	json, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}

	return r.api.V1TaskCancelWithBodyWithResponse(
		getContext(ctx...),
		r.tenantId,
		"application/json",
		bytes.NewReader(json),
	)

}
