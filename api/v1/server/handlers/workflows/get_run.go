package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (t *WorkflowService) WorkflowRunGet(ctx echo.Context, request gen.WorkflowRunGetRequestObject) (gen.WorkflowRunGetResponseObject, error) {
	run := ctx.Get("workflow-run").(*db.WorkflowRunModel)

	resp, err := transformers.ToWorkflowRun(run)

	if err != nil {
		return nil, err
	}

	return gen.WorkflowRunGet200JSONResponse(
		*resp,
	), nil
}
