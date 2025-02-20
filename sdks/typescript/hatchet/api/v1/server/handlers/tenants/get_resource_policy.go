package tenants

import (
	"context"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *TenantService) TenantResourcePolicyGet(ctx echo.Context, request gen.TenantResourcePolicyGetRequestObject) (gen.TenantResourcePolicyGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	limits, err := t.config.EntitlementRepository.TenantLimit().GetLimits(context.Background(), tenant.ID)

	if err != nil {
		return nil, err
	}

	return gen.TenantResourcePolicyGet200JSONResponse(
		*transformers.ToTenantResourcePolicy(limits),
	), nil
}
