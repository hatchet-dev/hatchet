package features

import (
	"context"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// WorkersClient provides methods for interacting with workers
// in the Hatchet platform.
type WorkersClient interface {
	// Get retrieves a worker by its ID.
	Get(ctx context.Context, workerId string) (*rest.Worker, error)

	// List retrieves all workers for the tenant.
	List(ctx context.Context) (*rest.WorkerList, error)

	// IsPaused checks if a worker is paused.
	IsPaused(ctx context.Context, workerId string) (bool, error)

	// Pause pauses a worker.
	Pause(ctx context.Context, workerId string) (*rest.Worker, error)

	// Unpause unpauses a worker.
	Unpause(ctx context.Context, workerId string) (*rest.Worker, error)
}

// workersClientImpl implements the WorkersClient interface.
type workersClientImpl struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

// NewWorkersClient creates a new client for interacting with workers.
func NewWorkersClient(
	api *rest.ClientWithResponses,
	tenantId *string,
) WorkersClient {
	tenantIdUUID := uuid.MustParse(*tenantId)

	return &workersClientImpl{
		api:      api,
		tenantId: tenantIdUUID,
	}
}

// Get retrieves a worker by its ID.
func (w *workersClientImpl) Get(ctx context.Context, workerId string) (*rest.Worker, error) {
	workerIdUUID, err := uuid.Parse(workerId)
	if err != nil {
		return nil, err
	}

	resp, err := w.api.WorkerGetWithResponse(
		ctx,
		workerIdUUID,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// List retrieves all workers for the tenant.
func (w *workersClientImpl) List(ctx context.Context) (*rest.WorkerList, error) {
	resp, err := w.api.WorkerListWithResponse(
		ctx,
		w.tenantId,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// IsPaused checks if a worker is paused.
func (w *workersClientImpl) IsPaused(ctx context.Context, workerId string) (bool, error) {
	worker, err := w.Get(ctx, workerId)
	if err != nil {
		return false, err
	}

	status := worker.Status

	if status == nil {
		return false, nil
	}

	return *worker.Status == rest.WorkerStatus("PAUSED"), nil
}

// Pause pauses a worker.
func (w *workersClientImpl) Pause(ctx context.Context, workerId string) (*rest.Worker, error) {
	workerIdUUID, err := uuid.Parse(workerId)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return resp.JSON200, nil
}

// Unpause unpauses a worker.
func (w *workersClientImpl) Unpause(ctx context.Context, workerId string) (*rest.Worker, error) {
	workerIdUUID, err := uuid.Parse(workerId)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return resp.JSON200, nil
}
