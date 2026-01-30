package alerting

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/hatchet-dev/hatchet/internal/integrations/alerting/alerttypes"
	"github.com/hatchet-dev/hatchet/pkg/integrations/email"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantAlertManager) sendEmailWorkflowRunAlert(tenant *sqlcv1.Tenant, emailGroup *v1.TenantAlertEmailGroupForSend, numFailed int, failedRuns []alerttypes.WorkflowRunFailedItem) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	subject := fmt.Sprintf("%d Hatchet workflows failed", numFailed)

	if numFailed <= 1 {
		subject = fmt.Sprintf("%d Hatchet workflow failed", numFailed)
	}

	tenantId := tenant.ID.String()

	return t.email.SendWorkflowRunFailedAlerts(
		ctx,
		emailGroup.Emails,
		email.WorkflowRunsFailedEmailData{
			TenantName:   tenant.Name,
			Items:        failedRuns,
			Subject:      subject,
			Summary:      subject,
			SettingsLink: fmt.Sprintf("%s/tenants/%s/tenant-settings/alerting", t.serverURL, tenantId),
		},
	)
}

func (t *TenantAlertManager) sendEmailExpiringTokenAlert(tenant *sqlcv1.Tenant, emailGroup *v1.TenantAlertEmailGroupForSend, payload *alerttypes.ExpiringTokenItem) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	subject := fmt.Sprintf("Hatchet token expiring %s", payload.ExpiresAtRelativeDate)

	tenantId := tenant.ID.String()

	return t.email.SendExpiringTokenEmail(
		ctx,
		emailGroup.Emails,
		email.ExpiringTokenEmailData{
			TenantName:            tenant.Name,
			TokenName:             payload.TokenName,
			ExpiresAtAbsoluteDate: payload.ExpiresAtAbsoluteDate,
			ExpiresAtRelativeDate: payload.ExpiresAtRelativeDate,
			Subject:               subject,
			TokenSettings:         payload.Link,
			SettingsLink:          fmt.Sprintf("%s/tenants/%s/tenant-settings/alerting", t.serverURL, tenantId),
		},
	)
}

func (t *TenantAlertManager) sendEmailTenantResourceLimitAlert(tenant *sqlcv1.Tenant, emailGroup *v1.TenantAlertEmailGroupForSend, payload *alerttypes.ResourceLimitAlert) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var subject string
	var summary string
	var summary2 string

	resource := strings.ReplaceAll(strings.ToLower(payload.Resource), "_", " ")
	resource = cases.Title(language.English).String(resource)

	if payload.AlertType == string(sqlcv1.TenantResourceLimitAlertTypeAlarm) {
		subject = fmt.Sprintf("%s has exhausted %d%% of its %s limit (%d/%d)", resource, payload.Percentage, payload.LimitWindow, payload.CurrentValue, payload.LimitValue)
		summary = "We're sending you this alert because a resource on your Hatchet tenant is approaching its usage limit."
		summary2 = fmt.Sprintf("Once the %s limit is reached, any further resource usage will be denied. Last refilled %s.", payload.LimitWindow, payload.LastRefillAgo)
	}

	if payload.AlertType == string(sqlcv1.TenantResourceLimitAlertTypeExhausted) {
		subject = fmt.Sprintf("%s has exhausted 100%% of its %s limit (%d/%d)", payload.Resource, payload.LimitWindow, payload.CurrentValue, payload.LimitValue)
		summary = "We're sending you this alert because a resource on your Hatchet tenant has exhausted its usage limit."
		summary2 = fmt.Sprintf("Any further resource usage will be denied until the limit is increased or its refill window is reached. Last refilled %s.", payload.LastRefillAgo)
	}

	return t.email.SendTenantResourceLimitAlert(
		ctx,
		emailGroup.Emails,
		email.ResourceLimitAlertData{
			TenantName:   tenant.Name,
			Subject:      subject,
			Summary:      summary,
			Summary2:     summary2,
			Resource:     payload.Resource,
			AlertType:    payload.AlertType,
			CurrentValue: payload.CurrentValue,
			LimitValue:   payload.LimitValue,
			Percentage:   payload.Percentage,
			Link:         payload.Link,
			SettingsLink: fmt.Sprintf("%s/tenants/%s/tenant-settings/alerting", t.serverURL, tenant.ID.String()),
		},
	)
}
