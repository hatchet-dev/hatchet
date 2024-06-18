package billing

import "github.com/hatchet-dev/hatchet/internal/repository/prisma/db"

type Billing interface {
	Enabled() bool
	UpsertTenant(tenant db.TenantModel) error
	MeterMetric(tenantId string, resource string, uniqueId string, limitVal *int32) error
}

type NoOpBilling struct{}

func (a NoOpBilling) Enabled() bool {
	return false
}

func (a NoOpBilling) UpsertTenant(tenant db.TenantModel) error {
	return nil
}

func (a NoOpBilling) MeterMetric(tenantId string, resource string, uniqueId string, limitVal *int32) error {
	return nil
}
