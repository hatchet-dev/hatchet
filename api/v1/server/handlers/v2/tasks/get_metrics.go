package tasks

import (
	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v2"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/labstack/echo/v4"
)

func (t *TasksService) V2TaskListStatusMetrics(ctx echo.Context, request gen.V2TaskListStatusMetricsRequestObject) (gen.V2TaskListStatusMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	var workflowIds []uuid.UUID

	if request.Params.WorkflowIds != nil {
		workflowIds = *request.Params.WorkflowIds
	}

	metrics, err := t.config.EngineRepository.OLAP().ReadTaskRunMetrics(ctx.Request().Context(), tenant.ID, repository.ReadTaskRunMetricsOpts{
		CreatedAfter: request.Params.Since,
		WorkflowIds:  workflowIds,
	})

	if err != nil {
		return nil, err
	}

	result := transformers.ToTaskRunMetrics(&metrics)

	// Search for api errors to see how we handle errors in other cases
	return gen.V2TaskListStatusMetrics200JSONResponse(
		result,
	), nil
}
