package transformers

import (
	"strings"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func ToTenant(tenant *dbsqlc.Tenant) *gen.Tenant {
	return &gen.Tenant{
		Metadata:          *toAPIMetadata(sqlchelpers.UUIDToStr(tenant.ID), tenant.CreatedAt.Time, tenant.UpdatedAt.Time),
		Name:              tenant.Name,
		Slug:              tenant.Slug,
		AnalyticsOptOut:   &tenant.AnalyticsOptOut,
		AlertMemberEmails: &tenant.AlertMemberEmails,
	}
}

func ToTenantAlertingSettings(alerting *dbsqlc.TenantAlertingSettings) *gen.TenantAlertingSettings {
	res := &gen.TenantAlertingSettings{
		Metadata:                        *toAPIMetadata(sqlchelpers.UUIDToStr(alerting.ID), alerting.CreatedAt.Time, alerting.UpdatedAt.Time),
		MaxAlertingFrequency:            alerting.MaxFrequency,
		EnableExpiringTokenAlerts:       &alerting.EnableExpiringTokenAlerts,
		EnableWorkflowRunFailureAlerts:  &alerting.EnableWorkflowRunFailureAlerts,
		EnableTenantResourceLimitAlerts: &alerting.EnableTenantResourceLimitAlerts,
	}

	if alerting.LastAlertedAt.Valid {
		res.LastAlertedAt = &alerting.LastAlertedAt.Time
	}

	return res
}

func ToTenantAlertEmailGroup(group *dbsqlc.TenantAlertEmailGroup) *gen.TenantAlertEmailGroup {
	emails := strings.Split(group.Emails, ",")

	return &gen.TenantAlertEmailGroup{
		Metadata: *toAPIMetadata(sqlchelpers.UUIDToStr(group.ID), group.CreatedAt.Time, group.UpdatedAt.Time),
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
