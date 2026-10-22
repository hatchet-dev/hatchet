package concurrencyv1

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *V1ConcurrencyService) V1ConcurrencyStrategyList(ctx echo.Context, request gen.V1ConcurrencyStrategyListRequestObject) (gen.V1ConcurrencyStrategyListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	strategies, err := t.config.V1.ConcurrencyOverview().ListStrategySummaries(ctx.Request().Context(), tenant.ID)

	if err != nil {
		return nil, err
	}

	return gen.V1ConcurrencyStrategyList200JSONResponse(
		transformers.ToV1ConcurrencyStrategyList(strategies),
	), nil
}
