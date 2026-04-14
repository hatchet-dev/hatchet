package logs

import (
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	v1handlers "github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (t *LogsService) V1TenantLogLineList(ctx echo.Context, request gen.V1TenantLogLineListRequestObject) (gen.V1TenantLogLineListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	reqCtx, span := telemetry.NewSpan(ctx.Request().Context(), "GET /api/v1/stable/tenants/{tenant}/logs")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: tenantId},
	)

	var (
		limit            = int64(50)
		since            *time.Time
		until            *time.Time
		levels           []string
		search           *string
		orderByDirection *string
		attempt          *int32
		taskExternalIds  []uuid.UUID
		workflowIds      []uuid.UUID
		stepIds          []uuid.UUID
	)

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	clampedSince := v1handlers.ClampToRetentionPtr(request.Params.Since, tenant.DataRetentionPeriod)
	since = &clampedSince

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

	if request.Params.Attempt != nil {
		attemptInt32 := int32(*request.Params.Attempt) // nolint: gosec
		attempt = &attemptInt32
	}

	if request.Params.TaskExternalIds != nil {
		taskExternalIds = append(taskExternalIds, *request.Params.TaskExternalIds...)
	}

	if request.Params.WorkflowIds != nil {
		workflowIds = append(workflowIds, *request.Params.WorkflowIds...)
	}

	if request.Params.StepIds != nil {
		stepIds = append(stepIds, *request.Params.StepIds...)
	}

	limitInt := int(limit)

	opts := &v1.ListLogsOpts{
		Limit:            &limitInt,
		Since:            since,
		Until:            until,
		Search:           search,
		Levels:           levels,
		OrderByDirection: orderByDirection,
		Attempt:          attempt,
		TaskExternalIds:  taskExternalIds,
		WorkflowIds:      workflowIds,
		StepIds:          stepIds,
	}

	logLines, err := t.config.V1.Logs().ListLogLines(reqCtx, tenantId, opts)

	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "log_lines.count", Value: len(logLines)},
	)

	t.config.Analytics.Count(ctx.Request().Context(), analytics.Log, analytics.List, analytics.Properties{
		"has_search":            search != nil,
		"has_levels":            len(levels) > 0,
		"has_task_external_ids": len(taskExternalIds) > 0,
		"has_workflow_ids":      len(workflowIds) > 0,
		"has_step_ids":          len(stepIds) > 0,
	})

	rows := make([]gen.V1LogLine, len(logLines))

	for i, log := range logLines {
		rows[i] = *transformers.ToV1LogLine(log)
	}

	totalPages := int64(0)
	currPage := int64(0)
	nextPage := int64(0)

	return gen.V1TenantLogLineList200JSONResponse(
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
