package workflows

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/integrations/vcs/vcsutils"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (t *WorkflowService) StepRunGetDiff(ctx echo.Context, request gen.StepRunGetDiffRequestObject) (gen.StepRunGetDiffResponseObject, error) {
	stepRun := ctx.Get("step-run").(*db.StepRunModel)

	diffs, originalValues, err := vcsutils.GetStepRunOverrideDiffs(t.config.Repository.StepRun(), stepRun)

	if err != nil {
		return nil, fmt.Errorf("could not get diffs: %s", err)
	}

	resp := make([]gen.StepRunDiff, 0)

	for key, val := range diffs {
		resp = append(resp, gen.StepRunDiff{
			Key:      key,
			Modified: val,
			Original: originalValues[key],
		})
	}

	return gen.StepRunGetDiff200JSONResponse(gen.GetStepRunDiffResponse{
		Diffs: resp,
	}), nil
}
