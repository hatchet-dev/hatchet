package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *TenantService) TenantInviteDelete(ctx echo.Context, request gen.TenantInviteDeleteRequestObject) (gen.TenantInviteDeleteResponseObject, error) {
	invite := ctx.Get("tenant-invite").(*db.TenantInviteLinkModel)

	// delete the invite
	err := t.config.APIRepository.TenantInvite().DeleteTenantInvite(invite.ID)

	if err != nil {
		return nil, err
	}

	return gen.TenantInviteDelete200JSONResponse(
		*transformers.ToTenantInviteLink(invite),
	), nil
}
