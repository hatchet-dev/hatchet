package tasks

import (
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *TasksService) V1LogLineList(ctx echo.Context, request gen.V1LogLineListRequestObject) (gen.V1LogLineListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	task := ctx.Get("task").(*sqlcv1.V1TasksOlap)

	reqCtx, span := telemetry.NewSpan(ctx.Request().Context(), "GET /api/v1/stable/tasks/{task}/logs")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: tenantId},
		telemetry.AttributeKV{Key: "task.id", Value: task.ID},
	)

	var (
		limit            = int64(50)
		since            *time.Time
		until            *time.Time
		levels           []string
		search           *string
		orderByDirection *string
	)

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Since != nil {
		since = request.Params.Since
	}

	if request.Params.Until != nil {
		until = request.Params.Until
	}

	if request.Params.Levels != nil {
		for _, level := range *request.Params.Levels {
			levels = append(levels, string(level))
		}
	}

	if request.Params.Search != nil {
		search = request.Params.Search
	}

	if request.Params.OrderByDirection != nil {
		orderByDirectionStr := string(*request.Params.OrderByDirection)
		orderByDirection = &orderByDirectionStr
	}

	limitInt := int(limit)

	opts := &v1.ListLogsOpts{
		Limit:            &limitInt,
		Since:            since,
		Until:            until,
		Search:           search,
		Levels:           levels,
		OrderByDirection: orderByDirection,
	}

	logLines, err := t.config.V1.Logs().ListLogLines(reqCtx, tenantId, task.ExternalID, opts)

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

	totalPages := int64(0)
	currPage := int64(0)
	nextPage := int64(0)

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
