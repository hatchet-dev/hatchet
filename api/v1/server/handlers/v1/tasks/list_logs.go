package tasks

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *TasksService) V1LogLineList(ctx echo.Context, request gen.V1LogLineListRequestObject) (gen.V1LogLineListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	task := ctx.Get("task").(*sqlcv1.V1TasksOlap)

	reqCtx, span := telemetry.NewSpan(ctx.Request().Context(), fmt.Sprintf("GET /api/v1/stable/tasks/%d/logs", task.ID))
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: tenantId},
		telemetry.AttributeKV{Key: "task.id", Value: task.ID},
	)

	logLines, err := t.config.V1.Logs().ListLogLines(reqCtx, tenantId, task.ID, task.InsertedAt, &v1.ListLogsOpts{})

	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "log_lines.count", Value: len(logLines)},
	)

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
