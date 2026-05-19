package stepruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *StepRunService) WorkflowRunListStepRunEvents(ctx echo.Context, request gen.WorkflowRunListStepRunEventsRequestObject) (gen.WorkflowRunListStepRunEventsResponseObject, error) {
	return gen.WorkflowRunListStepRunEvents400JSONResponse(apierrors.NewAPIErrors(
		"WorkflowRunListStepRunEvents is deprecated",
	)), nil
}
