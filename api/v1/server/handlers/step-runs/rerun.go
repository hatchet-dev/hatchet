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
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
)

func (t *StepRunService) StepRunUpdateRerun(ctx echo.Context, request gen.StepRunUpdateRerunRequestObject) (gen.StepRunUpdateRerunResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	stepRun := ctx.Get("step-run").(*db.StepRunModel)

	// preflight check to make sure there's at least one worker to serve this request
	action := stepRun.Step().ActionID

	tenSecAgo := time.Now().Add(-10 * time.Second)

	workers, err := t.config.Repository.Worker().ListWorkers(tenant.ID, &repository.ListWorkersOpts{
		Action:             &action,
		LastHeartbeatAfter: &tenSecAgo,
	})

	if err != nil || len(workers) == 0 {
		return gen.StepRunUpdateRerun400JSONResponse(
			apierrors.NewAPIErrors("There are no workers available to execute this step run."),
		), nil
	}

	err = t.config.Repository.StepRun().ArchiveStepRunResult(tenant.ID, stepRun.ID)

	if err != nil {
		return nil, fmt.Errorf("could not archive step run result: %w", err)
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

	// update step run
	_, _, err = t.config.Repository.StepRun().UpdateStepRun(tenant.ID, stepRun.ID, &repository.UpdateStepRunOpts{
		Input:   inputBytes,
		Status:  repository.StepRunStatusPtr(db.StepRunStatusPending),
		IsRerun: true,
	})

	if err != nil {
		return nil, fmt.Errorf("could not update step run: %w", err)
	}

	// requeue the step run in the task queue
	jobRun, err := t.config.Repository.JobRun().GetJobRunById(tenant.ID, stepRun.JobRunID)

	if err != nil {
		return nil, fmt.Errorf("could not get job run: %w", err)
	}

	// send a task to the taskqueue
	err = t.config.TaskQueue.AddTask(
		ctx.Request().Context(),
		taskqueue.JOB_PROCESSING_QUEUE,
		tasktypes.StepRunQueuedToTask(jobRun.Job(), stepRun),
	)

	if err != nil {
		return nil, fmt.Errorf("could not add step queued task to task queue: %w", err)
	}

	stepRun, err = t.config.Repository.StepRun().GetStepRunById(tenant.ID, stepRun.ID)

	if err != nil {
		return nil, fmt.Errorf("could not get step run: %w", err)
	}

	res, err := transformers.ToStepRun(stepRun)

	if err != nil {
		return nil, fmt.Errorf("could not transform step run: %w", err)
	}

	return gen.StepRunUpdateRerun200JSONResponse(
		*res,
	), nil
}
