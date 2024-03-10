package stepruns

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
)

func (t *StepRunService) StepRunUpdateRerun(ctx echo.Context, request gen.StepRunUpdateRerunRequestObject) (gen.StepRunUpdateRerunResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	stepRun := ctx.Get("step-run").(*db.StepRunModel)

	// preflight check to make sure there's at least one worker to serve this request
	action := stepRun.Step().ActionID

	sixSecAgo := time.Now().Add(-6 * time.Second)

	workers, err := t.config.Repository.Worker().ListWorkers(tenant.ID, &repository.ListWorkersOpts{
		Action:             &action,
		LastHeartbeatAfter: &sixSecAgo,
		Assignable:         repository.BoolPtr(true),
	})

	if err != nil || len(workers) == 0 {
		return gen.StepRunUpdateRerun400JSONResponse(
			apierrors.NewAPIErrors("There are no workers available to execute this step run."),
		), nil
	}

	// make sure input can be marshalled and unmarshalled to input type
	inputBytes, err := json.Marshal(request.Body.Input)

	if err != nil {
		return gen.StepRunUpdateRerun400JSONResponse(
			apierrors.NewAPIErrors("Invalid input"),
		), nil
	}

	data := &datautils.StepRunData{}

	if err := json.Unmarshal(inputBytes, data); err != nil || data == nil {
		return gen.StepRunUpdateRerun400JSONResponse(
			apierrors.NewAPIErrors("Invalid input"),
		), nil
	}

	inputBytes, err = json.Marshal(data)

	if err != nil {
		return gen.StepRunUpdateRerun400JSONResponse(
			apierrors.NewAPIErrors("Invalid input"),
		), nil
	}

	// set the job run and workflow run to running status
	err = t.config.Repository.JobRun().SetJobRunStatusRunning(tenant.ID, stepRun.JobRunID)

	if err != nil {
		return nil, err
	}

	engineStepRun, err := t.config.Repository.StepRun().GetStepRunForEngine(tenant.ID, stepRun.ID)

	if err != nil {
		return nil, fmt.Errorf("could not get step run for engine: %w", err)
	}

	// send a task to the taskqueue
	err = t.config.MessageQueue.AddMessage(
		ctx.Request().Context(),
		msgqueue.JOB_PROCESSING_QUEUE,
		tasktypes.StepRunRetryToTask(engineStepRun, inputBytes),
	)

	if err != nil {
		return nil, fmt.Errorf("could not add step queued task to task queue: %w", err)
	}

	// wait for a short period of time
	for i := 0; i < 5; i++ {
		newStepRun, err := t.config.Repository.StepRun().GetStepRunById(tenant.ID, stepRun.ID)

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

	return gen.StepRunUpdateRerun200JSONResponse(
		*res,
	), nil
}
