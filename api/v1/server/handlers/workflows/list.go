package workflows

import (
	"math"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowList(ctx echo.Context, request gen.WorkflowListRequestObject) (gen.WorkflowListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	limit := 50
	offset := 0
	name := ""

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	if request.Params.Name != nil {
		name = *request.Params.Name
	}

	if limit <= 0 {
		limit = 50
	}

	if offset < 0 {
		offset = 0
	}

	listOpts := &v1.ListWorkflowsOpts{
		Limit:  &limit,
		Offset: &offset,
		Name:   &name,
	}

	listResp, err := t.config.V1.Workflows().ListWorkflows(tenantId, listOpts)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.Workflow, len(listResp.Rows))

	for i := range listResp.Rows {
		workflow := transformers.ToWorkflowFromSQLC(listResp.Rows[i])

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
