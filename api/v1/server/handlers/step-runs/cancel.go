package stepruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *StepRunService) StepRunUpdateCancel(ctx echo.Context, request gen.StepRunUpdateCancelRequestObject) (gen.StepRunUpdateCancelResponseObject, error) {
	return gen.StepRunUpdateCancel400JSONResponse(apierrors.NewAPIErrors(
		"StepRunUpdateCancel is deprecated; please use V1TaskCancel instead",
	)), nil
}
