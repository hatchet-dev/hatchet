package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (t *WorkflowService) WorkflowRunMetrics(ctx echo.Context, request gen.WorkflowRunMetricsRequestObject) (gen.WorkflowRunMetricsResponseObject, error) {
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
	running := int(workflowRunsMetricsCount.RUNNING)
	succeeded := int(workflowRunsMetricsCount.SUCCEEDED)

	return gen.WorkflowRunMetrics200JSONResponse(
		gen.WorkflowRunsMetrics{
			Counts: &gen.WorkflowRunsMetricsCounts{
				FAILED:    &failed,
				PENDING:   &pending,
				RUNNING:   &running,
				SUCCEEDED: &succeeded,
			},
		},
	), nil
}
