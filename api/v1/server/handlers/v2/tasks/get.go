package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v2"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/timescalev2"
)

func (t *TasksService) V2TaskGet(ctx echo.Context, request gen.V2TaskGetRequestObject) (gen.V2TaskGetResponseObject, error) {
	task := ctx.Get("task").(*timescalev2.V2TasksOlap)

	result := transformers.ToTask(task)

	return gen.V2TaskGet200JSONResponse(
		result,
	), nil
}
