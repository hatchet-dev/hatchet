package tenants

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/billing"
)

func (t *TenantService) TenantResourcePolicyGet(ctx echo.Context, request gen.TenantResourcePolicyGetRequestObject) (gen.TenantResourcePolicyGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	var subscription *dbsqlc.TenantSubscription
	var err error
	var methods []*billing.PaymentMethod

	if t.config.Billing.Enabled() {
		// subscription, err = t.config.EntitlementRepository.TenantSubscription().GetSubscription(context.Background(), tenant.ID)

		// customer, err := t.config.Billing.UpsertTenantSubscription()(billing.CustomerOpts{
		// 	Email: tenant.Email,
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

	return gen.TenantResourcePolicyGet200JSONResponse(
		*transformers.ToTenantResourcePolicy(limits, subscription, checkoutLink, methods),
	), nil
}
