package transformers

import (
	"strings"

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

func ToTenantAlertingSettings(alerting *db.TenantAlertingSettingsModel) *gen.TenantAlertingSettings {
	res := &gen.TenantAlertingSettings{
		Metadata:             *toAPIMetadata(alerting.ID, alerting.CreatedAt, alerting.UpdatedAt),
		MaxAlertingFrequency: alerting.MaxFrequency,
	}

	if lastAlertedAt, ok := alerting.LastAlertedAt(); ok {
		res.LastAlertedAt = &lastAlertedAt
	}

	return res
}

func ToTenantAlertEmailGroup(group *db.TenantAlertEmailGroupModel) *gen.TenantAlertEmailGroup {
	emails := strings.Split(group.Emails, ",")

	return &gen.TenantAlertEmailGroup{
		Metadata: *toAPIMetadata(group.ID, group.CreatedAt, group.UpdatedAt),
		Emails:   emails,
	}
}
