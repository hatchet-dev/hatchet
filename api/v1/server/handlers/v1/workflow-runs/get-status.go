package workflowruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
)

func (t *V1WorkflowRunsService) V1WorkflowRunGetStatus(ctx echo.Context, request gen.V1WorkflowRunGetStatusRequestObject) (gen.V1WorkflowRunGetStatusResponseObject, error) {
	rawWorkflowRun := ctx.Get("v1-workflow-run").(*v1.V1WorkflowRunPopulator)

	if rawWorkflowRun == nil {
		return gen.V1WorkflowRunGetStatus404JSONResponse(apierrors.NewAPIErrors("could not find the workflow run provided")), nil
	}

	status := transformers.ToWorkflowRunStatus(*rawWorkflowRun)

	// Search for api errors to see how we handle errors in other cases
	return gen.V1WorkflowRunGetStatus200JSONResponse(
		status,
	), nil
}
