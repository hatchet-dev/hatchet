package observability

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *V1ObservabilityService) V1ObservabilityGetTrace(ctx echo.Context, request gen.V1ObservabilityGetTraceRequestObject) (gen.V1ObservabilityGetTraceResponseObject, error) {
	if !t.config.Observability.Enabled {
		return gen.V1ObservabilityGetTrace200JSONResponse(gen.OtelSpanList{}), nil
	}

	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	limit := int64(1000)
	offset := int64(0)

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	if limit < 1 {
		limit = 1000
	}

	if offset < 0 {
		offset = 0
	}

	result, err := t.config.V1.OLAP().ListSpansForRun(ctx.Request().Context(), tenant.ID, request.Params.RunExternalId, offset, limit)
	if err != nil {
		return nil, err
	}

	if len(result.Rows) == 0 {
		return gen.V1ObservabilityGetTrace200JSONResponse(gen.OtelSpanList{}), nil
	}

	return gen.V1ObservabilityGetTrace200JSONResponse(transformers.ToV1OtelSpanList(result.Rows, nil, limit, offset, result.Total)), nil
}
