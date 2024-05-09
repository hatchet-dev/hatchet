package alerting

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hatchet-dev/hatchet/internal/integrations/alerting/alerttypes"
	"github.com/hatchet-dev/hatchet/internal/integrations/email"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
)

func (t *TenantAlertManager) sendEmailAlert(tenant *dbsqlc.Tenant, emailGroup *dbsqlc.TenantAlertEmailGroup, numFailed int, failedRuns []alerttypes.WorkflowRunFailedItem) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	subject := fmt.Sprintf("%d Hatchet workflows failed", numFailed)

	if numFailed <= 1 {
		subject = fmt.Sprintf("%d Hatchet workflow failed", numFailed)
	}

	emails := strings.Split(emailGroup.Emails, ",")
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	return t.email.SendWorkflowRunFailedAlerts(
		ctx,
		emails,
		email.WorkflowRunsFailedEmailData{
			TenantName:   tenant.Name,
			Items:        failedRuns,
			Subject:      subject,
			Summary:      subject,
			SettingsLink: fmt.Sprintf("%s/tenant-settings/alerting?tenant=%s", t.serverURL, tenantId),
		},
	)
}
