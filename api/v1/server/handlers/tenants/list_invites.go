package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantInviteList(ctx echo.Context, request gen.TenantInviteListRequestObject) (gen.TenantInviteListResponseObject, error) {
	tenant, err := populator.FromContext(ctx).GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	tenantInvites, err := t.config.APIRepository.TenantInvite().ListTenantInvitesByTenantId(ctx.Request().Context(), tenantId, &repository.ListTenantInvitesOpts{
		Expired: repository.BoolPtr(false),
		Status:  repository.StringPtr("PENDING"),
	})

	if err != nil {
		return nil, err
	}

	rows := make([]gen.TenantInvite, len(tenantInvites))

	for i := range tenantInvites {
		rows[i] = *transformers.ToTenantInviteLink(tenantInvites[i])
	}

	return gen.TenantInviteList200JSONResponse{
		Rows: &rows,
	}, nil
}
