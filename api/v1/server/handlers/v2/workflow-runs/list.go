package v2workflowruns

import (
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v2"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
)

func (t *V2WorkflowRunsService) V2WorkflowRunGet(ctx echo.Context, request gen.V2WorkflowRunGetRequestObject) (gen.V2WorkflowRunGetResponseObject, error) {

	workflow_run, err := t.config.EngineRepository.OLAP().ReadTaskRun(request.Tenant, request.WorkflowRun)

	if err != nil {
		return nil, err
	}

	result := transformers.ToWorkflowRun(&workflow_run)

	// Search for api errors to see how we handle errors in other cases
	return gen.V2WorkflowRunGet200JSONResponse(
		result,
	), nil
}

func (t *V2WorkflowRunsService) V2WorkflowRunsList(ctx echo.Context, request gen.V2WorkflowRunsListRequestObject) (gen.V2WorkflowRunsListResponseObject, error) {
	var (
		statuses = []gen.V2TaskStatus{
			gen.V2TaskStatusCANCELLED,
			gen.V2TaskStatusCOMPLETED,
			gen.V2TaskStatusFAILED,
			gen.V2TaskStatusQUEUED,
			gen.V2TaskStatusRUNNING,
		}
		since             = time.Now().Add(-24 * time.Hour)
		workflowIds       = []uuid.UUID{}
		limit       int64 = 50
		offset      int64 = 0
	)

	if request.Params.Statuses != nil {
		if len(*request.Params.Statuses) > 0 {
			statuses = *request.Params.Statuses
		}
	}

	if request.Params.Since != nil {
		since = *request.Params.Since
	}

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	if request.Params.WorkflowIds != nil {
		workflowIds = *request.Params.WorkflowIds
	}

	workflow_runs, total, err := t.config.EngineRepository.OLAP().ReadTaskRuns(
		request.Tenant,
		since,
		statuses,
		workflowIds,
		limit,
		offset,
	)

	if err != nil {
		return nil, err
	}

	workflowRunsPtr := make([]*olap.WorkflowRun, len(workflow_runs))
	for i := range workflow_runs {
		workflowRunsPtr[i] = &workflow_runs[i]
	}

	result := transformers.ToWorkflowRuns(workflowRunsPtr, total, limit, offset)

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

func (t *V2WorkflowRunsService) TaskRunGetMetrics(ctx echo.Context, request gen.TaskRunGetMetricsRequestObject) (gen.TaskRunGetMetricsResponseObject, error) {
	metrics, err := t.config.EngineRepository.OLAP().ReadTaskRunMetrics(request.Tenant, *request.Params.Since)

	if err != nil {
		return nil, err
	}

	result := transformers.ToTaskRunMetrics(&metrics)

	// Search for api errors to see how we handle errors in other cases
	return gen.TaskRunGetMetrics200JSONResponse(
		result,
	), nil
}
