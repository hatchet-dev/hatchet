package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *TasksService) V1TaskList(ctx echo.Context, request gen.V1TaskListRequestObject) (gen.V1TaskListResponseObject, error) {
	code := uint64(501)
	return gen.V1TaskList501JSONResponse(gen.APIErrors{
		Errors: []gen.APIError{{
			Code:        &code,
			Description: "Not implemented",
		}},
	}), nil
}
