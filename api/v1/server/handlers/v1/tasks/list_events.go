package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *TasksService) V1TaskEventList(ctx echo.Context, request gen.V1TaskEventListRequestObject) (gen.V1TaskEventListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	task := ctx.Get("task").(*sqlcv1.V1TasksOlap)

	taskRunEvents, err := t.config.V1.OLAP().ListTaskRunEvents(ctx.Request().Context(), tenantId, task.ID, task.InsertedAt, *request.Params.Limit, *request.Params.Offset)

	if err != nil {
		return nil, err
	}

	result := transformers.ToTaskRunEventMany(taskRunEvents, sqlchelpers.UUIDToStr(task.ExternalID))

	return gen.V1TaskEventList200JSONResponse(
		result,
	), nil
}
