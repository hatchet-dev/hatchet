package stepruns

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
)

func (t *StepRunService) StepRunUpdateCancel(ctx echo.Context, request gen.StepRunUpdateCancelRequestObject) (gen.StepRunUpdateCancelResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	stepRun := ctx.Get("step-run").(*db.StepRunModel)

	// check to see if the step run is in a running or pending state
	status := stepRun.Status

	if status == db.StepRunStatusFailed || status == db.StepRunStatusCancelled || status == db.StepRunStatusSucceeded {
		return gen.StepRunUpdateCancel400JSONResponse(
			apierrors.NewAPIErrors("step run is not in a running or pending state"),
		), nil
	}

	engineStepRun, err := t.config.EngineRepository.StepRun().GetStepRunForEngine(tenant.ID, stepRun.ID)

	if err != nil {
		return nil, fmt.Errorf("could not get step run for engine: %w", err)
	}

	var reason = "CANCELLED_BY_USER"

	// send a task to the taskqueue
	err = t.config.MessageQueue.AddMessage(
		ctx.Request().Context(),
		msgqueue.JOB_PROCESSING_QUEUE,
		tasktypes.StepRunCancelToTask(engineStepRun, reason),
	)

	if err != nil {
		return nil, fmt.Errorf("could not add step queued task to task queue: %w", err)
	}

	// wait for a short period of time
	for i := 0; i < 5; i++ {
		newStepRun, err := t.config.APIRepository.StepRun().GetStepRunById(tenant.ID, stepRun.ID)

		if err != nil {
			return nil, fmt.Errorf("could not get step run: %w", err)
		}

		if newStepRun.Status != stepRun.Status {
			stepRun = newStepRun
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	res, err := transformers.ToStepRun(stepRun)

	if err != nil {
		return nil, fmt.Errorf("could not transform step run: %w", err)
	}

	return gen.StepRunUpdateCancel200JSONResponse(
		*res,
	), nil
}
