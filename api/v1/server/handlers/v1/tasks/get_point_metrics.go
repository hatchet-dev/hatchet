package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *TasksService) V1TaskGetPointMetrics(ctx echo.Context, request gen.V1TaskGetPointMetricsRequestObject) (gen.V1TaskGetPointMetricsResponseObject, error) {
	code := uint64(501)
	return gen.V1TaskGetPointMetrics501JSONResponse(gen.APIErrors{
		Errors: []gen.APIError{{
			Code:        &code,
			Description: "Not implemented",
		}},
	}), nil
}
