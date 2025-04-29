package workflows

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkflowService) CronWorkflowList(ctx echo.Context, request gen.CronWorkflowListRequestObject) (gen.CronWorkflowListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	limit := 50
	offset := 0
	orderDirection := "DESC"
	orderBy := "createdAt"

	listOpts := &repository.ListCronWorkflowsOpts{
		Limit:          &limit,
		Offset:         &offset,
		OrderBy:        &orderBy,
		OrderDirection: &orderDirection,
	}

	if request.Params.OrderByField != nil {
		orderBy = string(*request.Params.OrderByField)
		listOpts.OrderBy = &orderBy
	}

	if request.Params.OrderByDirection != nil {
		orderDirection = string(*request.Params.OrderByDirection)
		listOpts.OrderDirection = &orderDirection
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

	if request.Params.CronName != nil {
		listOpts.CronName = request.Params.CronName
	}

	if request.Params.WorkflowName != nil {
		listOpts.WorkflowName = request.Params.WorkflowName
	}

	if request.Params.AdditionalMetadata != nil {
		additionalMetadata := make(map[string]interface{}, len(*request.Params.AdditionalMetadata))

		for _, v := range *request.Params.AdditionalMetadata {
			splitValue := strings.Split(fmt.Sprintf("%v", v), ":")

			if len(splitValue) == 2 {
				additionalMetadata[splitValue[0]] = splitValue[1]
			} else {
				return gen.CronWorkflowList400JSONResponse(apierrors.NewAPIErrors("Additional metadata filters must be in the format key:value.")), nil

			}
		}

		listOpts.AdditionalMetadata = additionalMetadata
	}

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	crons, count, err := t.config.APIRepository.Workflow().ListCronWorkflows(dbCtx, tenantId, listOpts)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.CronWorkflows, len(crons))

	for i, workflow := range crons {
		workflowCp := workflow
		rows[i] = *transformers.ToCronWorkflowsFromSQLC(workflowCp)
	}

	// use the total rows and limit to calculate the total pages
	totalPages := int64(math.Ceil(float64(count) / float64(limit)))
	currPage := 1 + int64(math.Ceil(float64(offset)/float64(limit)))
	nextPage := currPage + 1

	if currPage == totalPages {
		nextPage = currPage
	}

	return gen.CronWorkflowList200JSONResponse(
		gen.CronWorkflowsList{
			Rows: &rows,
			Pagination: &gen.PaginationResponse{
				NumPages:    &totalPages,
				CurrentPage: &currPage,
				NextPage:    &nextPage,
			},
		},
	), nil
}
