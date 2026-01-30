package features

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// WorkersClient provides methods for interacting with workers
type WorkersClient struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

// NewWorkersClient creates a new WorkersClient
func NewWorkersClient(
	api *rest.ClientWithResponses,
	tenantId uuid.UUID,
) *WorkersClient {
	tenantIdUUID := tenantId

	return &WorkersClient{
		api:      api,
		tenantId: tenantIdUUID,
	}
}

// Get retrieves a worker by its ID.
func (w *WorkersClient) Get(ctx context.Context, workerId string) (*rest.Worker, error) {
	workerIdUUID, err := uuid.Parse(workerId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse worker ID")
	}

	resp, err := w.api.WorkerGetWithResponse(
		ctx,
		workerIdUUID,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get worker")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// List retrieves all workers for the tenant.
func (w *WorkersClient) List(ctx context.Context) (*rest.WorkerList, error) {
	resp, err := w.api.WorkerListWithResponse(
		ctx,
		w.tenantId,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list workers")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Pause pauses a worker.
func (w *WorkersClient) Pause(ctx context.Context, workerId string) (*rest.Worker, error) {
	workerIdUUID, err := uuid.Parse(workerId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse worker ID")
	}

	paused := true

	request := rest.WorkerUpdateJSONRequestBody{
		IsPaused: &paused,
	}

	resp, err := w.api.WorkerUpdateWithResponse(
		ctx,
		workerIdUUID,
		request,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to pause worker")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Unpause unpauses a worker.
func (w *WorkersClient) Unpause(ctx context.Context, workerId string) (*rest.Worker, error) {
	workerIdUUID, err := uuid.Parse(workerId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse worker ID")
	}

	paused := false

	request := rest.WorkerUpdateJSONRequestBody{
		IsPaused: &paused,
	}

	resp, err := w.api.WorkerUpdateWithResponse(
		ctx,
		workerIdUUID,
		request,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unpause worker")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
