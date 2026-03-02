package tenants

import (
	"context"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) TenantResourcePolicyGet(ctx echo.Context, request gen.TenantResourcePolicyGetRequestObject) (gen.TenantResourcePolicyGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	limits, err := t.config.V1.TenantLimit().GetLimits(context.Background(), tenantId)

	if err != nil {
		return nil, err
	}

	return gen.TenantResourcePolicyGet200JSONResponse(
		*transformers.ToTenantResourcePolicy(limits),
	), nil
}
