// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package features

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// Deprecated: RunsClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
// RunsClient provides methods for interacting with workflow runs
// in the Hatchet platform.
type RunsClient interface {
	// Get retrieves a workflow run by its ID.
	Get(ctx context.Context, runId string) (*rest.V1WorkflowRunGetResponse, error)

	// Get the status of a workflow run by its ID.
	GetStatus(ctx context.Context, runId string) (*rest.V1WorkflowRunGetStatusResponse, error)

	// GetDetails retrieves detailed information about a workflow run by its ID.
	// Deprecated: Use Get instead.
	GetDetails(ctx context.Context, runId string) (*rest.V1WorkflowRunGetResponse, error)

	// List retrieves a collection of workflow runs based on the provided parameters.
	List(ctx context.Context, opts rest.V1WorkflowRunListParams) (*rest.V1WorkflowRunListResponse, error)

	// Replay requests a task to be replayed within a workflow run.
	Replay(ctx context.Context, opts rest.V1ReplayTaskRequest) (*rest.V1TaskReplayResponse, error)

	// Cancel requests cancellation of a specific task within a workflow run.
	Cancel(ctx context.Context, opts rest.V1CancelTaskRequest) (*rest.V1TaskCancelResponse, error)

	// SubscribeToStream subscribes to streaming events for a specific workflow run.
	SubscribeToStream(ctx context.Context, workflowRunId string) (<-chan string, error)
}

// runsClientImpl implements the RunsClient interface.
type runsClientImpl struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
	v0Client client.Client
	l        *zerolog.Logger
}

// Deprecated: NewRunsClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
// NewRunsClient creates a new client for interacting with workflow runs.
func NewRunsClient(
	api *rest.ClientWithResponses,
	tenantId *string,
	v0Client client.Client,
) RunsClient {
	tenantIdUUID := uuid.MustParse(*tenantId)
	logger := v0Client.Logger()

	return &runsClientImpl{
		api:      api,
		tenantId: tenantIdUUID,
		v0Client: v0Client,
		l:        logger,
	}
}

// Get retrieves a workflow run by its ID.
func (r *runsClientImpl) Get(ctx context.Context, runId string) (*rest.V1WorkflowRunGetResponse, error) {
	return r.api.V1WorkflowRunGetWithResponse(
		ctx,
		uuid.MustParse(runId),
	)
}

// GetStatus retrieves the status of a workflow run by its ID.
func (r *runsClientImpl) GetStatus(ctx context.Context, runId string) (*rest.V1WorkflowRunGetStatusResponse, error) {
	return r.api.V1WorkflowRunGetStatusWithResponse(
		ctx,
		uuid.MustParse(runId),
	)
}

// GetDetails retrieves detailed information about a workflow run by its ID.
// Deprecated: Use Get instead.
func (r *runsClientImpl) GetDetails(ctx context.Context, runId string) (*rest.V1WorkflowRunGetResponse, error) {
	return r.api.V1WorkflowRunGetWithResponse(
		ctx,
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

// SubscribeToStream subscribes to streaming events for a specific workflow run.
func (r *runsClientImpl) SubscribeToStream(ctx context.Context, workflowRunId string) (<-chan string, error) {
	ch := make(chan string)

	go func() {
		defer func() {
			close(ch)
			r.l.Debug().Str("workflowRunId", workflowRunId).Msg("stream subscription ended")
		}()

		r.l.Debug().Str("workflowRunId", workflowRunId).Msg("starting stream subscription")

		err := r.v0Client.Subscribe().Stream(ctx, workflowRunId, func(event client.StreamEvent) error {
			select {
			case ch <- string(event.Message):
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
		if err != nil {
			r.l.Error().Err(err).Str("workflowRunId", workflowRunId).Msg("failed to subscribe to stream")
			return
		}
	}()

	return ch, nil
}
