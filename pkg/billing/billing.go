package billing

import (
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type CustomerOpts struct {
	Email string
}

type SubscriptionOpts struct {
	PlanCode string
}

type Billing interface {
	Enabled() bool
	UpsertTenantSubscription(tenant db.TenantModel, opts *SubscriptionOpts) (*dbsqlc.TenantSubscription, error)
	MeterMetric(tenantId string, resource dbsqlc.LimitResource, uniqueId string, limitVal *int32) error
}

type NoOpBilling struct{}

func (a NoOpBilling) Enabled() bool {
	return false
}

func (a NoOpBilling) UpsertTenantSubscription(tenant db.TenantModel, opts *SubscriptionOpts) (*dbsqlc.TenantSubscription, error) {
	return nil, nil
}

func (a NoOpBilling) MeterMetric(tenantId string, resource dbsqlc.LimitResource, uniqueId string, limitVal *int32) error {
	return nil
}
