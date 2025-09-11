package tenants

import (
	"context"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantResourcePolicyGet(ctx echo.Context, request gen.TenantResourcePolicyGetRequestObject) (gen.TenantResourcePolicyGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	limits, err := t.config.EntitlementRepository.TenantLimit().GetLimits(context.Background(), tenantId)

	if err != nil {
		return nil, err
	}

	return gen.TenantResourcePolicyGet200JSONResponse(
		*transformers.ToTenantResourcePolicy(limits),
	), nil
}
