package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *TasksService) V1TaskEventList(ctx echo.Context, request gen.V1TaskEventListRequestObject) (gen.V1TaskEventListResponseObject, error) {
	code := uint64(501)
	return gen.V1TaskEventList501JSONResponse(gen.APIErrors{
		Errors: []gen.APIError{{
			Code:        &code,
			Description: "Not implemented",
		}},
	}), nil
}
