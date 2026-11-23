package concurrencyv1

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// V1ConcurrencyTotalsGet gets the tenant's concurrency totals across all strategies.
func (t *V1ConcurrencyService) V1ConcurrencyTotalsGet(ctx echo.Context, request gen.V1ConcurrencyTotalsGetRequestObject) (gen.V1ConcurrencyTotalsGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	totals, err := t.config.V1.ConcurrencyOverview().GetTotals(ctx.Request().Context(), tenant.ID)

	if err != nil {
		return nil, err
	}

	return gen.V1ConcurrencyTotalsGet200JSONResponse(
		transformers.ToV1ConcurrencyTotals(totals),
	), nil
}
