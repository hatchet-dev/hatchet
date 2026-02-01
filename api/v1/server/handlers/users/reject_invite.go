package users

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (u *UserService) TenantInviteReject(ctx echo.Context, request gen.TenantInviteRejectRequestObject) (gen.TenantInviteRejectResponseObject, error) {
	user := ctx.Get("user").(*sqlcv1.User)
	userId := user.ID.String()

	// validate the request
	if apiErrors, err := u.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.TenantInviteReject400JSONResponse(*apiErrors), nil
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
		return gen.TenantInviteReject400JSONResponse(apierrors.NewAPIErrors("wrong email for invite")), nil
	}

	// ensure the invite is not expired
	if invite.Expires.Time.Before(time.Now()) {
		return gen.TenantInviteReject400JSONResponse(apierrors.NewAPIErrors("invite is expired")), nil
	}

	// ensure invite is in a pending state
	if invite.Status != sqlcv1.InviteLinkStatusPENDING {
		return gen.TenantInviteReject400JSONResponse(apierrors.NewAPIErrors("invite has already been used")), nil
	}

	// construct the database query
	updateOpts := &v1.UpdateTenantInviteOpts{
		Status: v1.StringPtr(string(sqlcv1.InviteLinkStatusREJECTED)),
	}

	// update the invite
	invite, err = u.config.V1.TenantInvite().UpdateTenantInvite(ctx.Request().Context(), invite.ID, updateOpts)

	if err != nil {
		return nil, err
	}

	u.config.Analytics.Enqueue(
		"user-invite:reject",
		userId,
		&invite.TenantId,
		nil,
		map[string]interface{}{
			"user_id":   userId,
			"invite_id": inviteId,
			"role":      invite.Role,
		},
	)

	return nil, nil
}
