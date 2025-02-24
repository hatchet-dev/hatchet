package alerting

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/hatchet-dev/hatchet/internal/integrations/alerting/alerttypes"
	"github.com/hatchet-dev/hatchet/internal/integrations/email"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"

	"github.com/hatchet-dev/timediff"
)

type TenantAlertManager struct {
	repo      repository.EngineRepository
	enc       encryption.EncryptionService
	serverURL string
	email     email.EmailService
}

func New(repo repository.EngineRepository, e encryption.EncryptionService, serverURL string, email email.EmailService) *TenantAlertManager {
	return &TenantAlertManager{repo, e, serverURL, email}
}

func (t *TenantAlertManager) HandleAlert(tenantId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// read in the tenant alerting settings and determine if we should alert
	tenantAlerting, err := t.repo.TenantAlertingSettings().GetTenantAlertingSettings(ctx, tenantId)

	if err != nil {
		return err
	}

	lastAlertedAt := tenantAlerting.Settings.LastAlertedAt.Time.UTC()
	maxFrequency, err := time.ParseDuration(tenantAlerting.Settings.MaxFrequency)

	if err != nil {
		return err
	}

	isZero := lastAlertedAt.IsZero()

	if isZero || time.Since(lastAlertedAt) > maxFrequency {
		// update the lastAlertedAt
		now := time.Now().UTC()

		// if we're in the zero state, we don't want to alert since the very beginning of the interval
		if isZero {
			lastAlertedAt = now.Add(-1 * maxFrequency)
		}

		err = t.repo.TenantAlertingSettings().UpdateTenantAlertingSettings(ctx, tenantId, &repository.UpdateTenantAlertingSettingsOpts{
			LastAlertedAt: &now,
		})

		if err != nil {
			return err
		}

		return t.sendWorkflowRunAlert(ctx, tenantAlerting, lastAlertedAt)
	}

	return nil
}

func (t *TenantAlertManager) SendWorkflowRunAlert(tenantId string, prevLastAlertedAt time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// read in the tenant alerting settings and determine if we should alert
	tenantAlerting, err := t.repo.TenantAlertingSettings().GetTenantAlertingSettings(ctx, tenantId)

	if err != nil {
		return err
	}

	return t.sendWorkflowRunAlert(ctx, tenantAlerting, prevLastAlertedAt)
}

func (t *TenantAlertManager) sendWorkflowRunAlert(ctx context.Context, tenantAlerting *repository.GetTenantAlertingSettingsResponse, prevLastAlertedAt time.Time) error {

	if !tenantAlerting.Settings.EnableWorkflowRunFailureAlerts {
		return nil
	}

	// read in all failed workflow runs since the last alerted time, ordered by the most recent runs first
	statuses := []dbsqlc.WorkflowRunStatus{
		dbsqlc.WorkflowRunStatusFAILED,
	}

	limit := 5

	tenantId := sqlchelpers.UUIDToStr(tenantAlerting.Settings.TenantId)

	failedWorkflowRuns, err := t.repo.WorkflowRun().ListWorkflowRuns(
		ctx,
		tenantId,
		&repository.ListWorkflowRunsOpts{
			Statuses:       &statuses,
			Limit:          &limit,
			OrderBy:        repository.StringPtr("createdAt"),
			OrderDirection: repository.StringPtr("DESC"),
			FinishedAfter:  &prevLastAlertedAt,
		},
	)

	if err != nil {
		return err
	}

	if failedWorkflowRuns.Count == 0 {
		return nil
	}

	failedItems := t.getFailedItems(failedWorkflowRuns)

	if len(failedItems) == 0 {
		return nil
	}

	// iterate through possible alerters
	for _, slackWebhook := range tenantAlerting.SlackWebhooks {
		if innerErr := t.sendSlackWorkflowRunAlert(slackWebhook, failedWorkflowRuns.Count, failedItems); innerErr != nil {
			err = multierror.Append(err, innerErr)
		}
	}

	for _, emailGroup := range tenantAlerting.EmailGroups {
		if innerErr := t.sendEmailWorkflowRunAlert(tenantAlerting.Tenant, emailGroup, failedWorkflowRuns.Count, failedItems); innerErr != nil {
			err = multierror.Append(err, innerErr)
		}
	}

	return nil
}

func (t *TenantAlertManager) getFailedItems(failedWorkflowRuns *repository.ListWorkflowRunsResult) []alerttypes.WorkflowRunFailedItem {
	res := make([]alerttypes.WorkflowRunFailedItem, 0)

	for _, workflowRun := range failedWorkflowRuns.Rows {
		workflowRunId := sqlchelpers.UUIDToStr(workflowRun.WorkflowRun.ID)
		tenantId := sqlchelpers.UUIDToStr(workflowRun.WorkflowRun.TenantId)

		readableId := workflowRun.WorkflowRun.DisplayName.String

		if readableId == "" {
			readableId = workflowRun.Workflow.Name
		}

		res = append(res, alerttypes.WorkflowRunFailedItem{
			Link:                  fmt.Sprintf("%s/workflow-runs/%s?tenant=%s", t.serverURL, workflowRunId, tenantId),
			WorkflowName:          workflowRun.Workflow.Name,
			WorkflowRunReadableId: readableId,
			RelativeDate:          timediff.TimeDiff(workflowRun.WorkflowRun.FinishedAt.Time),
			AbsoluteDate:          workflowRun.WorkflowRun.FinishedAt.Time.Format("2006-01-02 15:04:05"),
		})
	}

	return res
}

func (t *TenantAlertManager) SendExpiringTokenAlert(tenantId string, token *dbsqlc.PollExpiringTokensRow) error {
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
		Link:                  fmt.Sprintf("%s/tenant-settings/api-tokens?tenant=%s", t.serverURL, tenantId),
	}

	return t.sendExpiringTokenAlert(ctx, tenantAlerting, payload)
}

func (t *TenantAlertManager) sendExpiringTokenAlert(ctx context.Context, tenantAlerting *repository.GetTenantAlertingSettingsResponse, payload *alerttypes.ExpiringTokenItem) error {

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

func (t *TenantAlertManager) SendTenantResourceLimitAlert(tenantId string, alert *dbsqlc.TenantResourceLimitAlert) error {
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
		Link:          fmt.Sprintf("%s/tenant-settings/billing-and-limits?tenant=%s", t.serverURL, tenantId),
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

func (t *TenantAlertManager) sendTenantResourceLimitAlert(ctx context.Context, tenantAlerting *repository.GetTenantAlertingSettingsResponse, payload *alerttypes.ResourceLimitAlert) error {

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
