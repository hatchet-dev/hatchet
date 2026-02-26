package features

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// RunsClient provides methods for interacting with workflow runs
type RunsClient struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
	v0Client client.Client
	l        *zerolog.Logger
}

// NewRunsClient creates a new client for interacting with workflow runs.
func NewRunsClient(
	api *rest.ClientWithResponses,
	tenantId string,
	v0Client client.Client,
) *RunsClient {
	tenantIdUUID := uuid.MustParse(tenantId)
	logger := v0Client.Logger()

	return &RunsClient{
		api:      api,
		tenantId: tenantIdUUID,
		v0Client: v0Client,
		l:        logger,
	}
}

// Get retrieves a workflow run by its ID.
func (r *RunsClient) Get(ctx context.Context, runId string) (*rest.V1WorkflowRunDetails, error) {
	resp, err := r.api.V1WorkflowRunGetWithResponse(
		ctx,
		uuid.MustParse(runId),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workflow run")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// GetStatus retrieves the status of a workflow run by its ID.
func (r *RunsClient) GetStatus(ctx context.Context, runId string) (*rest.V1TaskStatus, error) {
	resp, err := r.api.V1WorkflowRunGetStatusWithResponse(
		ctx,
		uuid.MustParse(runId),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workflow run status")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// GetDetails retrieves detailed information about a workflow run via gRPC,
// including task-level output, errors, and status.
func (r *RunsClient) GetDetails(ctx context.Context, runId uuid.UUID) (*client.RunDetails, error) {
	resp, err := r.v0Client.Admin().GetRunDetails(ctx, runId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workflow run details")
	}

	return resp, nil
}

// List retrieves a collection of workflow runs based on the provided parameters.
func (r *RunsClient) List(ctx context.Context, opts rest.V1WorkflowRunListParams) (*rest.V1TaskSummaryList, error) {
	resp, err := r.api.V1WorkflowRunListWithResponse(
		ctx,
		r.tenantId,
		&opts,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list workflow runs")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Replay requests a task to be replayed within a workflow run.
func (r *RunsClient) Replay(ctx context.Context, opts rest.V1ReplayTaskRequest) (*rest.V1ReplayedTasks, error) {
	json, err := json.Marshal(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal rest.V1ReplayTaskRequest")
	}

	resp, err := r.api.V1TaskReplayWithBodyWithResponse(
		ctx,
		r.tenantId,
		"application/json; charset=utf-8",
		bytes.NewReader(json),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to replay task")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Cancel requests cancellation of a specific task within a workflow run.
func (r *RunsClient) Cancel(ctx context.Context, opts rest.V1CancelTaskRequest) (*rest.V1CancelledTasks, error) {
	json, err := json.Marshal(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed marshal rest.V1CancelTaskRequest")
	}

	resp, err := r.api.V1TaskCancelWithBodyWithResponse(
		ctx,
		r.tenantId,
		"application/json; charset=utf-8",
		bytes.NewReader(json),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to cancel task")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// SubscribeToStream subscribes to streaming events for a specific workflow run.
func (r *RunsClient) SubscribeToStream(ctx context.Context, workflowRunId string) <-chan string {
	ch := make(chan string)

	go func() {
		defer func() {
			close(ch)
			r.l.Info().Str("workflowRunId", workflowRunId).Msg("stream subscription ended")
		}()

		r.l.Info().Str("workflowRunId", workflowRunId).Msg("starting stream subscription")

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
		}
	}()

	return ch
}
