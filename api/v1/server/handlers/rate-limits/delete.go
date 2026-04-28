package rate_limits

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *RateLimitService) RateLimitDelete(ctx echo.Context, request gen.RateLimitDeleteRequestObject) (gen.RateLimitDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	key := request.Params.Key
	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()
	err := t.config.V1.RateLimit().DeleteRateLimits(dbCtx, tenantId, key)
	if err != nil {
		return gen.RateLimitDelete400JSONResponse(apierrors.NewAPIErrors("failed to delete rate-limit")), nil
	}
	return gen.RateLimitDelete204Response{}, nil
}
