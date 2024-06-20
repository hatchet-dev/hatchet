package billing

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/billing"
)

func (b *BillingService) SubscriptionUpsert(ctx echo.Context, req gen.SubscriptionUpsertRequestObject) (gen.SubscriptionUpsertResponseObject, error) {

	if !b.config.Billing.Enabled() {
		return gen.SubscriptionUpsert200JSONResponse{}, nil
	}

	tenant := ctx.Get("tenant").(*db.TenantModel)

	_, err := b.config.Billing.UpsertTenantSubscription(*tenant, billing.SubscriptionOpts{
		Plan:   dbsqlc.TenantSubscriptionPlanCodes(*req.Body.Plan),
		Period: req.Body.Period,
	})

	if err != nil {
		return nil, fmt.Errorf("error updating subscription: %v", err)
	}

	return gen.SubscriptionUpsert200JSONResponse{}, nil
}
