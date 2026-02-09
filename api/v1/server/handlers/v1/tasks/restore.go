package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TasksService) V1TaskRestore(ctx echo.Context, request gen.V1TaskRestoreRequestObject) (gen.V1TaskRestoreResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	resp, err := t.proxyRestore.Do(
		ctx.Request().Context(),
		tenant,
		&dispatchercontracts.RestoreEvictedTaskRequest{
			TaskRunExternalId: request.Task.String(),
		},
	)
	if err != nil {
		return nil, err
	}

	return gen.V1TaskRestore200JSONResponse{
		Requeued: resp.Requeued,
	}, nil
}
