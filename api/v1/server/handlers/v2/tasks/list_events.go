package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v2"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/timescalev2"
)

func (t *TasksService) V2TaskEventList(ctx echo.Context, request gen.V2TaskEventListRequestObject) (gen.V2TaskEventListResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	task := ctx.Get("task").(*timescalev2.V2TasksOlap)

	taskRunEvents, err := t.config.EngineRepository.OLAP().ListTaskRunEvents(tenant.ID, task.ID, task.InsertedAt, *request.Params.Limit, *request.Params.Offset)

	if err != nil {
		return nil, err
	}

	result := transformers.ToTaskRunEventMany(taskRunEvents, sqlchelpers.UUIDToStr(task.ExternalID))

	return gen.V2TaskEventList200JSONResponse(
		result,
	), nil
}
