package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *TasksService) V1LogLineList(ctx echo.Context, request gen.V1LogLineListRequestObject) (gen.V1LogLineListResponseObject, error) {
	tenant, err := populator.FromContext(ctx).GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	task := ctx.Get("task").(*sqlcv1.V1TasksOlap)

	logLines, err := t.config.V1.Logs().ListLogLines(ctx.Request().Context(), tenantId, task.ID, task.InsertedAt, &v1.ListLogsOpts{})

	if err != nil {
		return nil, err
	}

	rows := make([]gen.V1LogLine, len(logLines))

	for i, log := range logLines {
		rows[i] = *transformers.ToV1LogLine(log)
	}

	// use the total rows and limit to calculate the total pages
	totalPages := int64(1)
	currPage := int64(1)
	nextPage := int64(1)

	return gen.V1LogLineList200JSONResponse(
		gen.V1LogLineList{
			Rows: &rows,
			Pagination: &gen.PaginationResponse{
				NumPages:    &totalPages,
				CurrentPage: &currPage,
				NextPage:    &nextPage,
			},
		},
	), nil
}
