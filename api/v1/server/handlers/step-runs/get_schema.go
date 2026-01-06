package stepruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *StepRunService) StepRunGetSchema(ctx echo.Context, request gen.StepRunGetSchemaRequestObject) (gen.StepRunGetSchemaResponseObject, error) {
	return gen.StepRunGetSchema400JSONResponse(apierrors.NewAPIErrors(
		"StepRunGetSchema is deprecated",
	)), nil
}
