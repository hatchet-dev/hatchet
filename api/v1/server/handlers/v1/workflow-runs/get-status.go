package workflowruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
)

func (t *V1WorkflowRunsService) V1WorkflowRunGetStatus(ctx echo.Context, request gen.V1WorkflowRunGetStatusRequestObject) (gen.V1WorkflowRunGetStatusResponseObject, error) {
	maybeWorkflowRun := ctx.Get("v1-workflow-run").(*v1.V1WorkflowRunPopulator)

	return gen.V1WorkflowRunGetStatus200JSONResponse(
		gen.V1TaskStatus(maybeWorkflowRun.WorkflowRun.ReadableStatus),
	), nil
}
