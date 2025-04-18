package workflowruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *V1WorkflowRunsService) V1WorkflowRunTaskEventsList(ctx echo.Context, request gen.V1WorkflowRunTaskEventsListRequestObject) (gen.V1WorkflowRunTaskEventsListResponseObject, error) {
	populator := populator.FromContext(ctx)

	tenant, err := populator.GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	rawWorkflowRun, err := populator.GetV1WorkflowRun()
	if err != nil {
		return nil, err
	}

	workflowRun := rawWorkflowRun.WorkflowRun

	taskRunEvents, err := t.config.V1.OLAP().ListTaskRunEventsByWorkflowRunId(
		ctx.Request().Context(),
		tenantId,
		workflowRun.ExternalID,
	)

	if err != nil {
		return nil, err
	}

	result := transformers.ToWorkflowRunTaskRunEventsMany(taskRunEvents)

	// Search for api errors to see how we handle errors in other cases
	return gen.V1WorkflowRunTaskEventsList200JSONResponse(
		result,
	), nil
}
