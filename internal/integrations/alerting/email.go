package alerting

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/integrations/alerting/alerttypes"
	"github.com/hatchet-dev/hatchet/internal/integrations/email"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
)

func (t *TenantAlertManager) sendEmailWorkflowRunAlert(tenant *dbsqlc.Tenant, emailGroup *repository.TenantAlertEmailGroupForSend, numFailed int, failedRuns []alerttypes.WorkflowRunFailedItem) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	subject := fmt.Sprintf("%d Hatchet workflows failed", numFailed)

	if numFailed <= 1 {
		subject = fmt.Sprintf("%d Hatchet workflow failed", numFailed)
	}

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	return t.email.SendWorkflowRunFailedAlerts(
		ctx,
		emailGroup.Emails,
		email.WorkflowRunsFailedEmailData{
			TenantName:   tenant.Name,
			Items:        failedRuns,
			Subject:      subject,
			Summary:      subject,
			SettingsLink: fmt.Sprintf("%s/tenant-settings/alerting?tenant=%s", t.serverURL, tenantId),
		},
	)
}

func (t *TenantAlertManager) sendEmailExpiringTokenAlert(tenant *dbsqlc.Tenant, emailGroup *repository.TenantAlertEmailGroupForSend, payload *alerttypes.ExpiringTokenItem) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	subject := fmt.Sprintf("Hatchet token expiring %s", payload.ExpiresAtRelativeDate)

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

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
			SettingsLink:          fmt.Sprintf("%s/tenant-settings/alerting?tenant=%s", t.serverURL, tenantId),
		},
	)
}
