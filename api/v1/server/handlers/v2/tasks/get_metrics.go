package tasks

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v2"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/labstack/echo/v4"
)

func (t *TasksService) V2TaskListStatusMetrics(ctx echo.Context, request gen.V2TaskListStatusMetricsRequestObject) (gen.V2TaskListStatusMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	metrics, err := t.config.EngineRepository.OLAP().ReadTaskRunMetrics(tenant.ID, *request.Params.Since)

	if err != nil {
		return nil, err
	}

	result := transformers.ToTaskRunMetrics(&metrics)

	// Search for api errors to see how we handle errors in other cases
	return gen.V2TaskListStatusMetrics200JSONResponse(
		result,
	), nil
}
