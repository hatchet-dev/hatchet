package workflows

import (
	"math"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkflowService) WorkflowList(ctx echo.Context, request gen.WorkflowListRequestObject) (gen.WorkflowListResponseObject, error) {
	tenant, err := populator.FromContext(ctx).GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	if request.Params.Limit == nil {
		request.Params.Limit = new(int)
		*request.Params.Limit = 50
	}

	if request.Params.Offset == nil {
		request.Params.Offset = new(int)
		*request.Params.Offset = 0
	}

	if request.Params.Name == nil {
		request.Params.Name = new(string)

	}

	name := *request.Params.Name

	limit := *request.Params.Limit
	offset := *request.Params.Offset

	listOpts := &repository.ListWorkflowsOpts{
		Limit:  &limit,
		Offset: &offset,
		Name:   &name,
	}

	listResp, err := t.config.APIRepository.Workflow().ListWorkflows(tenantId, listOpts)

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
