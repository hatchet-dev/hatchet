package tenants

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantDelete(ctx echo.Context, request gen.TenantDeleteRequestObject) (gen.TenantDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	user := ctx.Get("user").(*dbsqlc.User)

	unauthorizedErr := gen.TenantDelete403JSONResponse(
		gen.APIError{
			Code:        &[]uint64{403}[0],
			Description: "Only tenant owners can delete the tenant",
		},
	)

	member, err := t.config.APIRepository.Tenant().GetTenantMemberByUserID(ctx.Request().Context(), sqlchelpers.UUIDToStr(tenant.ID), sqlchelpers.UUIDToStr(user.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return unauthorizedErr, nil
		}

		return nil, err
	}

	if member.Role != dbsqlc.TenantMemberRoleOWNER {
		return unauthorizedErr, nil
	}

	err = t.config.APIRepository.Tenant().SoftDeleteTenant(ctx.Request().Context(), sqlchelpers.UUIDToStr(tenant.ID))

	if err != nil {
		return nil, err
	}

	return gen.TenantDelete204Response{}, nil
}
