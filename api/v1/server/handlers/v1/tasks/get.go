package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *TasksService) V1TaskGet(ctx echo.Context, request gen.V1TaskGetRequestObject) (gen.V1TaskGetResponseObject, error) {
	populator := populator.FromContext(ctx)

	task, err := populator.GetTask()
	if err != nil {
		return nil, err
	}

	taskWithData, workflowRunExternalId, err := t.config.V1.OLAP().ReadTaskRunData(ctx.Request().Context(), task.TenantID, task.ID, task.InsertedAt)

	if err != nil {
		return nil, err
	}

	result := transformers.ToTask(taskWithData, workflowRunExternalId)

	return gen.V1TaskGet200JSONResponse(
		result,
	), nil
}
