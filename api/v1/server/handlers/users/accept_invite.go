package users

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/constants"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (u *UserService) TenantInviteAccept(ctx echo.Context, request gen.TenantInviteAcceptRequestObject) (gen.TenantInviteAcceptResponseObject, error) {
	user := ctx.Get("user").(*sqlcv1.User)
	userId := user.ID

	// validate the request
	if apiErrors, err := u.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.TenantInviteAccept400JSONResponse(*apiErrors), nil
	}

	inviteIdStr := request.Body.Invite

	if inviteIdStr == "" {
		return nil, errors.New("invalid invite id")
	}

	inviteId, err := uuid.Parse(inviteIdStr)

	if err != nil {
		return nil, errors.New("invalid invite id")
	}

	// get the invite
	invite, err := u.config.V1.TenantInvite().GetTenantInvite(ctx.Request().Context(), inviteId)

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
	if invite.Status != sqlcv1.InviteLinkStatusPENDING {
		return gen.TenantInviteAccept400JSONResponse(apierrors.NewAPIErrors("invite has already been used")), nil
	}

	// ensure the user is not already a member of the tenant
	_, err = u.config.V1.Tenant().GetTenantMemberByEmail(ctx.Request().Context(), invite.TenantId, user.Email)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	} else if err == nil {
		return gen.TenantInviteAccept400JSONResponse(apierrors.NewAPIErrors("user is already a member of the tenant")), nil
	}

	// construct the database query
	updateOpts := &v1.UpdateTenantInviteOpts{
		Status: v1.StringPtr(string(sqlcv1.InviteLinkStatusACCEPTED)),
	}

	// update the invite
	invite, err = u.config.V1.TenantInvite().UpdateTenantInvite(ctx.Request().Context(), invite.ID, updateOpts)

	if err != nil {
		return nil, err
	}

	// add the user to the tenant
	member, err := u.config.V1.Tenant().CreateTenantMember(ctx.Request().Context(), invite.TenantId, &v1.CreateTenantMemberOpts{
		UserId: userId,
		Role:   string(invite.Role),
	})

	if err != nil {
		return nil, err
	}

	u.config.Analytics.Enqueue(
		"user-invite:accept",
		userId.String(),
		&invite.TenantId,
		nil,
		map[string]interface{}{
			"user_id":   userId.String(),
			"invite_id": inviteId.String(),
			"role":      invite.Role,
		},
	)

	ctx.Set("tenant-member", member)

	tenant, err := u.config.V1.Tenant().GetTenantByID(ctx.Request().Context(), invite.TenantId)
	if err != nil {
		return nil, err
	}

	ctx.Set("tenant", tenant)

	ctx.Set(constants.ResourceIdKey.String(), inviteId.String())
	ctx.Set(constants.ResourceTypeKey.String(), constants.ResourceTypeTenantInvite.String())

	return nil, nil
}
