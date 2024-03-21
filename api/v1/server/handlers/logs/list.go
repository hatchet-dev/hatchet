package logs

import (
	"math"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (t *LogService) LogLineList(ctx echo.Context, request gen.LogLineListRequestObject) (gen.LogLineListResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	stepRun := ctx.Get("step-run").(*db.StepRunModel)

	limit := 1000
	offset := 0

	listOpts := &repository.ListLogsOpts{
		Limit:     &limit,
		Offset:    &offset,
		StepRunId: &stepRun.ID,
	}

	if request.Params.Search != nil {
		listOpts.Search = request.Params.Search
	}

	if request.Params.Levels != nil {
		levels := make([]string, len(*request.Params.Levels))

		for i, level := range *request.Params.Levels {
			levels[i] = string(level)
		}

		listOpts.Levels = levels
	}

	if request.Params.OrderByField != nil {
		listOpts.OrderBy = repository.StringPtr(string(*request.Params.OrderByField))
	}

	if request.Params.OrderByDirection != nil {
		listOpts.OrderDirection = repository.StringPtr(strings.ToUpper(string(*request.Params.OrderByDirection)))
	}

	if request.Params.Limit != nil {
		limit = int(*request.Params.Limit)
		listOpts.Limit = &limit
	}

	if request.Params.Offset != nil {
		offset = int(*request.Params.Offset)
		listOpts.Offset = &offset
	}

	listRes, err := t.config.APIRepository.Log().ListLogLines(tenant.ID, listOpts)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.LogLine, len(listRes.Rows))

	for i, log := range listRes.Rows {
		rows[i] = *transformers.ToLogFromSQLC(log)
	}

	// use the total rows and limit to calculate the total pages
	totalPages := int64(math.Ceil(float64(listRes.Count) / float64(limit)))
	currPage := 1 + int64(math.Ceil(float64(offset)/float64(limit)))
	nextPage := currPage + 1

	if currPage == totalPages {
		nextPage = currPage
	}

	return gen.LogLineList200JSONResponse(
		gen.LogLineList{
			Rows: &rows,
			Pagination: &gen.PaginationResponse{
				NumPages:    &totalPages,
				NextPage:    &nextPage,
				CurrentPage: &currPage,
			},
		},
	), nil
}
