package stepruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *StepRunService) StepRunListEvents(ctx echo.Context, request gen.StepRunListEventsRequestObject) (gen.StepRunListEventsResponseObject, error) {
	return gen.StepRunListEvents400JSONResponse(apierrors.NewAPIErrors(
		"StepRunListEvents is deprecated",
	)), nil
}
