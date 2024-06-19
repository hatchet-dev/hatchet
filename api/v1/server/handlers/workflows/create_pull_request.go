package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/services/worker"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *WorkflowService) StepRunUpdateCreatePr(ctx echo.Context, request gen.StepRunUpdateCreatePrRequestObject) (gen.StepRunUpdateCreatePrResponseObject, error) {
	stepRun := ctx.Get("step-run").(*db.StepRunModel)

	// trigger the workflow run
	_, err := t.config.InternalClient.Admin().RunWorkflow(worker.PullRequestWorkflow, &worker.StartPullRequestEvent{
		TenantID:   stepRun.TenantID,
		StepRunID:  stepRun.ID,
		BranchName: request.Body.BranchName,
	})

	if err != nil {
		return nil, err
	}

	return gen.StepRunUpdateCreatePr200JSONResponse{}, nil
}
