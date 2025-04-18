package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkflowService) WorkflowRunGet(ctx echo.Context, request gen.WorkflowRunGetRequestObject) (gen.WorkflowRunGetResponseObject, error) {
	populator := populator.FromContext(ctx)

	run, err := populator.GetWorkflowRun()
	if err != nil {
		return nil, err
	}

	jobs, err := t.config.APIRepository.JobRun().ListJobRunByWorkflowRunId(
		ctx.Request().Context(),
		sqlchelpers.UUIDToStr(run.TenantId),
		sqlchelpers.UUIDToStr(run.ID),
	)

	if err != nil {
		return nil, err
	}
	jobIds := make([]string, len(jobs))

	for i, job := range jobs {
		jobIds[i] = sqlchelpers.UUIDToStr(job.ID)
	}

	steps, err := t.config.APIRepository.WorkflowRun().GetStepsForJobs(
		ctx.Request().Context(),
		sqlchelpers.UUIDToStr(run.TenantId),
		jobIds)
	if err != nil {
		return nil, err
	}

	stepRuns, err := t.config.APIRepository.WorkflowRun().GetStepRunsForJobRuns(
		ctx.Request().Context(),
		sqlchelpers.UUIDToStr(run.TenantId),
		jobIds)

	if err != nil {
		return nil, err
	}

	resp, err := transformers.ToWorkflowRun(run, jobs, steps, stepRuns)

	if err != nil {
		return nil, err
	}

	return gen.WorkflowRunGet200JSONResponse(
		*resp,
	), nil
}
