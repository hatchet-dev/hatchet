package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (t *WorkflowService) WorkflowRunGetState(ctx echo.Context, request gen.WorkflowRunGetStateRequestObject) (gen.WorkflowRunGetStateResponseObject, error) {
	run := ctx.Get("workflow-run").(*dbsqlc.GetWorkflowRunByIdRow)

	jobs, err := t.config.APIRepository.JobRun().ListJobRunByWorkflowRunId(
		ctx.Request().Context(),
		sqlchelpers.UUIDToStr(run.TenantId),
		sqlchelpers.UUIDToStr(run.ID),
	)

	if err != nil {
		return nil, err
	}

	return gen.WorkflowRunGetState200JSONResponse(
		*transformers.ToWorkflowRunState(run, jobs, nil),
	), nil
}
