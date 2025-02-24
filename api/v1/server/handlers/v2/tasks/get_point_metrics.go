package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *TasksService) V2TaskGetPointMetrics(ctx echo.Context, request gen.V2TaskGetPointMetricsRequestObject) (gen.V2TaskGetPointMetricsResponseObject, error) {
	code := uint64(501)
	return gen.V2TaskGetPointMetrics501JSONResponse(gen.APIErrors{
		Errors: []gen.APIError{{
			Code:        &code,
			Description: "Not implemented",
		}},
	}), nil
}
