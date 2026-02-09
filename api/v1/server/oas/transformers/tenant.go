package transformers

import (
	"strings"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func ToTenant(tenant *sqlcv1.Tenant) *gen.Tenant {
	var environment *gen.TenantEnvironment
	if tenant.Environment.Valid {
		env := gen.TenantEnvironment(tenant.Environment.TenantEnvironment)
		environment = &env
	}

	return &gen.Tenant{
		Metadata:          *toAPIMetadata(tenant.ID, tenant.CreatedAt.Time, tenant.UpdatedAt.Time),
		Name:              tenant.Name,
		Slug:              tenant.Slug,
		AnalyticsOptOut:   &tenant.AnalyticsOptOut,
		AlertMemberEmails: &tenant.AlertMemberEmails,
		Version:           gen.TenantVersion(tenant.Version),
		Environment:       environment,
	}
}

func ToTenantAlertingSettings(alerting *sqlcv1.TenantAlertingSettings) *gen.TenantAlertingSettings {
	res := &gen.TenantAlertingSettings{
		Metadata:                        *toAPIMetadata(alerting.ID, alerting.CreatedAt.Time, alerting.UpdatedAt.Time),
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

func ToTenantAlertEmailGroup(group *sqlcv1.TenantAlertEmailGroup) *gen.TenantAlertEmailGroup {
	emails := strings.Split(group.Emails, ",")

	return &gen.TenantAlertEmailGroup{
		Metadata: *toAPIMetadata(group.ID, group.CreatedAt.Time, group.UpdatedAt.Time),
		Emails:   emails,
	}
}

func ToTenantResourcePolicy(_limits []*sqlcv1.TenantResourceLimit) *gen.TenantResourcePolicy {

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
			Metadata:   *toAPIMetadata(limit.ID, limit.CreatedAt.Time, limit.UpdatedAt.Time),
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

func ToTaskStats(stats map[string]v1.TaskStat) gen.TaskStats {
	result := make(gen.TaskStats)

	for taskName, taskStat := range stats {
		var queued *gen.TaskStatusStat
		var running *gen.TaskStatusStat

		if taskStat.Queued != nil {
			queued = toTaskStatusStat(*taskStat.Queued)
		}
		if taskStat.Running != nil {
			running = toTaskStatusStat(*taskStat.Running)
		}

		result[taskName] = gen.TaskStat{
			Queued:  queued,
			Running: running,
		}
	}

	return result
}

func toTaskStatusStat(stat v1.TaskStatusStat) *gen.TaskStatusStat {
	result := &gen.TaskStatusStat{
		Total:  &stat.Total,
		Oldest: stat.Oldest,
	}

	if len(stat.Concurrency) > 0 {
		concurrency := make([]gen.ConcurrencyStat, len(stat.Concurrency))
		for i, c := range stat.Concurrency {
			concurrency[i] = gen.ConcurrencyStat{
				Expression: &c.Expression,
				Type:       &c.Type,
				Keys:       &c.Keys,
			}
		}
		result.Concurrency = &concurrency
	}

	if len(stat.Queues) > 0 {
		result.Queues = &stat.Queues
	}

	return result
}
