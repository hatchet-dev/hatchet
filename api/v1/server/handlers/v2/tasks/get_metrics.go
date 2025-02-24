package tasks

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/labstack/echo/v4"
)

func (t *TasksService) V2TaskListStatusMetrics(ctx echo.Context, request gen.V2TaskListStatusMetricsRequestObject) (gen.V2TaskListStatusMetricsResponseObject, error) {
	code := uint64(501)
	return gen.V2TaskListStatusMetrics501JSONResponse(gen.APIErrors{
		Errors: []gen.APIError{{
			Code:        &code,
			Description: "Not implemented",
		}},
	}), nil
}
