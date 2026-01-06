package stepruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *StepRunService) StepRunGet(ctx echo.Context, request gen.StepRunGetRequestObject) (gen.StepRunGetResponseObject, error) {
	return gen.StepRunGet400JSONResponse(apierrors.NewAPIErrors(
		"StepRunGet is deprecated",
	)), nil
}
