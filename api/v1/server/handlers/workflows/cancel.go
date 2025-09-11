package workflows

import (
	"fmt"
	"sync"

	"github.com/labstack/echo/v4"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkflowService) WorkflowRunCancel(ctx echo.Context, request gen.WorkflowRunCancelRequestObject) (gen.WorkflowRunCancelResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	runIds := request.Body.WorkflowRunIds

	var wg sync.WaitGroup
	var mu sync.Mutex
	var cancelledWorkflowRunIds []uuid.UUID
	var returnErr error

	for _, runId := range runIds {
		wg.Add(1)
		go func(runId uuid.UUID) {
			defer wg.Done()

			// Lookup step runs for the workflow run
			runIdStr := runId.String()
			jobRun, err := t.config.EngineRepository.JobRun().ListJobRunsForWorkflowRun(ctx.Request().Context(), tenantId, runIdStr)
			if err != nil {
				returnErr = multierror.Append(err, fmt.Errorf("failed to list job runs for workflow run %s", runIdStr))
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
			cancelledWorkflowRunIds = append(cancelledWorkflowRunIds, runId)
			mu.Unlock()
		}(runId)
	}

	wg.Wait()

	if returnErr != nil {
		return nil, returnErr
	}

	// Create a new instance of gen.WorkflowRunCancel200JSONResponse and assign canceledWorkflowRunUUIDs to its WorkflowRunIds field
	response := gen.WorkflowRunCancel200JSONResponse{
		WorkflowRunIds: &cancelledWorkflowRunIds,
	}

	// Return the response
	return response, nil
}
