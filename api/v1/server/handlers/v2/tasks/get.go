package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *TasksService) V2TaskGet(ctx echo.Context, request gen.V2TaskGetRequestObject) (gen.V2TaskGetResponseObject, error) {
	code := uint64(501)
	return gen.V2TaskGet501JSONResponse(gen.APIErrors{
		Errors: []gen.APIError{{
			Code:        &code,
			Description: "Not implemented",
		}},
	}), nil
}
