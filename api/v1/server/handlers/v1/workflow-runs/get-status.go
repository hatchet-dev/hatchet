package workflowruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/enums"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func (t *V1WorkflowRunsService) V1WorkflowRunGetStatus(ctx echo.Context, request gen.V1WorkflowRunGetStatusRequestObject) (gen.V1WorkflowRunGetStatusResponseObject, error) {
	maybeWorkflowRun := ctx.Get("v1-workflow-run").(*v1.V1WorkflowRunPopulator)

	status := maybeWorkflowRun.WorkflowRun.ReadableStatus
	if maybeWorkflowRun.WorkflowRun.ReadableStatus == enums.V1ReadableStatusOlapEVICTED {
		status = enums.V1ReadableStatusOlapRUNNING
	}

	return gen.V1WorkflowRunGetStatus200JSONResponse(status), nil
}
