package workflowruns

import (
	"math"

	"github.com/hashicorp/go-multierror"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (t *WorkflowRunsService) WorkflowRunUpdateReplay(ctx echo.Context, request gen.WorkflowRunUpdateReplayRequestObject) (gen.WorkflowRunUpdateReplayResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	workflowRunIds := make([]string, len(request.Body.WorkflowRunIds))

	for i := range request.Body.WorkflowRunIds {
		workflowRunIds[i] = request.Body.WorkflowRunIds[i].String()
	}

	limit := 500

	// make sure all workflow runs belong to the tenant
	filteredWorkflowRuns, err := t.config.EngineRepository.WorkflowRun().ListWorkflowRuns(ctx.Request().Context(), tenant.ID, &repository.ListWorkflowRunsOpts{
		Ids:   workflowRunIds,
		Limit: &limit,
	})

	if err != nil {
		return nil, err
	}

	var allErrs error

	for i := range filteredWorkflowRuns.Rows {
		// push to task queue
		err = t.config.MessageQueue.AddMessage(
			ctx.Request().Context(),
			msgqueue.WORKFLOW_PROCESSING_QUEUE,
			tasktypes.WorkflowRunReplayToTask(tenant.ID, sqlchelpers.UUIDToStr(filteredWorkflowRuns.Rows[i].WorkflowRun.ID)),
		)

		if err != nil {
			allErrs = multierror.Append(allErrs, err)
		}
	}

	if allErrs != nil {
		return nil, allErrs
	}

	newWorkflowRuns, err := t.config.APIRepository.WorkflowRun().ListWorkflowRuns(tenant.ID, &repository.ListWorkflowRunsOpts{
		Ids:   workflowRunIds,
		Limit: &limit,
	})

	if err != nil {
		return nil, err
	}

	rows := make([]gen.WorkflowRun, len(newWorkflowRuns.Rows))

	for i, workflow := range newWorkflowRuns.Rows {
		workflowCp := workflow
		rows[i] = *transformers.ToWorkflowRunFromSQLC(workflowCp)
	}

	// use the total rows and limit to calculate the total pages
	totalPages := int64(math.Ceil(float64(newWorkflowRuns.Count) / float64(limit)))
	currPage := 1 + int64(math.Ceil(float64(0)/float64(limit)))
	nextPage := currPage + 1

	if currPage == totalPages {
		nextPage = currPage
	}

	return gen.WorkflowRunUpdateReplay200JSONResponse(
		gen.WorkflowRunList{
			Rows: &rows,
			Pagination: &gen.PaginationResponse{
				NumPages:    &totalPages,
				CurrentPage: &currPage,
				NextPage:    &nextPage,
			},
		},
	), nil
}
