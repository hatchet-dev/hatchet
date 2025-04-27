package workflowruns

import (
	"context"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors" // Import added
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkflowRunsService) WorkflowRunUpdateReplay(ctx echo.Context, request gen.WorkflowRunUpdateReplayRequestObject) (gen.WorkflowRunUpdateReplayResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	workflowRunIds := make([]string, len(request.Body.WorkflowRunIds))

	for i := range request.Body.WorkflowRunIds {
		workflowRunIds[i] = request.Body.WorkflowRunIds[i].String()
	}

	limit := 500

	// make sure all workflow runs belong to the tenant
	filteredWorkflowRuns, err := t.config.EngineRepository.WorkflowRun().ListWorkflowRuns(ctx.Request().Context(), tenantId, &repository.ListWorkflowRunsOpts{
		Ids:   workflowRunIds,
		Limit: &limit,
	})

	if err != nil {
		return nil, err
	}

	// Check if all requested workflow runs were found
	if len(filteredWorkflowRuns.Rows) != len(workflowRunIds) {
		// Return 404 if not all runs were found
		return gen.WorkflowRunUpdateReplay404JSONResponse(
			apierrors.NewAPIErrors("One or more workflow runs not found or do not belong to the tenant"),
		), nil
	}

	var allErrs error

	for i := range filteredWorkflowRuns.Rows {
		// push to task queue
		err = t.config.MessageQueue.AddMessage(
			ctx.Request().Context(),
			msgqueue.WORKFLOW_PROCESSING_QUEUE,
			tasktypes.WorkflowRunReplayToTask(tenantId, sqlchelpers.UUIDToStr(filteredWorkflowRuns.Rows[i].WorkflowRun.ID)),
		)

		if err != nil {
			allErrs = multierror.Append(allErrs, err)
		}
	}

	if allErrs != nil {
		return nil, allErrs
	}

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 60*time.Second)
	defer cancel()

	// Fetch the runs again to return in the response (already verified they exist and belong to the tenant)
	// This second fetch is primarily for getting the latest state to return to the user.
	newWorkflowRuns, err := t.config.APIRepository.WorkflowRun().ListWorkflowRuns(dbCtx, tenantId, &repository.ListWorkflowRunsOpts{
		Ids:   workflowRunIds,
		Limit: &limit,
	})

	// This error should ideally not happen if the previous check passed, but handle defensively
	if err != nil {
		return nil, err
	}

	// If somehow the second fetch returns a different count (highly unlikely given the initial check),
	// it indicates a potential race condition or data inconsistency. Returning 404 is still appropriate.
	if len(newWorkflowRuns.Rows) != len(workflowRunIds) {
		return gen.WorkflowRunUpdateReplay404JSONResponse(
			apierrors.NewAPIErrors("Mismatch fetching workflow runs after queueing replay, please try again."),
		), nil
	}

	rows := make([]gen.WorkflowRun, len(newWorkflowRuns.Rows))

	for i, workflow := range newWorkflowRuns.Rows {
		workflowCp := workflow
		rows[i] = *transformers.ToWorkflowRunFromSQLC(workflowCp)
	}

	return gen.WorkflowRunUpdateReplay200JSONResponse(
		gen.ReplayWorkflowRunsResponse{
			WorkflowRuns: rows,
		},
	), nil
}
