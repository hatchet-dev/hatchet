package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *TenantService) TenantInviteUpdate(ctx echo.Context, request gen.TenantInviteUpdateRequestObject) (gen.TenantInviteUpdateResponseObject, error) {
	tenantMember := ctx.Get("tenant-member").(*db.TenantMemberModel)
	invite := ctx.Get("tenant-invite").(*db.TenantInviteLinkModel)

	// validate the request
	if apiErrors, err := t.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.TenantInviteUpdate400JSONResponse(*apiErrors), nil
	}

	// if user is not an owner, they cannot change a role to owner
	if tenantMember.Role != db.TenantMemberRoleOwner && request.Body.Role == gen.OWNER {
		return gen.TenantInviteUpdate400JSONResponse(
			apierrors.NewAPIErrors("only an owner can change a role to owner"),
		), nil
	}

	// construct the database query
	updateOpts := &repository.UpdateTenantInviteOpts{
		Role: repository.StringPtr(string(request.Body.Role)),
	}

	// update the invite
	invite, err := t.config.APIRepository.TenantInvite().UpdateTenantInvite(invite.ID, updateOpts)

	if err != nil {
		return nil, err
	}

	return gen.TenantInviteUpdate200JSONResponse(
		*transformers.ToTenantInviteLink(invite),
	), nil
}
