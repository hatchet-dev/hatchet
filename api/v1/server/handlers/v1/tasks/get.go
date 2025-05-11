package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *TasksService) V1TaskGet(ctx echo.Context, request gen.V1TaskGetRequestObject) (gen.V1TaskGetResponseObject, error) {
	task := ctx.Get("task").(*sqlcv1.V1TasksOlap)

	attempt := request.Params.Attempt

	var retryCount int

	if attempt != nil {
		retryCount = *attempt - 1
	}

	if retryCount < 0 {
		return nil, echo.NewHTTPError(400, "Attempt must be greater than 0")
	}

	taskWithData, workflowRunExternalId, err := t.config.V1.OLAP().ReadTaskRunData(
		ctx.Request().Context(),
		task.TenantID,
		task.ID,
		task.InsertedAt,
		&retryCount,
	)

	if err != nil {
		return nil, err
	}

	result := transformers.ToTask(taskWithData, workflowRunExternalId)

	return gen.V1TaskGet200JSONResponse(
		result,
	), nil
}
