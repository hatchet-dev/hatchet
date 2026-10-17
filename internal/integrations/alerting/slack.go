package alerting

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/slack-go/slack"

	"github.com/hatchet-dev/hatchet/internal/integrations/alerting/alerttypes"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// postSlackWebhook decrypts the webhook URL and posts the message. The decrypted
// URL carries a bearer secret in its path, so any error returned from here names
// the webhook by its row ID and never includes the URL.
func (t *TenantAlertManager) postSlackWebhook(slackWebhook *sqlcv1.SlackAppWebhook, msg *slack.WebhookMessage) error {
	whDecrypted, err := t.enc.Decrypt(slackWebhook.WebhookURL, "incoming_webhook_url")

	if err != nil {
		return fmt.Errorf("slack webhook %s: could not decrypt webhook url: %w", slackWebhook.ID, err)
	}

	webhookURL := string(whDecrypted)

	if err := slack.PostWebhook(webhookURL, msg); err != nil {
		return redactSlackWebhookError(slackWebhook.ID, webhookURL, err)
	}

	return nil
}

// redactSlackWebhookError strips the webhook URL from a PostWebhook error. Go's
// *url.Error embeds the full request URL in its message, so it is rebuilt from
// its operation and underlying cause only. slack-go's own status and rate limit
// errors do not contain the URL; the final check guards any other error type.
func redactSlackWebhookError(id uuid.UUID, webhookURL string, err error) error {
	var uerr *url.Error

	if errors.As(err, &uerr) {
		err = fmt.Errorf("%s: %w", uerr.Op, uerr.Err)
	}

	if strings.Contains(err.Error(), webhookURL) {
		return fmt.Errorf("slack webhook %s: request failed", id)
	}

	return fmt.Errorf("slack webhook %s: %w", id, err)
}

func (t *TenantAlertManager) sendSlackWorkflowRunAlert(slackWebhook *sqlcv1.SlackAppWebhook, numFailed int, failedRuns []alerttypes.WorkflowRunFailedItem) error {
	headerText, blocks := t.getSlackWorkflowRunTextAndBlocks(numFailed, failedRuns)

	return t.postSlackWebhook(slackWebhook, &slack.WebhookMessage{
		Text:   headerText,
		Blocks: blocks,
	})
}

func (t *TenantAlertManager) getSlackWorkflowRunTextAndBlocks(numFailed int, failedRuns []alerttypes.WorkflowRunFailedItem) (string, *slack.Blocks) {
	res := make([]slack.Block, 0)

	headerText := fmt.Sprintf("%d Hatchet workflows failed:", numFailed)

	if numFailed <= 1 {
		headerText = fmt.Sprintf("%d Hatchet workflow failed:", numFailed)
	}

	res = append(res, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, headerText, false, false),
		nil,
		nil,
	))

	for i, workflowRun := range failedRuns {
		// don't add more than 5 failed workflow runs
		if i >= 5 {
			break
		}

		buttonAccessory := slack.NewAccessory(
			slack.NewButtonBlockElement(
				"View",
				workflowRun.WorkflowRunReadableId,
				slack.NewTextBlockObject(slack.PlainTextType, "View", true, false),
			),
		)

		buttonAccessory.ButtonElement.URL = workflowRun.Link
		buttonAccessory.ButtonElement.ActionID = "button-action"

		res = append(res, slack.NewSectionBlock(
			slack.NewTextBlockObject(
				slack.MarkdownType,
				fmt.Sprintf(":warning: %s failed %s", workflowRun.WorkflowName, workflowRun.RelativeDate),
				false,
				false,
			),
			nil,
			buttonAccessory,
		))
	}

	return headerText, &slack.Blocks{
		BlockSet: res,
	}
}

func (t *TenantAlertManager) sendSlackExpiringTokenAlert(slackWebhook *sqlcv1.SlackAppWebhook, payload *alerttypes.ExpiringTokenItem) error {
	headerText, blocks := t.getSlackExpiringTokenTextAndBlocks(payload)

	return t.postSlackWebhook(slackWebhook, &slack.WebhookMessage{
		Text:   headerText,
		Blocks: blocks,
	})
}

func (t *TenantAlertManager) getSlackExpiringTokenTextAndBlocks(payload *alerttypes.ExpiringTokenItem) (string, *slack.Blocks) {
	res := make([]slack.Block, 0)

	headerText := fmt.Sprintf(":lock: Heads up! Your `%s` hatchet token will expire `%s`", payload.TokenName, payload.ExpiresAtRelativeDate)

	res = append(res, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, headerText, false, false),
		nil,
		nil,
	))

	buttonAccessory := slack.NewAccessory(
		slack.NewButtonBlockElement(
			"Manage Tokens",
			payload.TokenName,
			slack.NewTextBlockObject(slack.PlainTextType, "Manage Tokens", true, false),
		),
	)

	buttonAccessory.ButtonElement.URL = payload.Link
	buttonAccessory.ButtonElement.ActionID = "button-action"

	res = append(res, slack.NewSectionBlock(
		slack.NewTextBlockObject(
			slack.MarkdownType,
			"Once expired, any workers or clients using this token will no longer be able to connect to Hatchet.",
			false,
			false,
		),
		nil,
		buttonAccessory,
	))

	return headerText, &slack.Blocks{
		BlockSet: res,
	}
}

func (t *TenantAlertManager) sendSlackTenantResourceLimitAlert(slackWebhook *sqlcv1.SlackAppWebhook, payload *alerttypes.ResourceLimitAlert) error {
	headerText, blocks := t.getSlackTenantResourceLimitTextAndBlocks(payload)

	return t.postSlackWebhook(slackWebhook, &slack.WebhookMessage{
		Text:   headerText,
		Blocks: blocks,
	})
}

func (t *TenantAlertManager) getSlackTenantResourceLimitTextAndBlocks(payload *alerttypes.ResourceLimitAlert) (string, *slack.Blocks) {
	res := make([]slack.Block, 0)

	var headerText string

	if payload.AlertType == string(sqlcv1.TenantResourceLimitAlertTypeAlarm) {
		headerText = fmt.Sprintf(":warning: Limit Alarm! `%s` resource is at %d%% of its limit (%d/%d)", payload.Resource, payload.Percentage, payload.CurrentValue, payload.LimitValue)
	}

	if payload.AlertType == string(sqlcv1.TenantResourceLimitAlertTypeExhausted) {
		headerText = fmt.Sprintf(":no_entry: Limit Exhausted! `%s` resource is at 100%% of its limit (%d/%d)", payload.Resource, payload.CurrentValue, payload.LimitValue)
	}

	res = append(res, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, headerText, false, false),
		nil,
		nil,
	))

	buttonAccessory := slack.NewAccessory(
		slack.NewButtonBlockElement(
			"View Limits",
			payload.Resource,
			slack.NewTextBlockObject(slack.PlainTextType, "View Limits", true, false),
		),
	)

	buttonAccessory.ButtonElement.URL = payload.Link
	buttonAccessory.ButtonElement.ActionID = "button-action"

	if payload.AlertType == string(sqlcv1.TenantResourceLimitAlertTypeExhausted) {
		res = append(res, slack.NewSectionBlock(
			slack.NewTextBlockObject(
				slack.MarkdownType,
				"Any further resource usage will be denied until the limit is increased.",
				false,
				false,
			),
			nil,
			nil,
		))

	}

	res = append(res, slack.NewSectionBlock(
		slack.NewTextBlockObject(
			slack.MarkdownType,
			"Please review your resource usage and consider upgrading your plan.",
			false,
			false,
		),
		nil,
		buttonAccessory,
	))

	return headerText, &slack.Blocks{
		BlockSet: res,
	}
}
