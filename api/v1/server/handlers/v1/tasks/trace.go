package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TasksService) V1TaskGetTrace(ctx echo.Context, request gen.V1TaskGetTraceRequestObject) (gen.V1TaskGetTraceResponseObject, error) {
	task := ctx.Get("task").(*sqlcv1.V1TasksOlap)

	spans, err := t.config.V1.OTelCollector().ListSpansByTaskExternalID(
		ctx.Request().Context(), task.TenantID, task.ExternalID)
	if err != nil {
		return nil, err
	}

	apiSpans := convertToAPISpans(spans)

	return gen.V1TaskGetTrace200JSONResponse(gen.OtelSpanList{Rows: &apiSpans}), nil
}

func convertToAPISpans(spans []*repository.OtelSpanRow) []gen.OtelSpan {
	result := make([]gen.OtelSpan, len(spans))
	for i, s := range spans {
		result[i] = gen.OtelSpan{
			TraceId:            s.TraceID,
			SpanId:             s.SpanID,
			ParentSpanId:       &s.ParentSpanID,
			SpanName:           s.SpanName,
			SpanKind:           s.SpanKind,
			ServiceName:        s.ServiceName,
			StatusCode:         s.StatusCode,
			StatusMessage:      &s.StatusMessage,
			Duration:           int64(s.Duration), //nolint:gosec
			CreatedAt:          s.CreatedAt,
			ResourceAttributes: &s.ResourceAttributes,
			SpanAttributes:     &s.SpanAttributes,
			ScopeName:          &s.ScopeName,
			ScopeVersion:       &s.ScopeVersion,
		}
	}
	return result
}
