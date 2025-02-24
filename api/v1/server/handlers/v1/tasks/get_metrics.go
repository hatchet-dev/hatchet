package tasks

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/labstack/echo/v4"
)

func (t *TasksService) V1TaskListStatusMetrics(ctx echo.Context, request gen.V1TaskListStatusMetricsRequestObject) (gen.V1TaskListStatusMetricsResponseObject, error) {
	code := uint64(501)
	return gen.V1TaskListStatusMetrics501JSONResponse(gen.APIErrors{
		Errors: []gen.APIError{{
			Code:        &code,
			Description: "Not implemented",
		}},
	}), nil
}
