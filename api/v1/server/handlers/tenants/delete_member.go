package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *TenantService) TenantMemberDelete(ctx echo.Context, request gen.TenantMemberDeleteRequestObject) (gen.TenantMemberDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	tenantMember := ctx.Get("tenant-member").(*db.TenantMemberModel)

	if tenantMember.Role != db.TenantMemberRoleOwner {
		return gen.TenantMemberDelete403JSONResponse(
			apierrors.NewAPIErrors("Only owners can delete members"),
		), nil
	}

	memberToDelete, err := t.config.APIRepository.Tenant().GetTenantMemberByID(request.Member.String())

	if err != nil {
		return nil, err
	}

	if tenantMember.UserID == memberToDelete.UserID {
		return gen.TenantMemberDelete403JSONResponse(
			apierrors.NewAPIErrors("You cannot delete yourself"),
		), nil
	}

	if memberToDelete.TenantID != tenant.ID {
		return gen.TenantMemberDelete404JSONResponse(
			apierrors.NewAPIErrors("Member not found"),
		), nil
	}

	_, err = t.config.APIRepository.Tenant().DeleteTenantMember(memberToDelete.ID)

	if err != nil {
		return nil, err
	}

	return gen.TenantMemberDelete204JSONResponse{}, nil
}
