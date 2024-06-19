package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type TenantSubscriptionConfig struct {
	EnforceLimits bool
}

type TenantSubscriptionRepository interface {
	// GetSubscription returns the subscription for a tenant
	GetSubscription(ctx context.Context, tenantId string) (*dbsqlc.TenantSubscription, error)

	// UpsertSubscription creates or updates a subscription for a tenant
	UpsertSubscription(ctx context.Context, opts dbsqlc.UpsertTenantSubscriptionParams) (bool, *dbsqlc.TenantSubscription, error)
}
