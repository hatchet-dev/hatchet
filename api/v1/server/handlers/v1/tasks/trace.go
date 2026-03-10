package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TasksService) V1TaskGetTrace(ctx echo.Context, request gen.V1TaskGetTraceRequestObject) (gen.V1TaskGetTraceResponseObject, error) {
	task := ctx.Get("task").(*sqlcv1.V1TasksOlap)

	limit := int64(1000)
	offset := int64(0)

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	result, err := t.config.V1.OTelCollector().ListSpansByTaskExternalID(ctx.Request().Context(), task.TenantID, task.ExternalID, offset, limit)
	if err != nil {
		return nil, err
	}

	return gen.V1TaskGetTrace200JSONResponse(transformers.ToV1OtelSpanList(result.Rows, limit, offset, result.Total)), nil
}
