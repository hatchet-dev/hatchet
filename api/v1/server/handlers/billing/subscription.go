package billing

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/billing"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

func (b *BillingService) SubscriptionUpsert(ctx echo.Context, req gen.SubscriptionUpsertRequestObject) (gen.SubscriptionUpsertResponseObject, error) {

	if !b.config.Billing.Enabled() {
		return gen.SubscriptionUpsert200JSONResponse{}, nil
	}

	tenant := ctx.Get("tenant").(*db.TenantModel)
	tenantMember := ctx.Get("tenant-member").(*db.TenantMemberModel)

	if tenantMember.Role != db.TenantMemberRoleOwner {
		return gen.SubscriptionUpsert403JSONResponse(
			apierrors.NewAPIErrors("only tenant owners can update the subscription"),
		), nil
	}

	_, err := b.config.Billing.UpsertTenantSubscription(*tenant, billing.SubscriptionOpts{
		Plan:   dbsqlc.TenantSubscriptionPlanCodes(*req.Body.Plan),
		Period: req.Body.Period,
	})

	if err != nil {
		return nil, fmt.Errorf("error updating subscription: %v", err)
	}

	return gen.SubscriptionUpsert200JSONResponse{}, nil
}
