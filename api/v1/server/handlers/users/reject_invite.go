package users

import (
	"errors"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (u *UserService) TenantInviteReject(ctx echo.Context, request gen.TenantInviteRejectRequestObject) (gen.TenantInviteRejectResponseObject, error) {
	populator := populator.FromContext(ctx)

	user, err := populator.GetUser()
	if err != nil {
		return nil, err
	}
	userId := sqlchelpers.UUIDToStr(user.ID)

	// validate the request
	if apiErrors, err := u.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.TenantInviteReject400JSONResponse(*apiErrors), nil
	}

	inviteId := request.Body.Invite

	if inviteId == "" {
		return nil, errors.New("invalid invite id")
	}

	// get the invite
	invite, err := u.config.APIRepository.TenantInvite().GetTenantInvite(ctx.Request().Context(), inviteId)

	if err != nil {
		return nil, err
	}

	// ensure the invite belongs to the user
	if invite.InviteeEmail != user.Email {
		return gen.TenantInviteReject400JSONResponse(apierrors.NewAPIErrors("wrong email for invite")), nil
	}

	// ensure the invite is not expired
	if invite.Expires.Time.Before(time.Now()) {
		return gen.TenantInviteReject400JSONResponse(apierrors.NewAPIErrors("invite is expired")), nil
	}

	// ensure invite is in a pending state
	if invite.Status != dbsqlc.InviteLinkStatusPENDING {
		return gen.TenantInviteReject400JSONResponse(apierrors.NewAPIErrors("invite has already been used")), nil
	}

	// construct the database query
	updateOpts := &repository.UpdateTenantInviteOpts{
		Status: repository.StringPtr(string(dbsqlc.InviteLinkStatusREJECTED)),
	}

	// update the invite
	invite, err = u.config.APIRepository.TenantInvite().UpdateTenantInvite(ctx.Request().Context(), sqlchelpers.UUIDToStr(invite.ID), updateOpts)

	if err != nil {
		return nil, err
	}

	tenantId := sqlchelpers.UUIDToStr(invite.TenantId)

	u.config.Analytics.Enqueue(
		"user-invite:reject",
		userId,
		&tenantId,
		nil,
	)

	return nil, nil
}
