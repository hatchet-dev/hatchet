package stepruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *StepRunService) StepRunListArchives(ctx echo.Context, request gen.StepRunListArchivesRequestObject) (gen.StepRunListArchivesResponseObject, error) {
	return gen.StepRunListArchives400JSONResponse(apierrors.NewAPIErrors(
		"StepRunListArchives is deprecated",
	)), nil
}
