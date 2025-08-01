package users

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/constants"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (u *UserService) TenantInviteAccept(ctx echo.Context, request gen.TenantInviteAcceptRequestObject) (gen.TenantInviteAcceptResponseObject, error) {
	user := ctx.Get("user").(*dbsqlc.User)
	userId := sqlchelpers.UUIDToStr(user.ID)

	// validate the request
	if apiErrors, err := u.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.TenantInviteAccept400JSONResponse(*apiErrors), nil
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
		return gen.TenantInviteAccept400JSONResponse(apierrors.NewAPIErrors("wrong email for invite")), nil
	}

	// ensure the invite is not expired
	if invite.Expires.Time.Before(time.Now()) {
		return gen.TenantInviteAccept400JSONResponse(apierrors.NewAPIErrors("invite is expired")), nil
	}

	// ensure invite is in a pending state
	if invite.Status != dbsqlc.InviteLinkStatusPENDING {
		return gen.TenantInviteAccept400JSONResponse(apierrors.NewAPIErrors("invite has already been used")), nil
	}

	// ensure the user is not already a member of the tenant
	_, err = u.config.APIRepository.Tenant().GetTenantMemberByEmail(ctx.Request().Context(), sqlchelpers.UUIDToStr(invite.TenantId), user.Email)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	} else if err == nil {
		return gen.TenantInviteAccept400JSONResponse(apierrors.NewAPIErrors("user is already a member of the tenant")), nil
	}

	// construct the database query
	updateOpts := &repository.UpdateTenantInviteOpts{
		Status: repository.StringPtr(string(dbsqlc.InviteLinkStatusACCEPTED)),
	}

	// update the invite
	invite, err = u.config.APIRepository.TenantInvite().UpdateTenantInvite(ctx.Request().Context(), sqlchelpers.UUIDToStr(invite.ID), updateOpts)

	if err != nil {
		return nil, err
	}

	// add the user to the tenant
	_, err = u.config.APIRepository.Tenant().CreateTenantMember(ctx.Request().Context(), sqlchelpers.UUIDToStr(invite.TenantId), &repository.CreateTenantMemberOpts{
		UserId: userId,
		Role:   string(invite.Role),
	})

	if err != nil {
		return nil, err
	}

	tenantId := sqlchelpers.UUIDToStr(invite.TenantId)

	u.config.Analytics.Enqueue(
		"user-invite:accept",
		userId,
		&tenantId,
		nil,
	)

	ctx.Set(constants.ResourceIdKey.String(), inviteId)
	ctx.Set(constants.ResourceTypeKey.String(), constants.ResourceTypeTenantInvite.String())

	return nil, nil
}
