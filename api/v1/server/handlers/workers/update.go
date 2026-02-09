package workers

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkerService) WorkerUpdate(ctx echo.Context, request gen.WorkerUpdateRequestObject) (gen.WorkerUpdateResponseObject, error) {
	worker := ctx.Get("worker").(*sqlcv1.GetWorkerByIdRow)

	// validate the request
	if apiErrors, err := t.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.WorkerUpdate400JSONResponse(*apiErrors), nil
	}

	update := &v1.UpdateWorkerOpts{}

	if request.Body.IsPaused != nil {
		update.IsPaused = request.Body.IsPaused
	}

	updatedWorker, err := t.config.V1.Workers().UpdateWorker(
		ctx.Request().Context(),
		worker.Worker.TenantId,
		worker.Worker.ID,
		update,
	)

	if err != nil {
		return nil, err
	}

	return gen.WorkerUpdate200JSONResponse(*transformers.ToWorkerSqlc(updatedWorker, nil, nil, nil)), nil
}
