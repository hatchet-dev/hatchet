package workflowruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v2"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *V2WorkflowRunsService) V2WorkflowRunTaskEventsList(ctx echo.Context, request gen.V2WorkflowRunTaskEventsListRequestObject) (gen.V2WorkflowRunTaskEventsListResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	workflowRunId := request.WorkflowRunId

	taskRunEvents, err := t.config.EngineRepository.OLAP().ListTaskRunEventsByWorkflowRunId(
		ctx.Request().Context(),
		tenant.ID,
		workflowRunId,
	)

	if err != nil {
		return nil, err
	}

	result := transformers.ToTaskRunEventMany(taskRunEvents, workflowRunId.String())

	// Search for api errors to see how we handle errors in other cases
	return gen.V2WorkflowRunTaskEventsList200JSONResponse(
		result,
	), nil
}
