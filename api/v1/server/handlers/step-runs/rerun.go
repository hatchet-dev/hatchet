package stepruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *StepRunService) StepRunUpdateRerun(ctx echo.Context, request gen.StepRunUpdateRerunRequestObject) (gen.StepRunUpdateRerunResponseObject, error) {
	panic("not implemented in v2")

	// tenant := ctx.Get("tenant").(*db.TenantModel)
	// stepRun := ctx.Get("step-run").(*repository.GetStepRunFull)

	// // preflight check to verify step run status and worker availability
	// err := t.config.EngineRepository.StepRun().PreflightCheckReplayStepRun(
	// 	ctx.Request().Context(),
	// 	tenant.ID,
	// 	sqlchelpers.UUIDToStr(stepRun.ID),
	// )

	// if err != nil {

	// 	if errors.Is(err, repository.ErrNoWorkerAvailable) {
	// 		return gen.StepRunUpdateRerun400JSONResponse(
	// 			apierrors.NewAPIErrors("There are no workers available to execute this step run."),
	// 		), nil
	// 	}

	// 	if errors.Is(err, repository.ErrPreflightReplayStepRunNotInFinalState) {
	// 		return gen.StepRunUpdateRerun400JSONResponse(
	// 			apierrors.NewAPIErrors("Step run cannot be replayed because it is not finished running yet."),
	// 		), nil
	// 	}

	// 	if errors.Is(err, repository.ErrPreflightReplayChildStepRunNotInFinalState) {
	// 		return gen.StepRunUpdateRerun400JSONResponse(
	// 			apierrors.NewAPIErrors("Step run cannot be replayed because it has child step runs that are not finished running yet."),
	// 		), nil
	// 	}

	// 	return nil, fmt.Errorf("could not preflight check step run: %w", err)
	// }

	// // make sure input can be marshalled and unmarshalled to input type
	// inputBytes, err := json.Marshal(request.Body.Input)

	// if err != nil {
	// 	return gen.StepRunUpdateRerun400JSONResponse(
	// 		apierrors.NewAPIErrors("Invalid input"),
	// 	), nil
	// }

	// data := &datautils.StepRunData{}

	// if err := json.Unmarshal(inputBytes, data); err != nil {
	// 	return gen.StepRunUpdateRerun400JSONResponse(
	// 		apierrors.NewAPIErrors("Invalid input"),
	// 	), nil
	// }

	// inputBytes, err = json.Marshal(data)

	// if err != nil {
	// 	return gen.StepRunUpdateRerun400JSONResponse(
	// 		apierrors.NewAPIErrors("Invalid input"),
	// 	), nil
	// }

	// engineStepRun, err := t.config.EngineRepository.StepRun().GetStepRunForEngine(
	// 	ctx.Request().Context(),
	// 	tenant.ID,
	// 	sqlchelpers.UUIDToStr(stepRun.ID),
	// )

	// if err != nil {
	// 	return nil, fmt.Errorf("could not get step run for engine: %w", err)
	// }

	// // send a task to the taskqueue
	// err = t.config.MessageQueue.SendMessage(
	// 	ctx.Request().Context(),
	// 	msgqueue.JOB_PROCESSING_QUEUE,
	// 	tasktypes.StepRunReplayToTask(engineStepRun, inputBytes),
	// )

	// if err != nil {
	// 	return nil, fmt.Errorf("could not add step queued task to task queue: %w", err)
	// }

	// var result *repository.GetStepRunFull

	// // wait for a short period of time
	// for i := 0; i < 5; i++ {
	// 	result, err = t.config.APIRepository.StepRun().GetStepRunById(sqlchelpers.UUIDToStr(stepRun.ID))

	// 	if err != nil {
	// 		return nil, fmt.Errorf("could not get step run: %w", err)
	// 	}

	// 	if result.Status != stepRun.Status {
	// 		break
	// 	}

	// 	time.Sleep(100 * time.Millisecond)
	// }

	// return gen.StepRunUpdateRerun200JSONResponse(
	// 	*transformers.ToStepRunFull(result),
	// ), nil
}
