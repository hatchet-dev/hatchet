package stepruns

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *StepRunService) StepRunGet(ctx echo.Context, request gen.StepRunGetRequestObject) (gen.StepRunGetResponseObject, error) {
	stepRun := ctx.Get("step-run").(*db.StepRunModel)

	res, err := transformers.ToStepRun(stepRun)

	if err != nil {
		return nil, fmt.Errorf("could not transform step run: %w", err)
	}

	return gen.StepRunGet200JSONResponse(
		*res,
	), nil
}
