package workflows

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (t *WorkflowService) WorkflowRunGetShape(ctx echo.Context, request gen.WorkflowRunGetShapeRequestObject) (gen.WorkflowRunGetShapeResponseObject, error) {
	run := ctx.Get("workflow-run").(*dbsqlc.GetWorkflowRunByIdRow)

	reqCtx, cancel := context.WithTimeout(ctx.Request().Context(), 5*time.Second)
	defer cancel()

	jobs, err := t.config.APIRepository.JobRun().ListJobRunByWorkflowRunId(
		reqCtx,
		sqlchelpers.UUIDToStr(run.TenantId),
		sqlchelpers.UUIDToStr(run.ID),
	)

	if err != nil {
		return nil, err
	}

	workflowVersion, err := t.config.APIRepository.Workflow().GetWorkflowVersionById(
		sqlchelpers.UUIDToStr(run.TenantId),
		sqlchelpers.UUIDToStr(run.WorkflowVersionId),
	)

	if err != nil {
		return nil, err
	}

	jobIds := make([]string, len(jobs))

	for i := range jobs {
		jobIds[i] = sqlchelpers.UUIDToStr(jobs[i].JobId)
	}

	steps, err := t.config.APIRepository.WorkflowRun().GetStepsForJobs(
		reqCtx,
		sqlchelpers.UUIDToStr(run.TenantId),
		jobIds,
	)

	if err != nil {
		return nil, err
	}

	return gen.WorkflowRunGetShape200JSONResponse(
		*transformers.ToWorkflowRunShape(
			run,
			workflowVersion,
			jobs,
			steps,
		),
	), nil
}
