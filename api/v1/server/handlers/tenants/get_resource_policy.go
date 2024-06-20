package tenants

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/billing"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

func (t *TenantService) TenantResourcePolicyGet(ctx echo.Context, request gen.TenantResourcePolicyGetRequestObject) (gen.TenantResourcePolicyGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	var subscription *dbsqlc.TenantSubscription
	var err error
	var methods []*billing.PaymentMethod

	if t.config.Billing.Enabled() {
		// subscription, err = t.config.EntitlementRepository.TenantSubscription().GetSubscription(context.Background(), tenant.ID)

		// _, err := t.config.Billing.UpsertTenantSubscription(*tenant, billing.SubscriptionOpts{
		// 	Plan:   dbsqlc.TenantSubscriptionPlanCodesFree,
		// 	Period: nil,
		// })

		methods, err = t.config.Billing.GetPaymentMethods(tenant.ID)

		if err != nil {
			return nil, fmt.Errorf("failed to get customer: %w", err)
		}

		subscription, err = t.config.Billing.GetSubscription(tenant.ID)

		if err != nil {
			return nil, fmt.Errorf("failed to get subscription: %w", err)
		}
	}

	limits, err := t.config.EntitlementRepository.TenantLimit().GetLimits(context.Background(), tenant.ID)

	if err != nil {
		return nil, err
	}

	checkoutLink, err := t.config.Billing.GetCheckoutLink(tenant.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to get checkout link: %w", err)
	}

	plans, err := t.config.Billing.Plans()

	if err != nil {
		return nil, fmt.Errorf("failed to get plans: %w", err)
	}

	return gen.TenantResourcePolicyGet200JSONResponse(
		*transformers.ToTenantResourcePolicy(limits, subscription, checkoutLink, methods, plans),
	), nil
}
