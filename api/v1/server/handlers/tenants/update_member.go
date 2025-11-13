package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantMemberUpdate(ctx echo.Context, request gen.TenantMemberUpdateRequestObject) (gen.TenantMemberUpdateResponseObject, error) {
	tenantMember := ctx.Get("tenant-member").(*dbsqlc.PopulateTenantMembersRow)
	memberToUpdate := ctx.Get("member").(*dbsqlc.PopulateTenantMembersRow)

	if apiErrors, err := t.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.TenantMemberUpdate400JSONResponse(*apiErrors), nil
	}

	// Check if the user has permission to update roles
	if tenantMember.Role == dbsqlc.TenantMemberRoleMEMBER {
		return gen.TenantMemberUpdate403JSONResponse(
			apierrors.NewAPIErrors("Only admins and owners can update member roles"),
		), nil
	}

	// if user is not an owner, they cannot change a role to owner or change owner roles
	if tenantMember.Role != dbsqlc.TenantMemberRoleOWNER {
		if request.Body.Role == gen.OWNER {
			return gen.TenantMemberUpdate400JSONResponse(
				apierrors.NewAPIErrors("only an owner can change a role to owner"),
			), nil
		}

		// Cannot change role of an owner
		if memberToUpdate.Role == dbsqlc.TenantMemberRoleOWNER {
			return gen.TenantMemberUpdate400JSONResponse(
				apierrors.NewAPIErrors("only an owner can change the role of another owner"),
			), nil
		}
	}

	// Users cannot change their own role
	if sqlchelpers.UUIDToStr(tenantMember.UserId) == sqlchelpers.UUIDToStr(memberToUpdate.UserId) {
		return gen.TenantMemberUpdate400JSONResponse(
			apierrors.NewAPIErrors("you cannot change your own role"),
		), nil
	}

	updateOpts := &repository.UpdateTenantMemberOpts{
		Role: repository.StringPtr(string(request.Body.Role)),
	}

	updatedMember, err := t.config.APIRepository.Tenant().UpdateTenantMember(ctx.Request().Context(), sqlchelpers.UUIDToStr(memberToUpdate.ID), updateOpts)

	if err != nil {
		return nil, err
	}

	return gen.TenantMemberUpdate200JSONResponse(
		*transformers.ToTenantMember(updatedMember),
	), nil
}
