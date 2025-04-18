package stepruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
)

func (t *StepRunService) StepRunGet(ctx echo.Context, request gen.StepRunGetRequestObject) (gen.StepRunGetResponseObject, error) {
	populator := populator.FromContext(ctx)

	stepRun, err := populator.GetStepRun()
	if err != nil {
		return nil, err
	}

	return gen.StepRunGet200JSONResponse(
		*transformers.ToStepRunFull(stepRun),
	), nil
}
