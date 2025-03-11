package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantMemberDelete(ctx echo.Context, request gen.TenantMemberDeleteRequestObject) (gen.TenantMemberDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	tenantMember := ctx.Get("tenant-member").(*dbsqlc.PopulateTenantMembersRow)

	if tenantMember.Role != dbsqlc.TenantMemberRoleOWNER {
		return gen.TenantMemberDelete403JSONResponse(
			apierrors.NewAPIErrors("Only owners can delete members"),
		), nil
	}

	memberToDelete, err := t.config.APIRepository.Tenant().GetTenantMemberByID(ctx.Request().Context(), request.Member.String())

	if err != nil {
		return nil, err
	}

	if sqlchelpers.UUIDToStr(tenantMember.UserId) == sqlchelpers.UUIDToStr(memberToDelete.UserId) {
		return gen.TenantMemberDelete403JSONResponse(
			apierrors.NewAPIErrors("You cannot delete yourself"),
		), nil
	}

	if sqlchelpers.UUIDToStr(memberToDelete.TenantId) != tenantId {
		return gen.TenantMemberDelete404JSONResponse(
			apierrors.NewAPIErrors("Member not found"),
		), nil
	}

	err = t.config.APIRepository.Tenant().DeleteTenantMember(ctx.Request().Context(), sqlchelpers.UUIDToStr(memberToDelete.ID))

	if err != nil {
		return nil, err
	}

	return gen.TenantMemberDelete204JSONResponse{}, nil
}
