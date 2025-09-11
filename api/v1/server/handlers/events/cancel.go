package events

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *EventService) EventUpdateCancel(ctx echo.Context, request gen.EventUpdateCancelRequestObject) (gen.EventUpdateCancelResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	eventIds := make([]string, len(request.Body.EventIds))

	runIds := make([]string, 0)

	for i := range request.Body.EventIds {
		eventIds[i] = request.Body.EventIds[i].String()
	}

	for i := range eventIds {
		eventId := eventIds[i]

		runs, err := t.config.EngineRepository.WorkflowRun().ListWorkflowRuns(ctx.Request().Context(), tenantId, &repository.ListWorkflowRunsOpts{
			EventId: &eventId,
		})

		if err != nil {
			return nil, err
		}

		for _, run := range runs.Rows {
			runCp := run
			runIds = append(runIds, sqlchelpers.UUIDToStr(runCp.WorkflowRun.ID))
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var cancelledWorkflowRunIds []uuid.UUID
	var returnErr error

	for _, runId := range runIds {
		runIdCp := runId
		wg.Add(1)
		go func(runId string) {
			defer wg.Done()

			// Lookup step runs for the workflow run
			jobRun, err := t.config.EngineRepository.JobRun().ListJobRunsForWorkflowRun(ctx.Request().Context(), tenantId, runId)
			if err != nil {
				returnErr = multierror.Append(err, fmt.Errorf("failed to list job runs for workflow run %s", runId))
				return
			}

			for _, jobRun := range jobRun {
				// If the step run is not in a final state, send a task to the taskqueue to cancel it
				var reason = "CANCELLED_BY_USER"
				// send a task to the taskqueue
				jobRunId := sqlchelpers.UUIDToStr(jobRun.ID)
				err = t.config.MessageQueue.AddMessage(
					ctx.Request().Context(),
					msgqueue.JOB_PROCESSING_QUEUE,
					tasktypes.JobRunCancelledToTask(tenantId, jobRunId, &reason),
				)
				if err != nil {
					returnErr = multierror.Append(err, fmt.Errorf("failed to send cancel task for job run %s", jobRunId))
					continue
				}
			}

			// Add the canceled workflow run ID to the slice
			mu.Lock()
			cancelledWorkflowRunIds = append(cancelledWorkflowRunIds, uuid.MustParse(runId))
			mu.Unlock()
		}(runIdCp)
	}

	wg.Wait()

	if returnErr != nil {
		return nil, returnErr
	}

	// Create a new instance of gen.WorkflowRunCancel200JSONResponse and assign canceledWorkflowRunUUIDs to its WorkflowRunIds field
	response := gen.EventUpdateCancel200JSONResponse{
		WorkflowRunIds: &cancelledWorkflowRunIds,
	}

	return response, nil
}
