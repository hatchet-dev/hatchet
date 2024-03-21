package workflows

import (
	"math"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (t *WorkflowService) WorkflowRunList(ctx echo.Context, request gen.WorkflowRunListRequestObject) (gen.WorkflowRunListResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	limit := 50
	offset := 0

	listOpts := &repository.ListWorkflowRunsOpts{
		Limit:  &limit,
		Offset: &offset,
	}

	if request.Params.Limit != nil {
		limit = int(*request.Params.Limit)
		listOpts.Limit = &limit
	}

	if request.Params.Offset != nil {
		offset = int(*request.Params.Offset)
		listOpts.Offset = &offset
	}

	if request.Params.WorkflowId != nil {
		workflowIdStr := request.Params.WorkflowId.String()
		listOpts.WorkflowId = &workflowIdStr
	}

	if request.Params.EventId != nil {
		eventIdStr := request.Params.EventId.String()
		listOpts.EventId = &eventIdStr
	}

	workflowRuns, err := t.config.APIRepository.WorkflowRun().ListWorkflowRuns(tenant.ID, listOpts)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.WorkflowRun, len(workflowRuns.Rows))

	for i, workflow := range workflowRuns.Rows {
		workflowCp := workflow
		rows[i] = *transformers.ToWorkflowRunFromSQLC(workflowCp)
	}

	// use the total rows and limit to calculate the total pages
	totalPages := int64(math.Ceil(float64(workflowRuns.Count) / float64(limit)))
	currPage := 1 + int64(math.Ceil(float64(offset)/float64(limit)))
	nextPage := currPage + 1

	if currPage == totalPages {
		nextPage = currPage
	}

	return gen.WorkflowRunList200JSONResponse(
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
