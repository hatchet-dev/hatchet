package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantInviteDelete(ctx echo.Context, request gen.TenantInviteDeleteRequestObject) (gen.TenantInviteDeleteResponseObject, error) {
	populator := populator.FromContext(ctx)

	invite, err := populator.GetTenantInvite()
	if err != nil {
		return nil, err
	}

	// delete the invite
	err = t.config.APIRepository.TenantInvite().DeleteTenantInvite(ctx.Request().Context(), sqlchelpers.UUIDToStr(invite.ID))

	if err != nil {
		return nil, err
	}

	return gen.TenantInviteDelete200JSONResponse(
		*transformers.ToTenantInviteLink(invite),
	), nil
}
