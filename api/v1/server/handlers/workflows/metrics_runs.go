package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (t *WorkflowService) WorkflowRunGetMetrics(ctx echo.Context, request gen.WorkflowRunGetMetricsRequestObject) (gen.WorkflowRunGetMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	listOpts := &repository.WorkflowRunsMetricsOpts{}

	if request.Params.WorkflowId != nil {
		workflowIdStr := request.Params.WorkflowId.String()
		listOpts.WorkflowId = &workflowIdStr
	}

	if request.Params.EventId != nil {
		eventIdStr := request.Params.EventId.String()
		listOpts.EventId = &eventIdStr
	}

	if request.Params.ParentWorkflowRunId != nil {
		parentWorkflowRunIdStr := request.Params.ParentWorkflowRunId.String()
		listOpts.ParentId = &parentWorkflowRunIdStr
	}

	if request.Params.ParentStepRunId != nil {
		parentStepRunIdStr := request.Params.ParentStepRunId.String()
		listOpts.ParentStepRunId = &parentStepRunIdStr
	}

	workflowRunsMetricsCount, err := t.config.APIRepository.WorkflowRun().WorkflowRunMetricsCount(tenant.ID, listOpts)

	if err != nil {
		return nil, err
	}

	failed := int(workflowRunsMetricsCount.FAILED)
	pending := int(workflowRunsMetricsCount.PENDING)
	queued := int(workflowRunsMetricsCount.QUEUED)
	running := int(workflowRunsMetricsCount.RUNNING)
	succeeded := int(workflowRunsMetricsCount.SUCCEEDED)

	return gen.WorkflowRunGetMetrics200JSONResponse(
		gen.WorkflowRunsMetrics{
			Counts: &gen.WorkflowRunsMetricsCounts{
				FAILED:    &failed,
				PENDING:   &pending,
				QUEUED:    &queued,
				RUNNING:   &running,
				SUCCEEDED: &succeeded,
			},
		},
	), nil
}
