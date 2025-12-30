package workflowruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *WorkflowRunsService) WorkflowRunUpdateReplay(ctx echo.Context, request gen.WorkflowRunUpdateReplayRequestObject) (gen.WorkflowRunUpdateReplayResponseObject, error) {
	return gen.WorkflowRunUpdateReplay400JSONResponse(apierrors.NewAPIErrors(
		"WorkflowRunUpdateReplay is deprecated; please use V1TaskReplay instead",
	)), nil
}
