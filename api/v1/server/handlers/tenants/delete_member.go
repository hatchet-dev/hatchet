package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/constants"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) TenantMemberDelete(ctx echo.Context, request gen.TenantMemberDeleteRequestObject) (gen.TenantMemberDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID.String()
	tenantMember := ctx.Get("tenant-member").(*sqlcv1.PopulateTenantMembersRow)
	memberToDelete := ctx.Get("member").(*sqlcv1.PopulateTenantMembersRow)

	if tenantMember.Role != sqlcv1.TenantMemberRoleOWNER {
		return gen.TenantMemberDelete403JSONResponse(
			apierrors.NewAPIErrors("Only owners can delete members"),
		), nil
	}

	if tenantMember.UserId.String() == memberToDelete.UserId.String() {
		return gen.TenantMemberDelete403JSONResponse(
			apierrors.NewAPIErrors("You cannot delete yourself"),
		), nil
	}

	if memberToDelete.TenantId.String() != tenantId {
		return gen.TenantMemberDelete404JSONResponse(
			apierrors.NewAPIErrors("Member not found"),
		), nil
	}

	err := t.config.V1.Tenant().DeleteTenantMember(ctx.Request().Context(), memberToDelete.ID.String())

	if err != nil {
		return nil, err
	}

	ctx.Set(constants.ResourceIdKey.String(), memberToDelete.ID.String())
	ctx.Set(constants.ResourceTypeKey.String(), constants.ResourceTypeTenantMember.String())

	return gen.TenantMemberDelete204JSONResponse{}, nil
}
