package tasks

import (
	"math"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *TasksService) V1LogLineList(ctx echo.Context, request gen.V1LogLineListRequestObject) (gen.V1LogLineListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	task := ctx.Get("task").(*sqlcv1.V1TasksOlap)

	var (
		limit      = int64(50)
		offset     = int64(0)
		since      *time.Time
		nextPage   *int64
		currPage   *int64
		totalPages *int64
	)

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	if request.Params.Since != nil {
		since = request.Params.Since
	}

	limitInt := int(limit)
	offsetInt := int(offset)

	opts := &v1.ListLogsOpts{
		Limit:  &limitInt,
		Offset: &offsetInt,
		Since:  since,
	}

	logLines, count, err := t.config.V1.Logs().ListLogLines(ctx.Request().Context(), tenantId, task.ID, task.InsertedAt, opts)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.V1LogLine, len(logLines))

	for i, log := range logLines {
		rows[i] = *transformers.ToV1LogLine(log)
	}

	if since != nil && offset == 0 {
		hasMore := len(logLines) == int(limit) && len(logLines) > 0
		if hasMore {
			next := int64(1)
			nextPage = &next
		}
	} else {
		curr := (offset / limit) + 1
		currPage = &curr

		total := int64(math.Ceil(float64(count) / float64(limit)))
		totalPages = &total

		if curr < total {
			next := curr + 1
			nextPage = &next
		}
	}

	return gen.V1LogLineList200JSONResponse(
		gen.V1LogLineList{
			Rows: &rows,
			Pagination: &gen.PaginationResponse{
				NumPages:    totalPages,
				CurrentPage: currPage,
				NextPage:    nextPage,
			},
		},
	), nil
}
