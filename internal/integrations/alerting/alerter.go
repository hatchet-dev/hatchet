package alerting

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"

	"github.com/hatchet-dev/hatchet/internal/integrations/alerting/alerttypes"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/integrations/email"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	"github.com/hatchet-dev/timediff"
)

type TenantAlertManager struct {
	repo      v1.Repository
	enc       encryption.EncryptionService
	email     email.EmailService
	serverURL string
}

func New(repo v1.Repository, e encryption.EncryptionService, serverURL string, email email.EmailService) *TenantAlertManager {
	return &TenantAlertManager{repo, e, serverURL, email}
}

func (t *TenantAlertManager) SendWorkflowRunAlertV1(tenantId uuid.UUID, failedRuns []*v1.WorkflowRunData) error {
	if len(failedRuns) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// read in the tenant alerting settings and determine if we should alert
	tenantAlerting, err := t.repo.TenantAlertingSettings().GetTenantAlertingSettings(ctx, tenantId)

	if err != nil {
		return fmt.Errorf("could not get tenant alerting settings: %w", err)
	}

	failedItems := t.getFailedItemsV1(failedRuns)

	if len(failedItems) == 0 {
		return nil
	}

	now := time.Now().UTC()

	err = t.repo.TenantAlertingSettings().UpdateTenantAlertingSettings(ctx, tenantId, &v1.UpdateTenantAlertingSettingsOpts{
		LastAlertedAt: &now,
	})

	if err != nil {
		return fmt.Errorf("could not update tenant alerting settings: %w", err)
	}

	// iterate through possible alerters
	for _, slackWebhook := range tenantAlerting.SlackWebhooks {
		if innerErr := t.sendSlackWorkflowRunAlert(slackWebhook, len(failedRuns), failedItems); innerErr != nil {
			err = multierror.Append(err, innerErr)
		}
	}

	for _, emailGroup := range tenantAlerting.EmailGroups {
		if innerErr := t.sendEmailWorkflowRunAlert(tenantAlerting.Tenant, emailGroup, len(failedRuns), failedItems); innerErr != nil {
			err = multierror.Append(err, innerErr)
		}
	}

	if err != nil {
		return fmt.Errorf("could not send tenant alert: %w", err)
	}

	return nil
}

func (t *TenantAlertManager) getFailedItemsV1(failedRuns []*v1.WorkflowRunData) []alerttypes.WorkflowRunFailedItem {
	res := make([]alerttypes.WorkflowRunFailedItem, 0)

	for i, workflowRun := range failedRuns {
		if i >= 5 {
			break
		}

		workflowRunId := workflowRun.ExternalID.String()
		tenantId := workflowRun.TenantID.String()

		readableId := workflowRun.DisplayName

		res = append(res, alerttypes.WorkflowRunFailedItem{
			Link:                  fmt.Sprintf("%s/tenants/%s/runs/%s", t.serverURL, tenantId, workflowRunId),
			WorkflowName:          readableId,
			WorkflowRunReadableId: readableId,
			RelativeDate:          timediff.TimeDiff(workflowRun.FinishedAt.Time),
			AbsoluteDate:          workflowRun.FinishedAt.Time.Format("2006-01-02 15:04:05"),
		})
	}

	return res
}

func (t *TenantAlertManager) SendExpiringTokenAlert(tenantId uuid.UUID, token *sqlcv1.PollExpiringTokensRow) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// read in the tenant alerting settings and determine if we should alert
	tenantAlerting, err := t.repo.TenantAlertingSettings().GetTenantAlertingSettings(ctx, tenantId)

	if err != nil {
		return err
	}

	payload := &alerttypes.ExpiringTokenItem{
		TokenName:             token.Name.String,
		ExpiresAtRelativeDate: timediff.TimeDiff(token.ExpiresAt.Time),
		ExpiresAtAbsoluteDate: token.ExpiresAt.Time.Format("2006-01-02 15:04:05"),
		Link:                  fmt.Sprintf("%s/tenants/%s/tenant-settings/api-tokens", t.serverURL, tenantId),
	}

	return t.sendExpiringTokenAlert(ctx, tenantAlerting, payload)
}

func (t *TenantAlertManager) sendExpiringTokenAlert(ctx context.Context, tenantAlerting *v1.GetTenantAlertingSettingsResponse, payload *alerttypes.ExpiringTokenItem) error {

	if !tenantAlerting.Settings.EnableExpiringTokenAlerts {
		return nil
	}

	var err error

	// iterate through possible alerters
	for _, slackWebhook := range tenantAlerting.SlackWebhooks {
		if innerErr := t.sendSlackExpiringTokenAlert(slackWebhook, payload); innerErr != nil {
			err = multierror.Append(err, innerErr)
		}
	}

	for _, emailGroup := range tenantAlerting.EmailGroups {
		if innerErr := t.sendEmailExpiringTokenAlert(tenantAlerting.Tenant, emailGroup, payload); innerErr != nil {
			err = multierror.Append(err, innerErr)
		}
	}

	return nil
}

func (t *TenantAlertManager) SendTenantResourceLimitAlert(tenantId uuid.UUID, alert *sqlcv1.TenantResourceLimitAlert) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// read in the tenant alerting settings and determine if we should alert
	tenantAlerting, err := t.repo.TenantAlertingSettings().GetTenantAlertingSettings(ctx, tenantId)

	if err != nil {
		return err
	}

	percentage := int(float64(alert.Value) / float64(alert.Limit) * 100)

	state, err := t.repo.TenantAlertingSettings().GetTenantResourceLimitState(ctx, tenantId, string(alert.Resource))

	if err != nil {
		return err
	}

	lastRefillAgo := timediff.TimeDiff(state.LastRefill.Time)

	window := ""

	if state.Window.Valid {
		switch state.Window.String {
		case "24h0m0s":
			window = "daily"
		default:
			window = state.Window.String

		}
	}

	payload := &alerttypes.ResourceLimitAlert{
		Link:          fmt.Sprintf("%s/tenants/%s/tenant-settings/billing-and-limits", t.serverURL, tenantId),
		Resource:      string(alert.Resource),
		AlertType:     string(alert.AlertType),
		CurrentValue:  int(alert.Value),
		LimitValue:    int(alert.Limit),
		Percentage:    percentage,
		LimitWindow:   window,
		LastRefillAgo: lastRefillAgo,
	}

	return t.sendTenantResourceLimitAlert(ctx, tenantAlerting, payload)
}

func (t *TenantAlertManager) sendTenantResourceLimitAlert(ctx context.Context, tenantAlerting *v1.GetTenantAlertingSettingsResponse, payload *alerttypes.ResourceLimitAlert) error {

	if !tenantAlerting.Settings.EnableExpiringTokenAlerts {
		return nil
	}

	var err error

	// iterate through possible alerters
	for _, slackWebhook := range tenantAlerting.SlackWebhooks {
		if innerErr := t.sendSlackTenantResourceLimitAlert(slackWebhook, payload); innerErr != nil {
			err = multierror.Append(err, innerErr)
		}
	}

	for _, emailGroup := range tenantAlerting.EmailGroups {
		if innerErr := t.sendEmailTenantResourceLimitAlert(tenantAlerting.Tenant, emailGroup, payload); innerErr != nil {
			err = multierror.Append(err, innerErr)
		}
	}

	return nil
}
