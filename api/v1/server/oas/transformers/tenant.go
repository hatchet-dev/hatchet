package transformers

import (
	"strings"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/billing"
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

func ToTenantResourcePolicy(_limits []*dbsqlc.TenantResourceLimit, _sub *dbsqlc.TenantSubscription, checkoutLink *string, _methods []*billing.PaymentMethod, _plans []*billing.Plan) *gen.TenantResourcePolicy {

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

	var subscription = gen.TenantSubscription{
		Plan:   nil,
		Period: nil,
		Status: nil,
		Note:   nil,
	}

	if _sub != nil {
		status := gen.TenantSubscriptionStatus(_sub.Status)

		plan := string(_sub.Plan)

		var period *string

		if _sub.Period.Valid {
			_period := string(_sub.Period.TenantSubscriptionPeriod)
			period = &_period
		}

		var note string
		if _sub.Note.Valid {
			note = _sub.Note.String
		}

		subscription = gen.TenantSubscription{
			Plan:   &plan,
			Period: period,
			Status: &status,
			Note:   &note,
		}
	}

	methods := func() []gen.TenantPaymentMethod {
		res := make([]gen.TenantPaymentMethod, len(_methods))

		for i, method := range _methods {
			res[i] = gen.TenantPaymentMethod{
				Last4:      method.Last4,
				Brand:      string(method.Brand),
				Expiration: method.Expiration,
			}
		}

		return res
	}()

	plans := func() []gen.SubscriptionPlan {
		res := make([]gen.SubscriptionPlan, len(_plans))

		for i, plan := range _plans {
			res[i] = gen.SubscriptionPlan{
				PlanCode:    plan.PlanCode,
				Name:        plan.Name,
				Description: plan.Description,
				AmountCents: plan.AmountCents,
				Period:      plan.Period,
			}
		}

		return res
	}()

	return &gen.TenantResourcePolicy{
		Limits:         limits,
		Subscription:   subscription,
		CheckoutLink:   checkoutLink,
		PaymentMethods: &methods,
		Plans:          &plans,
	}
}
