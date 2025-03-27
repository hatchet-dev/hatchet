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
	Get(ctx context.Context, runId string) (*rest.V1WorkflowRunGetResponse, error)

	// GetDetails retrieves detailed information about a workflow run by its ID.
	GetDetails(ctx context.Context, runId string) (*rest.WorkflowRunGetShapeResponse, error)

	// List retrieves a collection of workflow runs based on the provided parameters.
	List(ctx context.Context, opts rest.V1WorkflowRunListParams) (*rest.V1WorkflowRunListResponse, error)

	// Replay requests a task to be replayed within a workflow run.
	Replay(ctx context.Context, opts rest.V1ReplayTaskRequest) (*rest.V1TaskReplayResponse, error)

	// Cancel requests cancellation of a specific task within a workflow run.
	Cancel(ctx context.Context, opts rest.V1CancelTaskRequest) (*rest.V1TaskCancelResponse, error)
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
func (r *runsClientImpl) Get(ctx context.Context, runId string) (*rest.V1WorkflowRunGetResponse, error) {
	return r.api.V1WorkflowRunGetWithResponse(
		ctx,
		uuid.MustParse(runId),
	)
}

// GetDetails retrieves detailed information about a workflow run by its ID.
func (r *runsClientImpl) GetDetails(ctx context.Context, runId string) (*rest.WorkflowRunGetShapeResponse, error) {
	return r.api.WorkflowRunGetShapeWithResponse(
		ctx,
		r.tenantId,
		uuid.MustParse(runId),
	)
}

// List retrieves a collection of workflow runs based on the provided parameters.
func (r *runsClientImpl) List(ctx context.Context, opts rest.V1WorkflowRunListParams) (*rest.V1WorkflowRunListResponse, error) {
	return r.api.V1WorkflowRunListWithResponse(
		ctx,
		r.tenantId,
		&opts,
	)
}

// Replay requests a task to be replayed within a workflow run.
func (r *runsClientImpl) Replay(ctx context.Context, opts rest.V1ReplayTaskRequest) (*rest.V1TaskReplayResponse, error) {
	json, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}

	return r.api.V1TaskReplayWithBodyWithResponse(
		ctx,
		r.tenantId,
		"application/json",
		bytes.NewReader(json),
	)
}

// Cancel requests cancellation of a specific task within a workflow run.
func (r *runsClientImpl) Cancel(ctx context.Context, opts rest.V1CancelTaskRequest) (*rest.V1TaskCancelResponse, error) {
	json, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}

	return r.api.V1TaskCancelWithBodyWithResponse(
		ctx,
		r.tenantId,
		"application/json",
		bytes.NewReader(json),
	)
}
