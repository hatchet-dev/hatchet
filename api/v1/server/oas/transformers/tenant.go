package transformers

import (
	"strings"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func ToTenant(tenant *db.TenantModel) *gen.Tenant {
	return &gen.Tenant{
		Metadata:          *toAPIMetadata(tenant.ID, tenant.CreatedAt, tenant.UpdatedAt),
		Name:              tenant.Name,
		Slug:              tenant.Slug,
		AnalyticsOptOut:   &tenant.AnalyticsOptOut,
		AlertMemberEmails: &tenant.AlertMemberEmails,
	}
}

func ToTenantSqlc(tenant *dbsqlc.Tenant) *gen.Tenant {
	return &gen.Tenant{
		Metadata:          *toAPIMetadata(sqlchelpers.UUIDToStr(tenant.ID), tenant.CreatedAt.Time, tenant.UpdatedAt.Time),
		Name:              tenant.Name,
		Slug:              tenant.Slug,
		AnalyticsOptOut:   &tenant.AnalyticsOptOut,
		AlertMemberEmails: &tenant.AlertMemberEmails,
	}
}

func ToTenantAlertingSettings(alerting *db.TenantAlertingSettingsModel) *gen.TenantAlertingSettings {
	res := &gen.TenantAlertingSettings{
		Metadata:                        *toAPIMetadata(alerting.ID, alerting.CreatedAt, alerting.UpdatedAt),
		MaxAlertingFrequency:            alerting.MaxFrequency,
		EnableExpiringTokenAlerts:       &alerting.EnableExpiringTokenAlerts,
		EnableWorkflowRunFailureAlerts:  &alerting.EnableWorkflowRunFailureAlerts,
		EnableTenantResourceLimitAlerts: &alerting.EnableTenantResourceLimitAlerts,
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

func ToTenantResourcePolicy(_limits []*dbsqlc.TenantResourceLimit) *gen.TenantResourcePolicy {

	limits := make([]gen.TenantResourceLimit, len(_limits))

	for i, limit := range _limits {

		var alarmValue int
		if limit.AlarmValue.Valid {
			alarmValue = int(limit.AlarmValue.Int32)
		}

		var window string
		if limit.Window.Valid {
			window = limit.Window.String
		}

		limits[i] = gen.TenantResourceLimit{
			Metadata:   *toAPIMetadata(sqlchelpers.UUIDToStr(limit.ID), limit.CreatedAt.Time, limit.UpdatedAt.Time),
			Resource:   gen.TenantResource(limit.Resource),
			LimitValue: int(limit.LimitValue),
			AlarmValue: &alarmValue,
			Value:      int(limit.Value),
			Window:     &window,
			LastRefill: &limit.LastRefill.Time,
		}
	}

	return &gen.TenantResourcePolicy{
		Limits: limits,
	}
}
