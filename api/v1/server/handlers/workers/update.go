package workers

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkerService) WorkerUpdate(ctx echo.Context, request gen.WorkerUpdateRequestObject) (gen.WorkerUpdateResponseObject, error) {
	populator := populator.FromContext(ctx)

	worker, err := populator.GetWorker()
	if err != nil {
		return nil, err
	}

	// validate the request
	if apiErrors, err := t.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.WorkerUpdate400JSONResponse(*apiErrors), nil
	}

	update := repository.ApiUpdateWorkerOpts{}

	if request.Body.IsPaused != nil {
		update.IsPaused = request.Body.IsPaused
	}

	updatedWorker, err := t.config.APIRepository.Worker().UpdateWorker(
		sqlchelpers.UUIDToStr(worker.Worker.TenantId),
		sqlchelpers.UUIDToStr(worker.Worker.ID),
		update)

	if err != nil {
		return nil, err
	}

	return gen.WorkerUpdate200JSONResponse(*transformers.ToWorkerSqlc(updatedWorker, nil, nil, nil)), nil
}
