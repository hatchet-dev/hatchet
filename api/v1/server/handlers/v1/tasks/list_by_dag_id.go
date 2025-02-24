package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *TasksService) V1DagListTasks(ctx echo.Context, request gen.V1DagListTasksRequestObject) (gen.V1DagListTasksResponseObject, error) {
	code := uint64(501)
	return gen.V1DagListTasks501JSONResponse(gen.APIErrors{
		Errors: []gen.APIError{{
			Code:        &code,
			Description: "Not implemented",
		}},
	}), nil
}
