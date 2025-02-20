package users

import (
	"errors"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (u *UserService) TenantInviteReject(ctx echo.Context, request gen.TenantInviteRejectRequestObject) (gen.TenantInviteRejectResponseObject, error) {
	user := ctx.Get("user").(*db.UserModel)

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
	invite, err := u.config.APIRepository.TenantInvite().GetTenantInvite(inviteId)

	if err != nil {
		return nil, err
	}

	// ensure the invite belongs to the user
	if invite.InviteeEmail != user.Email {
		return gen.TenantInviteReject400JSONResponse(apierrors.NewAPIErrors("wrong email for invite")), nil
	}

	// ensure the invite is not expired
	if invite.Expires.Before(time.Now()) {
		return gen.TenantInviteReject400JSONResponse(apierrors.NewAPIErrors("invite is expired")), nil
	}

	// ensure invite is in a pending state
	if invite.Status != db.InviteLinkStatusPending {
		return gen.TenantInviteReject400JSONResponse(apierrors.NewAPIErrors("invite has already been used")), nil
	}

	// construct the database query
	updateOpts := &repository.UpdateTenantInviteOpts{
		Status: repository.StringPtr(string(db.InviteLinkStatusRejected)),
	}

	// update the invite
	invite, err = u.config.APIRepository.TenantInvite().UpdateTenantInvite(invite.ID, updateOpts)

	if err != nil {
		return nil, err
	}

	u.config.Analytics.Enqueue(
		"user-invite:accept",
		user.ID,
		&invite.TenantID,
		nil,
	)

	return nil, nil
}
