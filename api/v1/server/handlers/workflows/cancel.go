package workflows

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
)

func (t *WorkflowService) WorkflowRunCancel(ctx echo.Context, request gen.WorkflowRunCancelRequestObject) (gen.WorkflowRunCancelResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	runIds := request.Body.WorkflowRunIds

	canceledStepRunsChannel := make(chan []*db.StepRunModel, len(runIds))
	var returnErr error

	for _, runId := range runIds {
		go func(runId uuid.UUID) {
			// Lookup step runs for the workflow run
			runIdStr := runId.String()
			stepRuns, err := t.config.EngineRepository.StepRun().ListStepRunsByWorkflowRunId(ctx.Request().Context(), tenant.ID, runIdStr)
			if err != nil {
				returnErr = multierror.Append(err, fmt.Errorf("failed to list step runs for workflow run %s", runIdStr))
				canceledStepRunsChannel <- nil
				return
			}

			var canceledStepRuns []*db.StepRunModel

			for _, stepRun := range stepRuns {
				status := stepRun.StepRun.Status

				// If the step run is not in a final state, send a task to the taskqueue to cancel it
				if status != dbsqlc.StepRunStatusSUCCEEDED && status != dbsqlc.StepRunStatusFAILED && status != dbsqlc.StepRunStatusCANCELLED {
					var reason = "CANCELLED_BY_USER"
					// send a task to the taskqueue
					err = t.config.MessageQueue.AddMessage(
						ctx.Request().Context(),
						msgqueue.JOB_PROCESSING_QUEUE,
						tasktypes.StepRunCancelToTask(stepRun, reason),
					)
					if err != nil {
						returnErr = multierror.Append(err, fmt.Errorf("failed to send cancel task for step run %s", pgUUIDToStr(stepRun.StepRun.ID)))
						continue
					}

					stepRunId := pgUUIDToStr(stepRun.StepRun.ID)
					stepRunDb, err := t.config.APIRepository.StepRun().GetStepRunById(tenant.ID, stepRunId)
					if err != nil {
						returnErr = multierror.Append(err, fmt.Errorf("failed to get step run %s", stepRunId))
						continue
					}

					// Add the canceled step run to the list
					canceledStepRuns = append(canceledStepRuns, stepRunDb)
				}
			}

			canceledStepRunsChannel <- canceledStepRuns
		}(runId)
	}

	var allCanceledStepRuns []*db.StepRunModel
	for i := 0; i < len(runIds); i++ {
		canceledStepRuns := <-canceledStepRunsChannel
		if canceledStepRuns != nil {
			allCanceledStepRuns = append(allCanceledStepRuns, canceledStepRuns...)
		}
	}

	if returnErr != nil {
		return nil, returnErr
	}

	// Transform the canceled step runs to the response format
	canceledStepRunsResponse := make([]gen.StepRun, len(allCanceledStepRuns))
	for i, stepRun := range allCanceledStepRuns {

		res, err := transformers.ToStepRun(stepRun)

		if err != nil {
			return nil, err
		}

		canceledStepRunsResponse[i] = *res
	}

	// Return the list of canceled step runs in the response
	return gen.WorkflowRunCancel200JSONResponse(canceledStepRunsResponse), nil
}

func pgUUIDToStr(uuid pgtype.UUID) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid.Bytes[0:4], uuid.Bytes[4:6], uuid.Bytes[6:8], uuid.Bytes[8:10], uuid.Bytes[10:16])
}
