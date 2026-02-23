package workflowruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func (t *V1WorkflowRunsService) V1WorkflowRunGetStatus(ctx echo.Context, request gen.V1WorkflowRunGetStatusRequestObject) (gen.V1WorkflowRunGetStatusResponseObject, error) {
	maybeWorkflowRun := ctx.Get("v1-workflow-run").(*v1.V1WorkflowRunPopulator)

	// TODO-DURABLE: Derive status from runtime eviction state so this endpoint can
	// return EVICTED instead of only OLAP readable statuses.
	return gen.V1WorkflowRunGetStatus200JSONResponse(
		gen.V1TaskStatus(maybeWorkflowRun.WorkflowRun.ReadableStatus),
	), nil
}
