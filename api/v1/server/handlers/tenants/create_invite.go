package tenants

import (
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (t *TenantService) TenantInviteCreate(ctx echo.Context, request gen.TenantInviteCreateRequestObject) (gen.TenantInviteCreateResponseObject, error) {
	user := ctx.Get("user").(*db.UserModel)
	tenant := ctx.Get("tenant").(*db.TenantModel)
	tenantMember := ctx.Get("tenant-member").(*db.TenantMemberModel)

	// validate the request
	if apiErrors, err := t.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.TenantInviteCreate400JSONResponse(*apiErrors), nil
	}

	// ensure that this user isn't already a member of the tenant
	if _, err := t.config.APIRepository.Tenant().GetTenantMemberByEmail(tenant.ID, request.Body.Email); err == nil {
		return gen.TenantInviteCreate400JSONResponse(
			apierrors.NewAPIErrors("this user is already a member of this tenant"),
		), nil
	}

	// if user is not an owner, they cannot change a role to owner
	if tenantMember.Role != db.TenantMemberRoleOwner && request.Body.Role == gen.OWNER {
		return gen.TenantInviteCreate400JSONResponse(
			apierrors.NewAPIErrors("only an owner can change a role to owner"),
		), nil
	}

	// construct the database query
	createOpts := &repository.CreateTenantInviteOpts{
		InviteeEmail: request.Body.Email,
		InviterEmail: user.Email,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour), // 1 week expiration
		Role:         string(request.Body.Role),
	}

	// create the invite
	invite, err := t.config.APIRepository.TenantInvite().CreateTenantInvite(tenant.ID, createOpts)

	if err != nil {
		return nil, err
	}

	return gen.TenantInviteCreate201JSONResponse(
		*transformers.ToTenantInviteLink(invite),
	), nil
}
