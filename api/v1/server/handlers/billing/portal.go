package billing

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (b *BillingService) BillingPortalLinkGet(ctx echo.Context, req gen.BillingPortalLinkGetRequestObject) (gen.BillingPortalLinkGetResponseObject, error) {

	if !b.config.Billing.Enabled() {
		return gen.BillingPortalLinkGet200JSONResponse{}, nil
	}

	tenant := ctx.Get("tenant").(*db.TenantModel)
	tenantMember := ctx.Get("tenant-member").(*db.TenantMemberModel)

	if tenantMember.Role != db.TenantMemberRoleOwner {
		return gen.BillingPortalLinkGet403JSONResponse(
			apierrors.NewAPIErrors("only tenant owners can get the billing portal link"),
		), nil
	}

	link, err := b.config.Billing.GetCheckoutLink(*tenant)

	if err != nil {
		return nil, fmt.Errorf("getting billing portal link: %v", err)
	}

	return gen.BillingPortalLinkGetResponseObject(
		&gen.BillingPortalLinkGet200JSONResponse{
			Url: link,
		},
	), nil
}
