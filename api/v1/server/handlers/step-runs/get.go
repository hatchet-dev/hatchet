package stepruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

func (t *StepRunService) StepRunGet(ctx echo.Context, request gen.StepRunGetRequestObject) (gen.StepRunGetResponseObject, error) {
	stepRun := ctx.Get("step-run").(*dbsqlc.StepRun)

	return gen.StepRunGet200JSONResponse(
		*transformers.ToStepRunFull(stepRun),
	), nil
}
