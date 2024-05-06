package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func ToTenant(tenant *db.TenantModel) *gen.Tenant {
	return &gen.Tenant{
		Metadata:        *toAPIMetadata(tenant.ID, tenant.CreatedAt, tenant.UpdatedAt),
		Name:            tenant.Name,
		Slug:            tenant.Slug,
		AnalyticsOptOut: &tenant.AnalyticsOptOut,
	}
}
