package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantInviteUpdate(ctx echo.Context, request gen.TenantInviteUpdateRequestObject) (gen.TenantInviteUpdateResponseObject, error) {
	populator := populator.FromContext(ctx)

	tenantMember, err := populator.GetTenantMember()
	if err != nil {
		return nil, err
	}
	invite, err := populator.GetTenantInvite()
	if err != nil {
		return nil, err
	}
	// validate the request
	if apiErrors, err := t.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.TenantInviteUpdate400JSONResponse(*apiErrors), nil
	}

	// if user is not an owner, they cannot change a role to owner
	if tenantMember.Role != dbsqlc.TenantMemberRoleOWNER && request.Body.Role == gen.OWNER {
		return gen.TenantInviteUpdate400JSONResponse(
			apierrors.NewAPIErrors("only an owner can change a role to owner"),
		), nil
	}

	// construct the database query
	updateOpts := &repository.UpdateTenantInviteOpts{
		Role: repository.StringPtr(string(request.Body.Role)),
	}

	// update the invite
	invite, err = t.config.APIRepository.TenantInvite().UpdateTenantInvite(ctx.Request().Context(), sqlchelpers.UUIDToStr(invite.ID), updateOpts)

	if err != nil {
		return nil, err
	}

	return gen.TenantInviteUpdate200JSONResponse(
		*transformers.ToTenantInviteLink(invite),
	), nil
}
