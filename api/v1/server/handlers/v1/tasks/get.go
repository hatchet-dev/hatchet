package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *TasksService) V1TaskGet(ctx echo.Context, request gen.V1TaskGetRequestObject) (gen.V1TaskGetResponseObject, error) {
	task := ctx.Get("task").(*sqlcv1.V1TasksOlap)

	taskWithData, workflowRunExternalId, err := t.config.V1.OLAP().ReadTaskRunData(ctx.Request().Context(), task.TenantID, task.ID, task.InsertedAt)

	if err != nil {
		return nil, err
	}

	result := transformers.ToTask(taskWithData, workflowRunExternalId)

	return gen.V1TaskGet200JSONResponse(
		result,
	), nil
}
