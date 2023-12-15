package workflows

import (
	"math"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/labstack/echo/v4"
)

func (t *WorkflowService) WorkflowList(ctx echo.Context, request gen.WorkflowListRequestObject) (gen.WorkflowListResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	limit := 50
	offset := 0

	listOpts := &repository.ListWorkflowsOpts{
		Limit:  &limit,
		Offset: &offset,
	}

	listResp, err := t.config.Repository.Workflow().ListWorkflows(tenant.ID, listOpts)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.Workflow, len(listResp.Rows))

	for i := range listResp.Rows {
		workflow, err := transformers.ToWorkflow(listResp.Rows[i].WorkflowModel, listResp.Rows[i].LatestRun)

		if err != nil {
			return nil, err
		}

		rows[i] = *workflow
	}

	// use the total rows and limit to calculate the total pages
	totalPages := int64(math.Ceil(float64(listResp.Count) / float64(limit)))
	currPage := 1 + int64(math.Ceil(float64(offset)/float64(limit)))
	nextPage := currPage + 1

	if currPage == totalPages {
		nextPage = currPage
	}

	return gen.WorkflowList200JSONResponse(
		gen.WorkflowList{
			Rows: &rows,
			Pagination: &gen.PaginationResponse{
				NumPages:    &totalPages,
				CurrentPage: &currPage,
				NextPage:    &nextPage,
			},
		},
	), nil
}
