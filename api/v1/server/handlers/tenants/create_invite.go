package tenants

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/integrations/email"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantInviteCreate(ctx echo.Context, request gen.TenantInviteCreateRequestObject) (gen.TenantInviteCreateResponseObject, error) {
	user := ctx.Get("user").(*dbsqlc.User)
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	tenantMember := ctx.Get("tenant-member").(*dbsqlc.PopulateTenantMembersRow)
	if !t.config.Runtime.AllowInvites {
		t.config.Logger.Warn().Msg("tenant invites are disabled")
		return gen.TenantInviteCreate400JSONResponse(
			apierrors.NewAPIErrors("tenant invites are disabled"),
		), nil
	}

	// validate the request
	if apiErrors, err := t.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		t.config.Logger.Warn().Msg("invalid request")
		return gen.TenantInviteCreate400JSONResponse(*apiErrors), nil
	}

	// ensure that this user isn't already a member of the tenant
	if _, err := t.config.APIRepository.Tenant().GetTenantMemberByEmail(ctx.Request().Context(), tenantId, request.Body.Email); err == nil {
		t.config.Logger.Warn().Msg("this user is already a member of this tenant")
		return gen.TenantInviteCreate400JSONResponse(
			apierrors.NewAPIErrors("this user is already a member of this tenant"),
		), nil
	}

	// if user is not an owner, they cannot change a role to owner
	if tenantMember.Role != dbsqlc.TenantMemberRoleOWNER && request.Body.Role == gen.OWNER {
		t.config.Logger.Warn().Msg("only an owner can change a role to owner")
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
		MaxPending:   t.config.Runtime.MaxPendingInvites,
	}

	// create the invite
	invite, err := t.config.APIRepository.TenantInvite().CreateTenantInvite(ctx.Request().Context(), tenantId, createOpts)

	if err != nil {
		t.config.Logger.Err(err).Msg("could not create tenant invite")

		return gen.TenantInviteCreate403JSONResponse{
			Description: err.Error(),
		}, nil
	}

	// send an email
	go func() {
		emailCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		name := user.Email

		if user.Name.Valid {
			name = user.Name.String
		}

		if err := t.config.Email.SendTenantInviteEmail(emailCtx, invite.InviteeEmail, email.TenantInviteEmailData{
			InviteSenderName: name,
			TenantName:       tenant.Name,
			ActionURL:        t.config.Runtime.ServerURL,
		}); err != nil {
			t.config.Logger.Err(err).Msg("could not send tenant invite email")
		}
	}()

	t.config.Analytics.Enqueue("user-invite:create",
		sqlchelpers.UUIDToStr(user.ID),
		&tenantId,
		nil,
	)

	return gen.TenantInviteCreate201JSONResponse(
		*transformers.ToTenantInviteLink(invite),
	), nil
}
