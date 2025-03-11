package stepruns

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *StepRunService) StepRunUpdateCancel(ctx echo.Context, request gen.StepRunUpdateCancelRequestObject) (gen.StepRunUpdateCancelResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	stepRun := ctx.Get("step-run").(*repository.GetStepRunFull)

	// check to see if the step run is in a running or pending state
	status := stepRun.Status

	canCancel := status == dbsqlc.StepRunStatusRUNNING ||
		status == dbsqlc.StepRunStatusPENDING ||
		status == dbsqlc.StepRunStatusBACKOFF

	if !canCancel {
		return gen.StepRunUpdateCancel400JSONResponse(
			apierrors.NewAPIErrors("step run is not in a running, pending, or backoff state"),
		), nil
	}

	engineStepRun, err := t.config.EngineRepository.StepRun().GetStepRunForEngine(
		ctx.Request().Context(),
		tenantId,
		sqlchelpers.UUIDToStr(stepRun.ID),
	)

	if err != nil {
		return nil, fmt.Errorf("could not get step run for engine: %w", err)
	}

	var reason = "CANCELLED_BY_USER"

	// send a task to the taskqueue
	err = t.config.MessageQueue.AddMessage(
		ctx.Request().Context(),
		msgqueue.JOB_PROCESSING_QUEUE,
		tasktypes.StepRunCancelToTask(engineStepRun, reason, true),
	)

	if err != nil {
		return nil, fmt.Errorf("could not add step queued task to task queue: %w", err)
	}

	// wait for a short period of time
	for i := 0; i < 5; i++ {
		newStepRun, err := t.config.APIRepository.StepRun().GetStepRunById(sqlchelpers.UUIDToStr(stepRun.ID))

		if err != nil {
			return nil, fmt.Errorf("could not get step run: %w", err)
		}

		if newStepRun.Status != stepRun.Status {
			stepRun = newStepRun
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	return gen.StepRunUpdateCancel200JSONResponse(
		*transformers.ToStepRunFull(stepRun),
	), nil
}
