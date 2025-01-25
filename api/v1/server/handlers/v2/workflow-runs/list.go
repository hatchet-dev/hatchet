package v2workflowruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v2"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
)

func (t *V2WorkflowRunsService) V2WorkflowRunsList(ctx echo.Context, request gen.V2WorkflowRunsListRequestObject) (gen.V2WorkflowRunsListResponseObject, error) {
	// tenant := ctx.Get("tenant").(*db.TenantModel)

	workflow_runs, err := t.config.EngineRepository.OLAP().ReadTaskRuns(request.Tenant, *request.Params.Limit, *request.Params.Offset)

	if err != nil {
		return nil, err
	}

	workflowRunsPtr := make([]*olap.WorkflowRun, len(workflow_runs))
	for i := range workflow_runs {
		workflowRunsPtr[i] = &workflow_runs[i]
	}

	result := transformers.ToWorkflowRuns(workflowRunsPtr)

	// Search for api errors to see how we handle errors in other cases
	return gen.V2WorkflowRunsList200JSONResponse(
		result,
	), nil
}

func (t *V2WorkflowRunsService) V2WorkflowRunListStepRunEvents(ctx echo.Context, request gen.V2WorkflowRunListStepRunEventsRequestObject) (gen.V2WorkflowRunListStepRunEventsResponseObject, error) {
	taskRunEvents, err := t.config.EngineRepository.OLAP().ReadTaskRunEvents(request.Tenant, request.WorkflowRun, *request.Params.Limit, *request.Params.Offset)

	if err != nil {
		return nil, err
	}

	taskRunEventsPtr := make([]*olap.TaskRunEvent, len(taskRunEvents))

	for i := range taskRunEvents {
		taskRunEventsPtr[i] = &taskRunEvents[i]
	}

	result := transformers.ToTaskRunEvent(taskRunEventsPtr)

	// Search for api errors to see how we handle errors in other cases
	return gen.V2WorkflowRunListStepRunEvents200JSONResponse(
		result,
	), nil
}
