package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *TasksService) V1TaskEventList(ctx echo.Context, request gen.V1TaskEventListRequestObject) (gen.V1TaskEventListResponseObject, error) {
	populator := populator.FromContext(ctx)

	tenant, err := populator.GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	task, err := populator.GetTask()
	if err != nil {
		return nil, err
	}

	taskRunEvents, err := t.config.V1.OLAP().ListTaskRunEvents(ctx.Request().Context(), tenantId, task.ID, task.InsertedAt, *request.Params.Limit, *request.Params.Offset)

	if err != nil {
		return nil, err
	}

	result := transformers.ToTaskRunEventMany(taskRunEvents, sqlchelpers.UUIDToStr(task.ExternalID))

	return gen.V1TaskEventList200JSONResponse(
		result,
	), nil
}
