package billing

import "github.com/hatchet-dev/hatchet/internal/repository/prisma/db"

type Billing interface {
	UpsertTenant(tenant db.TenantModel) error
}

type NoOpBilling struct{}

func (a NoOpBilling) UpsertTenant(tenant db.TenantModel) error {
	return nil
}
